/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func NewMonitorAction() Action {
	return &monitorAction{}
}

type monitorAction struct {
	baseAction
}

func (action *monitorAction) Name() string {
	return "monitor"
}

func (action *monitorAction) CanHandle(integration *v1.Integration) bool {
	return integration.Status.Phase == v1.IntegrationPhaseDeploying ||
		integration.Status.Phase == v1.IntegrationPhaseRunning ||
		integration.Status.Phase == v1.IntegrationPhaseError
}

func (action *monitorAction) Handle(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {
	// At that staged the Integration must have a Kit
	if integration.Status.IntegrationKit == nil {
		return nil, fmt.Errorf("no kit set on integration %s", integration.Name)
	}

	// Check if the Integration requires a rebuild
	hash, err := digest.ComputeForIntegration(integration)
	if err != nil {
		return nil, err
	}

	if hash != integration.Status.Digest {
		action.L.Info("Integration needs a rebuild")

		integration.Initialize()
		integration.Status.Digest = hash

		return integration, nil
	}

	kit, err := kubernetes.GetIntegrationKit(ctx, action.client, integration.Status.IntegrationKit.Name, integration.Status.IntegrationKit.Namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to find integration kit %s/%s: %w", integration.Status.IntegrationKit.Namespace, integration.Status.IntegrationKit.Name, err)
	}

	// Check if an IntegrationKit with higher priority is ready
	priority, ok := kit.Labels[v1.IntegrationKitPriorityLabel]
	if !ok {
		priority = "0"
	}
	withHigherPriority, err := labels.NewRequirement(v1.IntegrationKitPriorityLabel, selection.GreaterThan, []string{priority})
	if err != nil {
		return nil, err
	}
	kits, err := lookupKitsForIntegration(ctx, action.client, integration, ctrl.MatchingLabelsSelector{
		Selector: labels.NewSelector().Add(*withHigherPriority),
	})
	if err != nil {
		return nil, err
	}
	priorityReadyKit, err := findHighestPriorityReadyKit(kits)
	if err != nil {
		return nil, err
	}
	if priorityReadyKit != nil {
		integration.SetIntegrationKit(priorityReadyKit)
	}

	// Run traits that are enabled for the phase
	environment, err := trait.Apply(ctx, action.client, integration, kit)
	if err != nil {
		return nil, err
	}

	// Enforce the scale sub-resource label selector.
	// It is used by the HPA that queries the scale sub-resource endpoint,
	// to list the pods owned by the integration.
	integration.Status.Selector = v1.IntegrationLabel + "=" + integration.Name

	// Update the replicas count
	pendingPods := &corev1.PodList{}
	err = action.client.List(ctx, pendingPods,
		ctrl.InNamespace(integration.Namespace),
		ctrl.MatchingLabels{v1.IntegrationLabel: integration.Name},
		ctrl.MatchingFields{"status.phase": string(corev1.PodPending)})
	if err != nil {
		return nil, err
	}
	runningPods := &corev1.PodList{}
	err = action.client.List(ctx, runningPods,
		ctrl.InNamespace(integration.Namespace),
		ctrl.MatchingLabels{v1.IntegrationLabel: integration.Name},
		ctrl.MatchingFields{"status.phase": string(corev1.PodRunning)})
	if err != nil {
		return nil, err
	}
	nonTerminatingPods := 0
	for _, pod := range runningPods.Items {
		if pod.DeletionTimestamp != nil {
			continue
		}
		nonTerminatingPods++
	}
	podCount := int32(len(pendingPods.Items) + nonTerminatingPods)
	integration.Status.Replicas = &podCount

	// Reconcile Integration phase and ready condition
	if integration.Status.Phase == v1.IntegrationPhaseDeploying {
		integration.Status.Phase = v1.IntegrationPhaseRunning
	}
	err = action.updateIntegrationPhaseAndReadyCondition(ctx, environment, integration, pendingPods.Items, runningPods.Items)
	if err != nil {
		return nil, err
	}

	return integration, nil
}

func (action *monitorAction) updateIntegrationPhaseAndReadyCondition(ctx context.Context, environment *trait.Environment, integration *v1.Integration, pendingPods []corev1.Pod, runningPods []corev1.Pod) error {
	var controller ctrl.Object
	var lastCompletedJob *batchv1.Job

	switch {
	case isConditionTrue(integration, v1.IntegrationConditionDeploymentAvailable):
		controller = &appsv1.Deployment{}
	case isConditionTrue(integration, v1.IntegrationConditionKnativeServiceAvailable):
		controller = &servingv1.Service{}
	case isConditionTrue(integration, v1.IntegrationConditionCronJobAvailable):
		controller = &batchv1beta1.CronJob{}
	default:
		return fmt.Errorf("unsupported controller for integration %s", integration.Name)
	}

	// Retrieve the controller updated from the deployer trait execution
	controller = environment.Resources.GetController(func(object ctrl.Object) bool {
		return reflect.TypeOf(controller) == reflect.TypeOf(object)
	})
	if controller == nil {
		return fmt.Errorf("unable to retrieve controller for integration %s", integration.Name)
	}

	switch c := controller.(type) {
	case *appsv1.Deployment:
		// Check the Deployment progression
		if progressing := kubernetes.GetDeploymentCondition(*c, appsv1.DeploymentProgressing); progressing != nil && progressing.Status == corev1.ConditionFalse && progressing.Reason == "ProgressDeadlineExceeded" {
			integration.Status.Phase = v1.IntegrationPhaseError
			setReadyConditionError(integration, progressing.Message)
			return nil
		}

	case *servingv1.Service:
		// Check the KnativeService conditions
		if ready := kubernetes.GetKnativeServiceCondition(*c, servingv1.ServiceConditionReady); ready.IsFalse() && ready.GetReason() == "RevisionFailed" {
			integration.Status.Phase = v1.IntegrationPhaseError
			setReadyConditionError(integration, ready.Message)
			return nil
		}

	case *batchv1beta1.CronJob:
		// Check latest job result
		if lastScheduleTime := c.Status.LastScheduleTime; lastScheduleTime != nil && len(c.Status.Active) == 0 {
			jobs := batchv1.JobList{}
			if err := action.client.List(ctx, &jobs,
				ctrl.InNamespace(integration.Namespace),
				ctrl.MatchingLabels{v1.IntegrationLabel: integration.Name},
			); err != nil {
				return err
			}
			t := lastScheduleTime.Time
			for i, job := range jobs.Items {
				if job.Status.Active == 0 && job.CreationTimestamp.Time.Before(t) {
					continue
				}
				lastCompletedJob = &jobs.Items[i]
				t = lastCompletedJob.CreationTimestamp.Time
			}
			if lastCompletedJob != nil {
				if failed := kubernetes.GetJobCondition(*lastCompletedJob, batchv1.JobFailed); failed != nil && failed.Status == corev1.ConditionTrue {
					setReadyCondition(integration, corev1.ConditionFalse, v1.IntegrationConditionLastJobFailedReason, fmt.Sprintf("last job %s failed: %s", lastCompletedJob.Name, failed.Message))
					integration.Status.Phase = v1.IntegrationPhaseError
					return nil
				}
			}
		}
	}

	// Check Pods statuses
	for _, pod := range pendingPods {
		// Check the scheduled condition
		if scheduled := kubernetes.GetPodCondition(pod, corev1.PodScheduled); scheduled != nil && scheduled.Status == corev1.ConditionFalse && scheduled.Reason == "Unschedulable" {
			integration.Status.Phase = v1.IntegrationPhaseError
			setReadyConditionError(integration, scheduled.Message)
			return nil
		}
	}
	// Check pending container statuses
	for _, pod := range pendingPods {
		var containers []corev1.ContainerStatus
		containers = append(containers, pod.Status.InitContainerStatuses...)
		containers = append(containers, pod.Status.ContainerStatuses...)
		for _, container := range containers {
			// Check the images are pulled
			if waiting := container.State.Waiting; waiting != nil && waiting.Reason == "ImagePullBackOff" {
				integration.Status.Phase = v1.IntegrationPhaseError
				setReadyConditionError(integration, waiting.Message)
				return nil
			}
		}
	}
	// Check running container statuses
	for _, pod := range runningPods {
		if pod.DeletionTimestamp != nil {
			continue
		}
		var containers []corev1.ContainerStatus
		containers = append(containers, pod.Status.InitContainerStatuses...)
		containers = append(containers, pod.Status.ContainerStatuses...)
		for _, container := range containers {
			// Check the container state
			if waiting := container.State.Waiting; waiting != nil && waiting.Reason == "CrashLoopBackOff" {
				integration.Status.Phase = v1.IntegrationPhaseError
				setReadyConditionError(integration, waiting.Message)
				return nil
			}
			if terminated := container.State.Terminated; terminated != nil && terminated.Reason == "Error" {
				integration.Status.Phase = v1.IntegrationPhaseError
				setReadyConditionError(integration, terminated.Message)
				return nil
			}
		}
	}

	integration.Status.Phase = v1.IntegrationPhaseRunning

	switch c := controller.(type) {
	case *appsv1.Deployment:
		replicas := int32(1)
		if r := integration.Spec.Replicas; r != nil {
			replicas = *r
		}
		// The Deployment status reports updated and ready replicas separately,
		// so that the number of ready replicas also accounts for older versions.
		// We compare the Integration PodSpec to that of the Pod in order to make
		// sure we account for up-to-date version.
		var readyPods []corev1.Pod
		for _, pod := range runningPods {
			if ready := kubernetes.GetPodCondition(pod, corev1.PodReady); ready == nil || ready.Status != corev1.ConditionTrue {
				continue
			}
			if equality.Semantic.DeepDerivative(c.Spec.Template.Spec, pod.Spec) {
				readyPods = append(readyPods, pod)
			}
		}
		readyReplicas := int32(len(readyPods))
		// The Integration is considered ready when the number of replicas
		// reported to be ready is larger than or equal to the specified number
		// of replicas. This avoids reporting a falsy readiness condition
		// when the Integration is being down-scaled.
		switch {
		case readyReplicas >= replicas:
			setReadyCondition(integration, corev1.ConditionTrue, v1.IntegrationConditionDeploymentReadyReason, fmt.Sprintf("%d/%d ready replicas", c.Status.ReadyReplicas, replicas))
		case c.Status.UpdatedReplicas < replicas:
			setReadyCondition(integration, corev1.ConditionFalse, v1.IntegrationConditionDeploymentProgressingReason, fmt.Sprintf("%d/%d updated replicas", c.Status.UpdatedReplicas, replicas))
		default:
			setReadyCondition(integration, corev1.ConditionFalse, v1.IntegrationConditionDeploymentProgressingReason, fmt.Sprintf("%d/%d ready replicas", readyReplicas, replicas))
		}

	case *servingv1.Service:
		ready := kubernetes.GetKnativeServiceCondition(*c, servingv1.ServiceConditionReady)
		if ready.IsTrue() {
			setReadyCondition(integration, corev1.ConditionTrue, v1.IntegrationConditionKnativeServiceReadyReason, "")
		} else {
			setReadyCondition(integration, corev1.ConditionFalse, ready.GetReason(), ready.GetMessage())
		}

	case *batchv1beta1.CronJob:
		switch {
		case c.Status.LastScheduleTime == nil:
			setReadyCondition(integration, corev1.ConditionTrue, v1.IntegrationConditionCronJobCreatedReason, "cronjob created")
		case len(c.Status.Active) > 0:
			setReadyCondition(integration, corev1.ConditionTrue, v1.IntegrationConditionCronJobActiveReason, "cronjob active")
		case c.Spec.SuccessfulJobsHistoryLimit != nil && *c.Spec.SuccessfulJobsHistoryLimit == 0 && c.Spec.FailedJobsHistoryLimit != nil && *c.Spec.FailedJobsHistoryLimit == 0:
			setReadyCondition(integration, corev1.ConditionTrue, v1.IntegrationConditionCronJobCreatedReason, "no jobs history available")
		case lastCompletedJob != nil:
			if complete := kubernetes.GetJobCondition(*lastCompletedJob, batchv1.JobComplete); complete != nil && complete.Status == corev1.ConditionTrue {
				setReadyCondition(integration, corev1.ConditionTrue, v1.IntegrationConditionLastJobSucceededReason, fmt.Sprintf("last job %s completed successfully", lastCompletedJob.Name))
			}
		default:
			integration.Status.SetCondition(v1.IntegrationConditionReady, corev1.ConditionUnknown, "", "")
		}
	}

	return nil
}

func findHighestPriorityReadyKit(kits []v1.IntegrationKit) (*v1.IntegrationKit, error) {
	if len(kits) == 0 {
		return nil, nil
	}
	var kit *v1.IntegrationKit
	priority := 0
	for i, k := range kits {
		if k.Status.Phase != v1.IntegrationKitPhaseReady {
			continue
		}
		p, err := strconv.Atoi(k.Labels[v1.IntegrationKitPriorityLabel])
		if err != nil {
			return nil, err
		}
		if p > priority {
			kit = &kits[i]
			priority = p
		}
	}
	return kit, nil
}

func isConditionTrue(integration *v1.Integration, conditionType v1.IntegrationConditionType) bool {
	cond := integration.Status.GetCondition(conditionType)
	if cond == nil {
		return false
	}
	return cond.Status == corev1.ConditionTrue
}

func setReadyConditionError(integration *v1.Integration, err string) {
	setReadyCondition(integration, corev1.ConditionFalse, v1.IntegrationConditionErrorReason, err)
}

func setReadyCondition(integration *v1.Integration, status corev1.ConditionStatus, reason string, message string) {
	integration.Status.SetCondition(v1.IntegrationConditionReady, status, reason, message)
}

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
	"errors"
	"fmt"
	"reflect"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

// The key used for propagating error details from Camel health to MicroProfile Health (See CAMEL-17138).
const runtimeHealthCheckErrorMessage = "error.message"

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

type controller interface {
	checkReadyCondition() (bool, error)
	getPodSpec() corev1.PodSpec
	updateReadyCondition(readyPods []corev1.Pod) bool
}

func (action *monitorAction) newController(ctx context.Context, env *trait.Environment, integration *v1.Integration) (controller, error) {
	var controller controller
	var obj ctrl.Object
	switch {
	case isConditionTrue(integration, v1.IntegrationConditionDeploymentAvailable):
		obj = getUpdatedController(env, &appsv1.Deployment{})
		controller = &deploymentController{
			obj:         obj.(*appsv1.Deployment),
			integration: integration,
		}
	case isConditionTrue(integration, v1.IntegrationConditionKnativeServiceAvailable):
		obj = getUpdatedController(env, &servingv1.Service{})
		controller = &knativeServiceController{
			obj:         obj.(*servingv1.Service),
			integration: integration,
		}
	case isConditionTrue(integration, v1.IntegrationConditionCronJobAvailable):
		obj = getUpdatedController(env, &batchv1.CronJob{})
		controller = &cronJobController{
			obj:         obj.(*batchv1.CronJob),
			integration: integration,
			client:      action.client,
			context:     ctx,
		}
	default:
		return nil, fmt.Errorf("unsupported controller for integration %s", integration.Name)
	}

	if obj == nil {
		return nil, fmt.Errorf("unable to retrieve controller for integration %s", integration.Name)
	}

	return controller, nil
}

// getUpdatedController retrieves the controller updated from the deployer trait execution.
func getUpdatedController(env *trait.Environment, obj ctrl.Object) ctrl.Object {
	return env.Resources.GetController(func(object ctrl.Object) bool {
		return reflect.TypeOf(obj) == reflect.TypeOf(object)
	})
}

func (action *monitorAction) updateIntegrationPhaseAndReadyCondition(ctx context.Context, environment *trait.Environment, integration *v1.Integration, pendingPods []corev1.Pod, runningPods []corev1.Pod) error {
	controller, err := action.newController(ctx, environment, integration)
	if err != nil {
		return err
	}

	if done, err := controller.checkReadyCondition(); done || err != nil {
		return err
	}
	if done := checkPodStatuses(integration, pendingPods, runningPods); done {
		return nil
	}
	integration.Status.Phase = v1.IntegrationPhaseRunning

	readyPods, unreadyPods := filterPodsByReadyStatus(runningPods, controller.getPodSpec())
	if done := controller.updateReadyCondition(readyPods); done {
		return nil
	}
	if err := action.probeReadiness(ctx, environment, integration, unreadyPods); err != nil {
		return err
	}

	return nil
}

func checkPodStatuses(integration *v1.Integration, pendingPods []corev1.Pod, runningPods []corev1.Pod) bool {
	// Check Pods statuses
	for _, pod := range pendingPods {
		// Check the scheduled condition
		if scheduled := kubernetes.GetPodCondition(pod, corev1.PodScheduled); scheduled != nil && scheduled.Status == corev1.ConditionFalse && scheduled.Reason == "Unschedulable" {
			integration.Status.Phase = v1.IntegrationPhaseError
			setReadyConditionError(integration, scheduled.Message)
			return true
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
				return true
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
				return true
			}
			if terminated := container.State.Terminated; terminated != nil && terminated.Reason == "Error" {
				integration.Status.Phase = v1.IntegrationPhaseError
				setReadyConditionError(integration, terminated.Message)
				return true
			}
		}
	}

	return false
}

func filterPodsByReadyStatus(runningPods []corev1.Pod, podSpec corev1.PodSpec) ([]corev1.Pod, []corev1.Pod) {
	var readyPods []corev1.Pod
	var unreadyPods []corev1.Pod
	for _, pod := range runningPods {
		// We compare the Integration PodSpec to that of the Pod in order to make
		// sure we account for up-to-date version.
		if !equality.Semantic.DeepDerivative(podSpec, pod.Spec) {
			continue
		}
		ready := kubernetes.GetPodCondition(pod, corev1.PodReady)
		if ready == nil {
			continue
		}
		switch ready.Status {
		case corev1.ConditionTrue:
			// We still account terminating Pods to handle rolling deployments
			readyPods = append(readyPods, pod)
		case corev1.ConditionFalse:
			if pod.DeletionTimestamp != nil {
				continue
			}
			unreadyPods = append(unreadyPods, pod)
		}
	}

	return readyPods, unreadyPods
}

// probeReadiness calls the readiness probes of the non-ready Pods directly to retrieve insights from the Camel runtime.
func (action *monitorAction) probeReadiness(ctx context.Context, environment *trait.Environment, integration *v1.Integration, unreadyPods []corev1.Pod) error {
	var runtimeNotReadyMessages []string
	for i := range unreadyPods {
		pod := &unreadyPods[i]
		if ready := kubernetes.GetPodCondition(*pod, corev1.PodReady); ready.Reason != "ContainersNotReady" {
			continue
		}
		container := getIntegrationContainer(environment, pod)
		if container == nil {
			return fmt.Errorf("integration container not found in Pod %s/%s", pod.Namespace, pod.Name)
		}
		if probe := container.ReadinessProbe; probe != nil && probe.HTTPGet != nil {
			body, err := proxyGetHTTPProbe(ctx, action.client, probe, pod, container)
			if err == nil {
				continue
			}
			if errors.Is(err, context.DeadlineExceeded) {
				runtimeNotReadyMessages = append(runtimeNotReadyMessages, fmt.Sprintf("readiness probe timed out for Pod %s/%s", pod.Namespace, pod.Name))
				continue
			}
			if !k8serrors.IsServiceUnavailable(err) {
				runtimeNotReadyMessages = append(runtimeNotReadyMessages, fmt.Sprintf("readiness probe failed for Pod %s/%s: %s", pod.Namespace, pod.Name, err.Error()))
				continue
			}
			health, err := NewHealthCheck(body)
			if err != nil {
				return err
			}
			for _, check := range health.Checks {
				if check.Status == HealthCheckStateUp {
					continue
				}
				if _, ok := check.Data[runtimeHealthCheckErrorMessage]; ok {
					integration.Status.Phase = v1.IntegrationPhaseError
				}
				runtimeNotReadyMessages = append(runtimeNotReadyMessages, fmt.Sprintf("Pod %s runtime is not ready: %s", pod.Name, check.Data))
			}
		}
	}
	if len(runtimeNotReadyMessages) > 0 {
		reason := v1.IntegrationConditionRuntimeNotReadyReason
		if integration.Status.Phase == v1.IntegrationPhaseError {
			reason = v1.IntegrationConditionErrorReason
		}
		setReadyCondition(integration, corev1.ConditionFalse, reason, fmt.Sprintf("%s", runtimeNotReadyMessages))
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

func getIntegrationContainer(environment *trait.Environment, pod *corev1.Pod) *corev1.Container {
	name := environment.GetIntegrationContainerName()
	for i, container := range pod.Spec.Containers {
		if container.Name == name {
			return &pod.Spec.Containers[i]
		}
	}
	return nil
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

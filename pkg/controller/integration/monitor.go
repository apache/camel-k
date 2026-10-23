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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/utils/ptr"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	servingv1 "github.com/apache/camel-k/v2/pkg/apis/duck/knative/serving/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/digest"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	utilResource "github.com/apache/camel-k/v2/pkg/util/resource"
)

// NewMonitorAction is an action used to monitor manager Integrations.
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

//nolint:nestif
func (action *monitorAction) Handle(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {
	// When in InitializationFailed condition a kit is not available for the integration
	// so handle it differently from the rest
	if isInInitializationFailed(integration.Status) {
		// Only check if the Integration requires a rebuild
		return action.checkDigestAndRebuild(ctx, integration, nil)
	}

	var kit *v1.IntegrationKit
	var err error
	if integration.Status.IntegrationKit == nil && integration.Status.Image == "" {
		return nil, fmt.Errorf("no kit nor container image set on integration %s", integration.Name)
	}

	if integration.Status.IntegrationKit != nil {
		// Managed Integration
		kit, err = kubernetes.GetIntegrationKit(ctx, action.client,
			integration.Status.IntegrationKit.Name, integration.Status.IntegrationKit.Namespace)
		if err != nil {
			return nil, fmt.Errorf("unable to find integration kit %s/%s: %w",
				integration.Status.IntegrationKit.Namespace, integration.Status.IntegrationKit.Name, err)
		}

		// If integration is in error and its kit is also in error then integration does not change
		if isInIntegrationKitFailed(integration.Status) &&
			kit.Status.Phase == v1.IntegrationKitPhaseError {
			return nil, nil
		}
	}

	// Check if the Integration requires a rebuild
	if changed, err := action.checkDigestAndRebuild(ctx, integration, kit); err != nil {
		return nil, err
	} else if changed != nil {
		return changed, nil
	}

	if kit != nil {
		// Check if an IntegrationKit with higher priority is ready
		priority, ok := kit.Labels[v1.IntegrationKitPriorityLabel]
		if !ok {
			priority = "0"
		}
		withHigherPriority, err := labels.NewRequirement(v1.IntegrationKitPriorityLabel,
			selection.GreaterThan, []string{priority})
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
	}
	// Run traits that are enabled for the phase
	environment, err := trait.Apply(ctx, action.client, integration, kit)
	if err != nil {
		integration.Status.Phase = v1.IntegrationPhaseError
		integration.SetReadyCondition(corev1.ConditionFalse,
			v1.IntegrationConditionInitializationFailedReason, err.Error())

		return integration, err
	}

	return action.monitorPods(ctx, environment, integration)
}

func (action *monitorAction) monitorPods(ctx context.Context, environment *trait.Environment, integration *v1.Integration) (*v1.Integration, error) {
	controller, err := action.newController(environment, integration)
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
	podCount := len(pendingPods.Items) + nonTerminatingPods
	replicas, err := util.IToInt32(podCount)
	if err != nil {
		return nil, err
	}
	integration.Status.Replicas = replicas

	// Reconcile Integration phase and ready condition
	if integration.Status.Phase == v1.IntegrationPhaseDeploying {
		integration.Status.Phase = v1.IntegrationPhaseRunning
	}

	controller.setReadyCondition(integration)

	return integration, nil
}

func isInInitializationFailed(status v1.IntegrationStatus) bool {
	if status.Phase != v1.IntegrationPhaseError {
		return false
	}
	if cond := status.GetCondition(v1.IntegrationConditionReady); cond != nil {
		if cond.Status == corev1.ConditionFalse &&
			cond.Reason == v1.IntegrationConditionInitializationFailedReason {
			return true
		}
	}

	return false
}

func isInIntegrationKitFailed(status v1.IntegrationStatus) bool {
	if cond := status.GetCondition(v1.IntegrationConditionKitAvailable); cond != nil {
		if cond.Status == corev1.ConditionFalse &&
			status.Phase != v1.IntegrationPhaseError {
			return true
		}
	}

	return false
}

func (action *monitorAction) checkDigestAndRebuild(ctx context.Context, integration *v1.Integration, kit *v1.IntegrationKit) (*v1.Integration, error) {
	secrets, configmaps := getIntegrationSecretAndConfigmapResourceVersions(ctx, action.client, integration)
	hash, err := digest.ComputeForIntegration(integration, configmaps, secrets)
	if err != nil {
		return nil, err
	}

	if hash != integration.Status.Digest {
		action.L.Infof("Integration %s digest has changed: resetting its status. Will check if it needs to be rebuilt and restarted.", integration.Name)
		if isIntegrationKitResetRequired(integration, kit) {
			integration.SetIntegrationKit(nil)
		}
		integration.Initialize()
		integration.Status.Digest = hash

		return integration, nil
	}

	return nil, nil
}

func isIntegrationKitResetRequired(integration *v1.Integration, kit *v1.IntegrationKit) bool {
	if kit == nil {
		return false
	}

	if v1.GetOperatorIDAnnotation(integration) != "" &&
		v1.GetOperatorIDAnnotation(integration) != v1.GetOperatorIDAnnotation(kit) {
		// Operator to reconcile the integration has changed. Reset integration kit
		// so new operator can handle the kit reference
		return true
	}

	if kubernetes.GetIntegrationProfileResource(integration) != "" &&
		kubernetes.GetIntegrationProfileResource(integration) != kubernetes.GetIntegrationProfileResource(kit) {
		// Integration profile for the integration has changed. Reset integration kit
		// so new profile can be applied
		return true
	}

	return false
}

// getIntegrationSecretAndConfigmapResourceVersions returns the list of resource versions only useful to watch for changes.
func getIntegrationSecretAndConfigmapResourceVersions(ctx context.Context, client client.Client, integration *v1.Integration) ([]string, []string) {
	configmaps := make([]string, 0)
	secrets := make([]string, 0)
	if integration.Status.Traits != nil && integration.Status.Traits.Mount != nil && ptr.Deref(integration.Status.Traits.Mount.HotReload, false) {
		mergedResources := make([]string, 0, len(integration.Status.Traits.Mount.Configs)+len(integration.Status.Traits.Mount.Resources))
		mergedResources = append(mergedResources, integration.Status.Traits.Mount.Configs...)
		mergedResources = append(mergedResources, integration.Status.Traits.Mount.Resources...)
		for _, c := range mergedResources {
			if conf, parseErr := utilResource.ParseConfig(c); parseErr == nil {
				if conf.StorageType() == utilResource.StorageTypeConfigmap {
					cm := corev1.ConfigMap{
						Kind:       corev1.ResourceConfigMaps.String(),
						APIVersion: corev1.SchemeGroupVersion.String(),
						Namespace:  integration.Namespace,
						Name:       conf.Name(),
					}
					configmaps = append(configmaps, kubernetes.LookupResourceVersion(ctx, client, &cm))
				} else if conf.StorageType() == utilResource.StorageTypeSecret {
					sec := corev1.Secret{
						Kind:       corev1.ResourceSecrets.String(),
						APIVersion: corev1.SchemeGroupVersion.String(),
						Namespace:  integration.Namespace,
						Name:       conf.Name(),
					}
					secrets = append(secrets, kubernetes.LookupResourceVersion(ctx, client, &sec))
				}
			}
		}
	}

	return secrets, configmaps
}

type controller interface {
	// in charge to set the readiness condition, specific to each controller.
	setReadyCondition(integration *v1.Integration)
}

func (action *monitorAction) newController(env *trait.Environment, integration *v1.Integration) (controller, error) {
	var controller controller
	var obj ctrl.Object
	switch {
	case integration.IsConditionTrue(v1.IntegrationConditionDeploymentAvailable):
		obj = getUpdatedController(env, &appsv1.Deployment{})
		deploy, ok := obj.(*appsv1.Deployment)
		if !ok {
			return nil, fmt.Errorf("type assertion failed, not a Deployment: %v", obj)
		}
		controller = &deploymentController{
			obj: deploy,
		}
	case integration.IsConditionTrue(v1.IntegrationConditionKnativeServiceAvailable):
		obj = getUpdatedController(env, &servingv1.Service{})
		svc, ok := obj.(*servingv1.Service)
		if !ok {
			return nil, fmt.Errorf("type assertion failed, not a Knative Service: %v", obj)
		}
		controller = &knativeServiceController{
			obj: svc,
		}
	case integration.IsConditionTrue(v1.IntegrationConditionCronJobAvailable):
		obj = getUpdatedController(env, &batchv1.CronJob{})
		cj, ok := obj.(*batchv1.CronJob)
		if !ok {
			return nil, fmt.Errorf("type assertion failed, not a CronJob: %v", obj)
		}
		controller = &cronJobController{
			obj: cj,
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

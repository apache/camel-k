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

package trait

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1"
)

// The Deployment trait is responsible for generating the Kubernetes deployment that will make sure
// the integration will run in the cluster.
//
// +camel-k:trait=deployment
type deploymentTrait struct {
	BaseTrait `property:",squash"`
	deployer  deployerTrait
}

func newDeploymentTrait() *deploymentTrait {
	return &deploymentTrait{
		BaseTrait: newBaseTrait("deployment"),
	}
}

func (t *deploymentTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionDeploymentAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionDeploymentAvailableReason,
			"explicitly disabled",
		)

		return false, nil
	}

	if e.IntegrationInPhase(v1.IntegrationPhaseRunning) {
		condition := e.Integration.Status.GetCondition(v1.IntegrationConditionDeploymentAvailable)
		return condition != nil && condition.Status == corev1.ConditionTrue, nil
	}

	enabled := false

	if e.IntegrationInPhase(v1.IntegrationPhaseDeploying) {
		//
		// Don't deploy when a different strategy is needed (e.g. Knative)
		//
		strategy, err := e.DetermineControllerStrategy(t.ctx, t.client)
		if err != nil {
			e.Integration.Status.SetErrorCondition(
				v1.IntegrationConditionDeploymentAvailable,
				v1.IntegrationConditionDeploymentAvailableReason,
				err,
			)

			return false, err
		}

		enabled = strategy == ControllerStrategyDeployment
	} else if e.IntegrationKitInPhase(v1.IntegrationKitPhaseReady) &&
		e.IntegrationInPhase(v1.IntegrationPhaseBuildingKit, v1.IntegrationPhaseResolvingKit) {
		enabled = true
	}

	if enabled {
		dt := e.Catalog.GetTrait("deployer")
		if dt != nil {
			t.deployer = *dt.(*deployerTrait)
		}
	}

	return enabled, nil
}

func (t *deploymentTrait) Apply(e *Environment) error {
	if e.IntegrationKitInPhase(v1.IntegrationKitPhaseReady) &&
		e.IntegrationInPhase(v1.IntegrationPhaseBuildingKit, v1.IntegrationPhaseResolvingKit) {
		e.PostProcessors = append(e.PostProcessors, func(environment *Environment) error {
			// trigger integration deploy
			e.Integration.Status.Phase = v1.IntegrationPhaseDeploying
			return nil
		})

		return nil
	}

	if e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseDeploying) ||
		e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseRunning) {
		maps := e.ComputeConfigMaps()
		deployment := t.getDeploymentFor(e)

		e.Resources.AddAll(maps)
		e.Resources.Add(deployment)

		e.Integration.Status.SetCondition(
			v1.IntegrationConditionDeploymentAvailable,
			corev1.ConditionTrue,
			v1.IntegrationConditionDeploymentAvailableReason,
			deployment.Name,
		)

		if e.IntegrationInPhase(v1.IntegrationPhaseRunning) {
			// Reconcile the deployment replicas
			replicas := e.Integration.Spec.Replicas
			// Deployment replicas defaults to 1, so we avoid forcing
			// an update to nil that will result to another update cycle
			// back to that default value by the Deployment controller.
			if replicas == nil {
				one := int32(1)
				replicas = &one
			}
			deployment.Spec.Replicas = replicas
		}
	}

	return nil
}

// IsPlatformTrait overrides base class method
func (t *deploymentTrait) IsPlatformTrait() bool {
	return true
}

// **********************************
//
// Deployment
//
// **********************************

func (t *deploymentTrait) getDeploymentFor(e *Environment) *appsv1.Deployment {
	// create a copy to avoid sharing the underlying annotation map
	annotations := make(map[string]string)
	if e.Integration.Annotations != nil {
		for k, v := range FilterTransferableAnnotations(e.Integration.Annotations) {
			annotations[k] = v
		}
	}

	// Resolve registry host names when used
	annotations["alpha.image.policy.openshift.io/resolve-names"] = "*"

	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Integration.Name,
			Namespace: e.Integration.Namespace,
			Labels: map[string]string{
				"camel.apache.org/integration": e.Integration.Name,
			},
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: e.Integration.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"camel.apache.org/integration": e.Integration.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"camel.apache.org/integration": e.Integration.Name,
					},
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: e.Integration.Spec.ServiceAccountName,
				},
			},
		},
	}

	return &deployment
}

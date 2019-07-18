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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
			v1alpha1.IntegrationConditionDeploymentAvailable,
			corev1.ConditionFalse,
			v1alpha1.IntegrationConditionDeploymentAvailableReason,
			"explicitly disabled",
		)

		return false, nil
	}

	enabled := false

	if e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		//
		// Don't deploy when a different strategy is needed (e.g. Knative)
		//
		strategy, err := e.DetermineControllerStrategy(t.ctx, t.client)
		if err != nil {
			e.Integration.Status.SetErrorCondition(
				v1alpha1.IntegrationConditionDeploymentAvailable,
				v1alpha1.IntegrationConditionDeploymentAvailableReason,
				err,
			)

			return false, err
		}

		enabled = strategy == ControllerStrategyDeployment
	} else if e.IntegrationKitInPhase(v1alpha1.IntegrationKitPhaseReady) &&
		e.IntegrationInPhase(v1alpha1.IntegrationPhaseBuildingKit, v1alpha1.IntegrationPhaseResolvingKit) {
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
	if e.IntegrationKitInPhase(v1alpha1.IntegrationKitPhaseReady) &&
		e.IntegrationInPhase(v1alpha1.IntegrationPhaseBuildingKit, v1alpha1.IntegrationPhaseResolvingKit) {

		e.PostProcessors = append(e.PostProcessors, func(environment *Environment) error {
			// trigger integration deploy
			e.Integration.Status.Phase = v1alpha1.IntegrationPhaseDeploying
			return nil
		})

		return nil
	}

	if e.InPhase(v1alpha1.IntegrationKitPhaseReady, v1alpha1.IntegrationPhaseDeploying) {
		maps := e.ComputeConfigMaps()
		depl := t.getDeploymentFor(e)

		e.Resources.AddAll(maps)
		e.Resources.Add(depl)

		e.Integration.Status.SetCondition(
			v1alpha1.IntegrationConditionDeploymentAvailable,
			corev1.ConditionTrue,
			v1alpha1.IntegrationConditionDeploymentAvailableReason,
			depl.Name,
		)
	}

	return nil
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

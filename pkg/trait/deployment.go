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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/util/label"
)

type deploymentTrait struct {
	BaseTrait
	traitv1.DeploymentTrait `property:",squash"`
}

var _ ControllerStrategySelector = &deploymentTrait{}

func newDeploymentTrait() Trait {
	return &deploymentTrait{
		BaseTrait: NewBaseTrait("deployment", 1100),
	}
}

func (t *deploymentTrait) Configure(e *Environment) (bool, error) {
	if e.Integration == nil || !pointer.BoolDeref(t.Enabled, true) {
		if e.Integration != nil {
			e.Integration.Status.SetCondition(
				v1.IntegrationConditionDeploymentAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionDeploymentAvailableReason,
				"explicitly disabled",
			)
		}

		return false, nil
	}

	if e.IntegrationInPhase(v1.IntegrationPhaseRunning, v1.IntegrationPhaseError) {
		condition := e.Integration.Status.GetCondition(v1.IntegrationConditionDeploymentAvailable)
		return condition != nil && condition.Status == corev1.ConditionTrue, nil
	}

	// Don't deploy when a different strategy is needed (e.g. Knative, Cron)
	strategy, err := e.DetermineControllerStrategy()
	if err != nil {
		e.Integration.Status.SetErrorCondition(
			v1.IntegrationConditionDeploymentAvailable,
			v1.IntegrationConditionDeploymentAvailableReason,
			err,
		)

		return false, err
	}

	if strategy != ControllerStrategyDeployment {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionDeploymentAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionDeploymentAvailableReason,
			"controller strategy: "+string(strategy),
		)
		return false, nil
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseDeploying), nil
}

func (t *deploymentTrait) SelectControllerStrategy(e *Environment) (*ControllerStrategy, error) {
	if !pointer.BoolDeref(t.Enabled, true) {
		return nil, nil
	}
	deploymentStrategy := ControllerStrategyDeployment
	return &deploymentStrategy, nil
}

func (t *deploymentTrait) ControllerStrategySelectorOrder() int {
	return 10000
}

func (t *deploymentTrait) Apply(e *Environment) error {
	deployment := t.getDeploymentFor(e)
	e.Resources.Add(deployment)

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionDeploymentAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionDeploymentAvailableReason,
		fmt.Sprintf("deployment name is %s", deployment.Name),
	)

	return nil
}

// IsPlatformTrait overrides base class method.
func (t *deploymentTrait) IsPlatformTrait() bool {
	return true
}

func (t *deploymentTrait) getDeploymentFor(e *Environment) *appsv1.Deployment {
	// create a copy to avoid sharing the underlying annotation map
	annotations := make(map[string]string)
	if e.Integration.Annotations != nil {
		for k, v := range filterTransferableAnnotations(e.Integration.Annotations) {
			annotations[k] = v
		}
	}

	deadline := int32(60)
	if t.ProgressDeadlineSeconds != nil {
		deadline = *t.ProgressDeadlineSeconds
	}

	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Integration.Name,
			Namespace: e.Integration.Namespace,
			Labels: map[string]string{
				v1.IntegrationLabel: e.Integration.Name,
			},
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			ProgressDeadlineSeconds: &deadline,
			Replicas:                e.Integration.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					v1.IntegrationLabel: e.Integration.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      label.AddLabels(e.Integration.Name),
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: e.Integration.Spec.ServiceAccountName,
				},
			},
		},
	}

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

	return &deployment
}

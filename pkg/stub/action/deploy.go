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

package action

import (
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"github.com/pkg/errors"
)

type DeployAction struct {
}

func NewDeployAction() *DeployAction {
	return &DeployAction{}
}

func (b *DeployAction) Name() string {
	return "deploy"
}

func (a *DeployAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseDeploying
}

func (a *DeployAction) Handle(integration *v1alpha1.Integration) error {

	deployment := a.getDeploymentFor(integration)
	err := sdk.Create(deployment)
	if err != nil && k8serrors.IsAlreadyExists(err) {
		err = sdk.Update(deployment)
	}

	if err != nil {
		return errors.Wrap(err, "could not create or replace deployment for integration " + integration.Name)
	}

	target := integration.DeepCopy()
	target.Status.Phase = v1alpha1.IntegrationPhaseRunning
	return sdk.Update(target)
}

func (*DeployAction) getDeploymentFor(integration *v1alpha1.Integration) *v1.Deployment {
	controller := true
	blockOwnerDeletion := true
	labels := map[string]string{
		"camel.apache.org/integration": integration.Name,
	}
	deployment := v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: integration.Name,
			Namespace: integration.Namespace,
			Labels: integration.Labels,
			Annotations: integration.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: integration.APIVersion,
					Kind: integration.Kind,
					Name: integration.Name,
					Controller: &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
					UID: integration.UID,
				},
			},
		},
		Spec: v1.DeploymentSpec{
			Replicas: integration.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: integration.Name,
							Image: integration.Status.Image,
						},
					},
				},
			},
		},
	}

	return &deployment
}

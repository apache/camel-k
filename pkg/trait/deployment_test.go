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
	"context"
	"testing"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigureDisabledDeploymentTraitDoesNotSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	deploymentTrait.Enabled = new(bool)

	configured, err := deploymentTrait.Configure(environment)

	assert.Nil(t, err)
	assert.False(t, configured)
	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, "explicitly disabled", conditions[0].Message)
}

func TestConfigureDeploymentTraitWhileIntegrationIsRuningDoesSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Status.SetCondition(
		v1alpha1.IntegrationConditionDeploymentAvailable,
		corev1.ConditionTrue,
		v1alpha1.IntegrationConditionDeploymentAvailableReason,
		"deployment-name",
	)
	environment.Integration.Status.Phase = v1alpha1.IntegrationPhaseRunning

	configured, err := deploymentTrait.Configure(environment)

	assert.Nil(t, err)
	assert.True(t, configured)
}

func TestConfigureDeploymentTraitDoesSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()

	configured, err := deploymentTrait.Configure(environment)

	assert.Nil(t, err)
	assert.True(t, configured)
	assert.NotNil(t, deploymentTrait.deployer)
}

func TestConfigureDeploymentTraitWhileBuildingKitDoesSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Status.Phase = v1alpha1.IntegrationPhaseBuildingKit

	configured, err := deploymentTrait.Configure(environment)

	assert.Nil(t, err)
	assert.True(t, configured)
	assert.NotNil(t, deploymentTrait.deployer)
}

func TestConfigureDeploymentTraitWhileWaitingPlatformDoesNotSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Status.Phase = v1alpha1.IntegrationPhaseBuildingKit
	environment.IntegrationKit.Status.Phase = v1alpha1.IntegrationKitPhaseWaitingForPlatform

	configured, err := deploymentTrait.Configure(environment)

	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestApplyDeploymentTraitWhileResolvingKitDoesSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Status.Phase = v1alpha1.IntegrationPhaseResolvingKit

	err := deploymentTrait.Apply(environment)

	assert.Nil(t, err)
	assert.Len(t, environment.PostProcessors, 1)
}

func TestApplyDeploymentTraitWhileDeployingIntegrationDoesSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()

	err := deploymentTrait.Apply(environment)

	assert.Nil(t, err)

	configMap := environment.Resources.GetConfigMap(func(cm *corev1.ConfigMap) bool { return true })
	assert.NotNil(t, configMap)

	deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool { return true })
	assert.NotNil(t, deployment)
	assert.Equal(t, "integration-name", deployment.Name)

	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, v1alpha1.IntegrationConditionDeploymentAvailable, conditions[0].Type)
	assert.Equal(t, "integration-name", conditions[0].Message)
}

func TestApplyDeploymentTraitWhileRunningIntegrationDoesSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Status.Phase = v1alpha1.IntegrationPhaseRunning

	err := deploymentTrait.Apply(environment)

	assert.Nil(t, err)

	deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool { return true })
	assert.NotNil(t, deployment)
	assert.Equal(t, "integration-name", deployment.Name)
	assert.Equal(t, int32(3), *deployment.Spec.Replicas)
}

func createNominalDeploymentTest() (*deploymentTrait, *Environment) {
	trait := newDeploymentTrait()
	enabled := true
	trait.Enabled = &enabled
	trait.client, _ = test.NewFakeClient(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "integration-name",
			Namespace: "namespace",
			Labels: map[string]string{
				"camel.apache.org/integration": "integration-name",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: new(int32),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/mount/path",
								},
							},
						},
					},
				},
			},
		},
	})

	replicas := int32(3)

	environment := &Environment{
		Catalog: NewCatalog(context.TODO(), nil),
		Integration: &v1alpha1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "integration-name",
			},
			Spec: v1alpha1.IntegrationSpec{
				Replicas: &replicas,
			},
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
		},
		IntegrationKit: &v1alpha1.IntegrationKit{
			Status: v1alpha1.IntegrationKitStatus{
				Phase: v1alpha1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
			},
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: v1alpha1.IntegrationPlatformClusterOpenShift,
				Profile: v1alpha1.TraitProfileKnative,
			},
		},
		Resources: kubernetes.NewCollection(),
	}

	return trait, environment
}

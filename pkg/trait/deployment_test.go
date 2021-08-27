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
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigureDisabledDeploymentTraitDoesNotSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	deploymentTrait.Enabled = BoolP(false)

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
		v1.IntegrationConditionDeploymentAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionDeploymentAvailableReason,
		"deployment-name",
	)
	environment.Integration.Status.Phase = v1.IntegrationPhaseRunning

	configured, err := deploymentTrait.Configure(environment)

	assert.Nil(t, err)
	assert.True(t, configured)
}

func TestConfigureDeploymentTraitDoesSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()

	configured, err := deploymentTrait.Configure(environment)

	assert.Nil(t, err)
	assert.True(t, configured)
}

func TestConfigureDeploymentTraitWhileBuildingKitDoesNotSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit

	configured, err := deploymentTrait.Configure(environment)

	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestConfigureDeploymentTraitWhileWaitingPlatformDoesNotSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit
	environment.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseWaitingForPlatform

	configured, err := deploymentTrait.Configure(environment)

	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestApplyDeploymentTraitWhileResolvingKitDoesNotSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit

	configured, err := deploymentTrait.Configure(environment)

	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestApplyDeploymentTraitWhileDeployingIntegrationDoesSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Spec.Configuration = append(environment.Integration.Spec.Configuration, v1.ConfigurationSpec{
		Type:  "property",
		Value: "a=b",
	})

	err := deploymentTrait.Apply(environment)

	assert.Nil(t, err)

	assert.NotNil(t, environment.Resources.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Labels["camel.apache.org/properties.type"] == "user"
	}))
	assert.Nil(t, environment.Resources.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Labels["camel.apache.org/properties.type"] == "application"
	}))

	deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool { return true })
	assert.NotNil(t, deployment)
	assert.Equal(t, "integration-name", deployment.Name)

	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, v1.IntegrationConditionDeploymentAvailable, conditions[0].Type)
	assert.Equal(t, "deployment name is integration-name", conditions[0].Message)
}

func TestApplyDeploymentTraitWhileRunningIntegrationDoesSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseRunning

	err := deploymentTrait.Apply(environment)

	assert.Nil(t, err)

	deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool { return true })
	assert.NotNil(t, deployment)
	assert.Equal(t, "integration-name", deployment.Name)
	assert.Equal(t, int32(3), *deployment.Spec.Replicas)
}

func createNominalDeploymentTest() (*deploymentTrait, *Environment) {
	trait := newDeploymentTrait().(*deploymentTrait)
	trait.Enabled = BoolP(true)
	trait.Client, _ = test.NewFakeClient(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "integration-name",
			Namespace: "namespace",
			Labels: map[string]string{
				v1.IntegrationLabel: "integration-name",
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
		Catalog: NewCatalog(nil),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "integration-name",
			},
			Spec: v1.IntegrationSpec{
				Replicas: &replicas,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
			},
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Profile: v1.TraitProfileKnative,
			},
		},
		Resources:             kubernetes.NewCollection(),
		ApplicationProperties: make(map[string]string),
	}
	environment.Platform.ResyncStatusFullConfig()

	return trait, environment
}

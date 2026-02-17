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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func TestConfigureDeploymentTraitWhileIntegrationIsRunningDoesSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Status.SetCondition(
		v1.IntegrationConditionDeploymentAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionDeploymentAvailableReason,
		"deployment-name",
	)
	environment.Integration.Status.Phase = v1.IntegrationPhaseRunning

	configured, condition, err := deploymentTrait.Configure(environment)
	assert.True(t, configured)
	require.NoError(t, err)
	assert.Nil(t, condition)
}

func TestConfigureDeploymentTraitDoesSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()

	configured, condition, err := deploymentTrait.Configure(environment)
	assert.True(t, configured)
	require.NoError(t, err)
	assert.Nil(t, condition)
}

func TestConfigureDeploymentTraitWhileBuildingKitDoesNotSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit

	configured, condition, err := deploymentTrait.Configure(environment)
	assert.False(t, configured)
	require.NoError(t, err)
	assert.Nil(t, condition)
}

func TestConfigureDeploymentTraitWhileWaitingPlatformDoesNotSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit
	environment.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseWaitingForPlatform

	configured, condition, err := deploymentTrait.Configure(environment)
	assert.False(t, configured)
	require.NoError(t, err)
	assert.Nil(t, condition)
}

func TestApplyDeploymentTraitWhileResolvingKitDoesNotSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit

	configured, condition, err := deploymentTrait.Configure(environment)
	assert.False(t, configured)
	require.NoError(t, err)
	assert.Nil(t, condition)
}

func TestApplyDeploymentTraitWhileDeployingIntegrationDoesSucceed(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	err := deploymentTrait.Apply(environment)
	require.NoError(t, err)

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

	require.NoError(t, err)

	deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool { return true })
	assert.NotNil(t, deployment)
	assert.Equal(t, "integration-name", deployment.Name)
	assert.Equal(t, int32(3), *deployment.Spec.Replicas)
	assert.Equal(t, int32(60), *deployment.Spec.ProgressDeadlineSeconds)
}

func TestApplyDeploymentTraitWithProgressDeadline(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	progressDeadlineSeconds := int32(120)
	deploymentTrait.ProgressDeadlineSeconds = &progressDeadlineSeconds
	environment.Integration.Status.Phase = v1.IntegrationPhaseRunning

	err := deploymentTrait.Apply(environment)

	require.NoError(t, err)

	deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool { return true })
	assert.NotNil(t, deployment)
	assert.Equal(t, "integration-name", deployment.Name)
	assert.Equal(t, int32(120), *deployment.Spec.ProgressDeadlineSeconds)
}

func TestApplyDeploymentTraitWitRecreateStrategy(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	maxSurge := intstr.FromInt(10)

	deploymentTrait.Strategy = appsv1.RecreateDeploymentStrategyType
	deploymentTrait.RollingUpdateMaxSurge = &maxSurge

	environment.Integration.Status.Phase = v1.IntegrationPhaseRunning

	err := deploymentTrait.Apply(environment)

	require.NoError(t, err)

	deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool { return true })
	assert.NotNil(t, deployment)
	assert.Equal(t, "integration-name", deployment.Name)
	assert.Equal(t, appsv1.RecreateDeploymentStrategyType, deployment.Spec.Strategy.Type)
	assert.Nil(t, deployment.Spec.Strategy.RollingUpdate)
}

func TestApplyDeploymentTraitWitRollingUpdateStrategy(t *testing.T) {

	t.Run("with defaults", func(t *testing.T) {
		deploymentTrait, environment := createNominalDeploymentTest()

		deploymentTrait.Strategy = appsv1.RollingUpdateDeploymentStrategyType
		environment.Integration.Status.Phase = v1.IntegrationPhaseRunning

		err := deploymentTrait.Apply(environment)

		require.NoError(t, err)

		deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool { return true })
		assert.NotNil(t, deployment)
		assert.Equal(t, "integration-name", deployment.Name)
		assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, deployment.Spec.Strategy.Type)
		assert.Nil(t, deployment.Spec.Strategy.RollingUpdate)
	})

	t.Run("with surge", func(t *testing.T) {
		deploymentTrait, environment := createNominalDeploymentTest()

		maxSurge := intstr.FromInt(10)

		deploymentTrait.Strategy = appsv1.RollingUpdateDeploymentStrategyType
		deploymentTrait.RollingUpdateMaxSurge = &maxSurge

		environment.Integration.Status.Phase = v1.IntegrationPhaseRunning

		err := deploymentTrait.Apply(environment)

		require.NoError(t, err)

		deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool { return true })
		assert.NotNil(t, deployment)
		assert.Equal(t, "integration-name", deployment.Name)
		assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, deployment.Spec.Strategy.Type)
		assert.NotNil(t, deployment.Spec.Strategy.RollingUpdate)
		assert.Nil(t, deployment.Spec.Strategy.RollingUpdate.MaxUnavailable)
		assert.Equal(t, maxSurge, *deployment.Spec.Strategy.RollingUpdate.MaxSurge)
	})

	t.Run("with surge and unavailable", func(t *testing.T) {
		deploymentTrait, environment := createNominalDeploymentTest()

		maxSurge := intstr.FromInt(10)
		maxUnavailable := intstr.FromInt(11)

		deploymentTrait.Strategy = appsv1.RollingUpdateDeploymentStrategyType
		deploymentTrait.RollingUpdateMaxSurge = &maxSurge
		deploymentTrait.RollingUpdateMaxUnavailable = &maxUnavailable

		environment.Integration.Status.Phase = v1.IntegrationPhaseRunning

		err := deploymentTrait.Apply(environment)

		require.NoError(t, err)

		deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool { return true })
		assert.NotNil(t, deployment)
		assert.Equal(t, "integration-name", deployment.Name)
		assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, deployment.Spec.Strategy.Type)
		assert.NotNil(t, deployment.Spec.Strategy.RollingUpdate)
		assert.Equal(t, maxUnavailable, *deployment.Spec.Strategy.RollingUpdate.MaxUnavailable)
		assert.Equal(t, maxSurge, *deployment.Spec.Strategy.RollingUpdate.MaxSurge)
	})

	t.Run("with surge and unavailable in percentage values", func(t *testing.T) {
		deploymentTrait, environment := createNominalDeploymentTest()

		maxSurge := intstr.FromString("10%")
		maxUnavailable := intstr.FromString("11%")

		deploymentTrait.Strategy = appsv1.RollingUpdateDeploymentStrategyType
		deploymentTrait.RollingUpdateMaxSurge = &maxSurge
		deploymentTrait.RollingUpdateMaxUnavailable = &maxUnavailable

		environment.Integration.Status.Phase = v1.IntegrationPhaseRunning

		err := deploymentTrait.Apply(environment)

		assert.Nil(t, err)

		deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool { return true })
		assert.NotNil(t, deployment)
		assert.Equal(t, "integration-name", deployment.Name)
		assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, deployment.Spec.Strategy.Type)
		assert.NotNil(t, deployment.Spec.Strategy.RollingUpdate)
		assert.Equal(t, maxUnavailable, *deployment.Spec.Strategy.RollingUpdate.MaxUnavailable)
		assert.Equal(t, maxSurge, *deployment.Spec.Strategy.RollingUpdate.MaxSurge)
	})

}

func createNominalDeploymentTest() (*deploymentTrait, *Environment) {
	trait, _ := newDeploymentTrait().(*deploymentTrait)
	trait.Client, _ = internal.NewFakeClient(&appsv1.Deployment{
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

	// disable the knative service api
	fakeClient := trait.Client.(*internal.FakeClient) //nolint
	fakeClient.DisableAPIGroupDiscovery("serving.knative.dev/v1")

	replicas := int32(3)
	catalog, _ := camel.QuarkusCatalog()

	environment := &Environment{
		CamelCatalog: catalog,
		Catalog:      NewCatalog(nil),
		Client:       trait.Client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "integration-name",
			},
			Spec: v1.IntegrationSpec{
				Replicas: &replicas,
				Traits:   v1.Traits{},
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

func TestApplyDeploymentWithCamelDashboardLabel(t *testing.T) {
	t.Setenv(kubernetes.CamelDashboardAppLabelEnvVar, "my-dashboard-label")

	deploymentTrait, environment := createNominalDeploymentTest()
	err := deploymentTrait.Apply(environment)
	require.NoError(t, err)

	deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool { return true })
	assert.NotNil(t, deployment)
	assert.Equal(t, "integration-name", deployment.Labels["my-dashboard-label"])
}

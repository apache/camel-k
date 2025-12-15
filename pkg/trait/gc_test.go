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
	"strconv"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apache/camel-k/v2/pkg/internal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestConfigureGCTraitDoesSucceed(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	configured, condition, err := gcTrait.Configure(environment)

	assert.True(t, configured)
	require.NoError(t, err)
	assert.Nil(t, condition)

}

func TestConfigureDisabledGCTraitDoesNotSucceed(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	gcTrait.Enabled = ptr.To(false)

	expectedCondition := NewIntegrationCondition(
		"GC",
		v1.IntegrationConditionTraitInfo,
		corev1.ConditionTrue,
		"TraitConfiguration",
		"explicitly disabled by the user",
	)
	configured, condition, err := gcTrait.Configure(environment)
	assert.False(t, configured)
	require.NoError(t, err)
	assert.NotNil(t, condition)
	assert.Equal(t, expectedCondition, condition)
}

func TestApplyGarbageCollectorTraitFirstGenerationDoesSucceed(t *testing.T) {
	gcTrait, environment := createNominalGCTest()

	err := gcTrait.Apply(environment)

	require.NoError(t, err)
	assert.Len(t, environment.PostProcessors, 1)
	assert.Len(t, environment.PostActions, 0)
}

func TestApplyGarbageCollectorTraitNextGenerationDoesSucceed(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	environment.Integration.Generation = 2

	err := gcTrait.Apply(environment)

	require.NoError(t, err)
	assert.Len(t, environment.PostProcessors, 1)
	assert.Len(t, environment.PostActions, 1)
}

func TestApplyGCTraitDuringInitializationPhaseSkipPostActions(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseInitialization

	err := gcTrait.Apply(environment)

	require.NoError(t, err)
	assert.Len(t, environment.PostProcessors, 1)
	assert.Len(t, environment.PostActions, 0)
}

func TestGetDefaultMinimalGarbageCollectableTypes(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	environment.Integration.Generation = 2

	gcTrait.Client, _ = internal.NewFakeClient()
	environment.Client = gcTrait.Client

	deletableTypes, err := gcTrait.getDeletableTypes(environment)

	require.NoError(t, err)
	assert.Len(t, deletableTypes, 6)
}

func TestGarbageCollectResources(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	environment.Integration.Generation = 2

	deployment := getIntegrationDeployment(environment.Integration)
	deployment.Labels[v1.IntegrationGenerationLabel] = "1"
	gcTrait.Client, _ = internal.NewFakeClient(deployment)

	environment.Client = gcTrait.Client

	resourceDeleted := false
	fakeClient := gcTrait.Client.(*internal.FakeClient) //nolint
	fakeClient.Intercept(&interceptor.Funcs{
		Delete: func(ctx context.Context, client ctrl.WithWatch, obj ctrl.Object, opts ...ctrl.DeleteOption) error {
			assert.Equal(t, environment.Integration.Name, obj.GetName())
			assert.Equal(t, "Deployment", obj.GetObjectKind().GroupVersionKind().Kind)
			resourceDeleted = true
			return nil
		},
	})
	err := gcTrait.garbageCollectResources(environment)

	require.NoError(t, err)
	assert.True(t, resourceDeleted)
}

func TestGarbageCollectPreserveResourcesWithSameGeneration(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	environment.Integration.Generation = 2

	deployment := getIntegrationDeployment(environment.Integration)
	gcTrait.Client, _ = internal.NewFakeClient(deployment)

	environment.Client = gcTrait.Client

	resourceDeleted := false
	fakeClient := gcTrait.Client.(*internal.FakeClient)
	fakeClient.Intercept(&interceptor.Funcs{
		Delete: func(ctx context.Context, client ctrl.WithWatch, obj ctrl.Object, opts ...ctrl.DeleteOption) error {
			resourceDeleted = true
			return nil
		},
	})
	err := gcTrait.garbageCollectResources(environment)

	require.NoError(t, err)
	assert.False(t, resourceDeleted)
}

func TestGarbageCollectUndeploying(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildComplete

	// Simulate undeploy scenario: DeploymentTimestamp set means integration was previously deployed
	now := metav1.Now()
	environment.Integration.Status.DeploymentTimestamp = &now

	deployment := getIntegrationDeployment(environment.Integration)
	gcTrait.Client, _ = internal.NewFakeClient(deployment)

	environment.Client = gcTrait.Client

	resourceDeleted := false
	fakeClient := gcTrait.Client.(*internal.FakeClient)
	fakeClient.Intercept(&interceptor.Funcs{
		Delete: func(ctx context.Context, client ctrl.WithWatch, obj ctrl.Object, opts ...ctrl.DeleteOption) error {
			resourceDeleted = true
			return nil
		},
	})
	err := gcTrait.garbageCollectResources(environment)

	require.NoError(t, err)
	assert.True(t, resourceDeleted)
}

func TestGarbageCollectPreserveResourcesOwnerReferenceMismatch(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	environment.Integration.Generation = 2

	deployment := getIntegrationDeployment(environment.Integration)
	deployment.Labels[v1.IntegrationGenerationLabel] = "1"
	deployment.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "Integration",
			Name:       "other-integration-owner",
		},
	}
	gcTrait.Client, _ = internal.NewFakeClient(deployment)

	environment.Client = gcTrait.Client

	resourceDeleted := false
	fakeClient := gcTrait.Client.(*internal.FakeClient) //nolint
	fakeClient.Intercept(&interceptor.Funcs{
		Delete: func(ctx context.Context, client ctrl.WithWatch, obj ctrl.Object, opts ...ctrl.DeleteOption) error {
			resourceDeleted = true
			return nil
		},
	})
	err := gcTrait.garbageCollectResources(environment)

	require.NoError(t, err)
	assert.False(t, resourceDeleted)
}

func TestGarbageCollectKnativeServiceResources(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	environment.Integration.Generation = 2
	environment.Integration.Spec.Profile = v1.TraitProfileKnative

	gcTrait.Client, _ = internal.NewFakeClient(&servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      environment.Integration.Name,
			Namespace: environment.Integration.Namespace,
			Labels: map[string]string{
				v1.IntegrationLabel:           environment.Integration.Name,
				v1.IntegrationGenerationLabel: "1",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "Integration",
					Name:       environment.Integration.Name,
				},
			},
		},
	})

	environment.Client = gcTrait.Client

	resourceDeleted := false
	fakeClient := gcTrait.Client.(*internal.FakeClient) //nolint
	fakeClient.Intercept(&interceptor.Funcs{
		Delete: func(ctx context.Context, client ctrl.WithWatch, obj ctrl.Object, opts ...ctrl.DeleteOption) error {
			assert.Equal(t, environment.Integration.Name, obj.GetName())
			assert.Equal(t, "Service", obj.GetObjectKind().GroupVersionKind().Kind)
			assert.Equal(t, servingv1.SchemeGroupVersion, obj.GetObjectKind().GroupVersionKind().GroupVersion())
			resourceDeleted = true
			return nil
		},
	})
	err := gcTrait.garbageCollectResources(environment)

	require.NoError(t, err)
	assert.True(t, resourceDeleted)
}

func TestGarbageCollectKnativeTriggerResources(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	environment.Integration.Generation = 2
	environment.Integration.Spec.Profile = v1.TraitProfileKnative

	gcTrait.Client, _ = internal.NewFakeClient(&eventingv1.Trigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      environment.Integration.Name,
			Namespace: environment.Integration.Namespace,
			Labels: map[string]string{
				v1.IntegrationLabel:           environment.Integration.Name,
				v1.IntegrationGenerationLabel: "1",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "Integration",
					Name:       environment.Integration.Name,
				},
			},
		},
	})

	environment.Client = gcTrait.Client

	resourceDeleted := false
	fakeClient := gcTrait.Client.(*internal.FakeClient) //nolint
	fakeClient.Intercept(&interceptor.Funcs{
		Delete: func(ctx context.Context, client ctrl.WithWatch, obj ctrl.Object, opts ...ctrl.DeleteOption) error {
			assert.Equal(t, environment.Integration.Name, obj.GetName())
			assert.Equal(t, "Trigger", obj.GetObjectKind().GroupVersionKind().Kind)
			assert.Equal(t, eventingv1.SchemeGroupVersion, obj.GetObjectKind().GroupVersionKind().GroupVersion())
			resourceDeleted = true
			return nil
		},
	})
	err := gcTrait.garbageCollectResources(environment)

	require.NoError(t, err)
	assert.True(t, resourceDeleted)
}

func getIntegrationDeployment(integration *v1.Integration) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      integration.Name,
			Namespace: integration.Namespace,
			Labels: map[string]string{
				v1.IntegrationLabel:           integration.Name,
				v1.IntegrationGenerationLabel: strconv.FormatInt(integration.Generation, 10),
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "Integration",
					Name:       integration.Name,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: new(int32),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
						},
					},
				},
			},
		},
	}
}

func createNominalGCTest() (*gcTrait, *Environment) {
	trait, _ := newGCTrait().(*gcTrait)
	trait.Enabled = ptr.To(true)

	environment := &Environment{
		Catalog: NewCatalog(nil),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "integration-name",
				Namespace:  "namespace",
				Generation: 1,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseRunning,
			},
		},
	}

	return trait, environment
}

func TestCanResourceBeDeleted(t *testing.T) {
	it := &v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-it",
		},
	}

	resNoOwner := unstructured.Unstructured{}
	resNoOwner.SetOwnerReferences(
		[]metav1.OwnerReference{},
	)
	assert.False(t, canBeDeleted(it, resNoOwner))

	resNotThisItOwner := unstructured.Unstructured{}
	resNotThisItOwner.SetOwnerReferences(
		[]metav1.OwnerReference{
			metav1.OwnerReference{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       "Integration",
				Name:       "another-it",
			},
		},
	)
	assert.False(t, canBeDeleted(it, resNotThisItOwner))

	resThisItOwner := unstructured.Unstructured{}
	resThisItOwner.SetOwnerReferences(
		[]metav1.OwnerReference{
			metav1.OwnerReference{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       "Integration",
				Name:       "my-it",
			},
		},
	)
	assert.True(t, canBeDeleted(it, resThisItOwner))
}

func TestHasNeverDeployed(t *testing.T) {
	tests := []struct {
		name                string
		deploymentTimestamp *metav1.Time
		readyCondition      *v1.IntegrationCondition
		expected            bool
	}{
		{
			name:                "never deployed - both checks nil",
			deploymentTimestamp: nil,
			readyCondition:      nil,
			expected:            true,
		},
		{
			name:                "deployed - DeploymentTimestamp set",
			deploymentTimestamp: ptr.To(metav1.Now()),
			readyCondition:      nil,
			expected:            false,
		},
		{
			name:                "deployed - Ready FirstTruthyTime set",
			deploymentTimestamp: nil,
			readyCondition: &v1.IntegrationCondition{
				Type:            v1.IntegrationConditionReady,
				Status:          corev1.ConditionTrue,
				FirstTruthyTime: ptr.To(metav1.Now()),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			integration := &v1.Integration{
				Status: v1.IntegrationStatus{
					DeploymentTimestamp: tt.deploymentTimestamp,
				},
			}
			if tt.readyCondition != nil {
				integration.Status.Conditions = []v1.IntegrationCondition{*tt.readyCondition}
			}

			result := hasNeverDeployed(integration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGarbageCollectDryBuildSkipsGC(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildComplete

	environment.Integration.Status.DeploymentTimestamp = nil
	environment.Integration.Status.Conditions = nil

	deployment := getIntegrationDeployment(environment.Integration)
	gcTrait.Client, _ = internal.NewFakeClient(deployment)

	environment.Client = gcTrait.Client
	resourceDeleted := false
	fakeClient := gcTrait.Client.(*internal.FakeClient)
	fakeClient.Intercept(&interceptor.Funcs{
		Delete: func(ctx context.Context, client ctrl.WithWatch, obj ctrl.Object, opts ...ctrl.DeleteOption) error {
			resourceDeleted = true
			return nil
		},
	})

	err := gcTrait.Apply(environment)

	require.NoError(t, err)
	assert.Len(t, environment.PostActions, 0)
	assert.False(t, resourceDeleted)
}

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

	eventingv1 "github.com/apache/camel-k/v2/pkg/apis/duck/knative/eventing/v1"
	servingv1 "github.com/apache/camel-k/v2/pkg/apis/duck/knative/serving/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
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

func TestGetDeletableTypes(t *testing.T) {
	gcTrait, _ := createNominalGCTest()

	deletableTypes, err := gcTrait.getDeletableTypes()

	require.NoError(t, err)
	// The default minimal set is always included.
	for gvk := range defaultDeletableTypes {
		assert.Contains(t, deletableTypes, gvk)
	}
	// Types the operator is allowed to delete according to the operator roles
	// shipped with the operator are included, including the ones created by
	// other traits.
	assert.Contains(t, deletableTypes, schema.GroupVersionKind{Group: "serving.knative.dev", Version: "v1", Kind: "Service"})
	assert.Contains(t, deletableTypes, schema.GroupVersionKind{Group: "eventing.knative.dev", Version: "v1", Kind: "Trigger"})
	assert.Contains(t, deletableTypes, schema.GroupVersionKind{Group: "keda.sh", Version: "v1alpha1", Kind: "ScaledObject"})
	assert.Contains(t, deletableTypes, schema.GroupVersionKind{Group: "keda.sh", Version: "v1alpha1", Kind: "TriggerAuthentication"})
	assert.Contains(t, deletableTypes, schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "PodMonitor"})
	// A type the operator is not allowed to delete is not included.
	assert.NotContains(t, deletableTypes, schema.GroupVersionKind{Group: "camel.apache.org", Version: "v1", Kind: "CamelCatalog"})
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

func TestGarbageCollectSkipsTypesNotInstalledInCluster(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	environment.Integration.Generation = 2

	deployment := getIntegrationDeployment(environment.Integration)
	deployment.Labels[v1.IntegrationGenerationLabel] = "1"
	gcTrait.Client, _ = internal.NewFakeClient(deployment)

	environment.Client = gcTrait.Client

	resourceDeleted := false
	fakeClient := gcTrait.Client.(*internal.FakeClient) //nolint
	fakeClient.Intercept(&interceptor.Funcs{
		List: func(ctx context.Context, client ctrl.WithWatch, list ctrl.ObjectList, opts ...ctrl.ListOption) error {
			// Simulate the types whose API is not installed in the cluster:
			// only the Deployment type can be listed successfully.
			if u, ok := list.(*unstructured.UnstructuredList); ok && u.GetAPIVersion() != "apps/v1" {
				return &meta.NoKindMatchError{
					GroupKind:        schema.GroupKind{Group: u.GroupVersionKind().Group, Kind: u.GetKind()},
					SearchedVersions: []string{u.GroupVersionKind().Version},
				}
			}
			return client.List(ctx, list, opts...)
		},
		Delete: func(ctx context.Context, client ctrl.WithWatch, obj ctrl.Object, opts ...ctrl.DeleteOption) error {
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
	environment.Integration.Status.Phase = v1.IntegrationPhaseUnDeploying

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

func TestDeletableTypesFromRules(t *testing.T) {
	t.Run("exact match keeps only resources with delete verb and matching rule", func(t *testing.T) {
		types := deletableTypesFromRules(
			[]rbacv1.PolicyRule{
				{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"create", "delete"}},
				{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"delete"}},
			},
		)
		assert.Equal(t,
			map[schema.GroupVersionKind]struct{}{
				{Group: "apps", Version: "v1", Kind: "Deployment"}: {},
				{Group: "", Version: "v1", Kind: "ConfigMap"}:      {},
			},
			types,
		)
	})

	t.Run("wildcard rule matches every mapped resource", func(t *testing.T) {
		types := deletableTypesFromRules(
			[]rbacv1.PolicyRule{
				{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}},
			},
		)
		assert.Len(t, types, len(deletableResourceTypes))
		assert.Contains(t, types, schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"})
		assert.Contains(t, types, schema.GroupVersionKind{Group: "keda.sh", Version: "v1alpha1", Kind: "ScaledObject"})
	})

	t.Run("rule without delete verb is excluded", func(t *testing.T) {
		types := deletableTypesFromRules(
			[]rbacv1.PolicyRule{
				{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"get", "list"}},
			},
		)
		assert.Empty(t, types)
	})

	t.Run("group or resource mismatch is excluded", func(t *testing.T) {
		types := deletableTypesFromRules(
			[]rbacv1.PolicyRule{
				{APIGroups: []string{"apps"}, Resources: []string{"services"}, Verbs: []string{"delete"}},
				{APIGroups: []string{"networking.k8s.io"}, Resources: []string{"deployments"}, Verbs: []string{"delete"}},
			},
		)
		assert.Empty(t, types)
	})

	t.Run("subresource rule does not match the resource itself", func(t *testing.T) {
		types := deletableTypesFromRules(
			[]rbacv1.PolicyRule{
				{APIGroups: []string{""}, Resources: []string{"pods/exec"}, Verbs: []string{"delete"}},
			},
		)
		assert.Empty(t, types)
	})

	t.Run("rule for a resource missing from the mapping is skipped", func(t *testing.T) {
		types := deletableTypesFromRules(
			[]rbacv1.PolicyRule{
				{APIGroups: []string{"apps"}, Resources: []string{"statefulsets"}, Verbs: []string{"delete"}},
			},
		)
		assert.Empty(t, types)
	})
}

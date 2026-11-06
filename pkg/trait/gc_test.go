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
	"slices"
	"strconv"
	"strings"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eventingv1 "github.com/apache/camel-k/v2/pkg/apis/duck/knative/eventing/v1"
	servingv1 "github.com/apache/camel-k/v2/pkg/apis/duck/knative/serving/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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
	// A small stand-in for the types resolved from the operator scheme.
	// Deployments stand in for a resource registered at several versions,
	// as happens in the real scheme: every registered version of a
	// deletable resource must be included.
	testResourceTypes := map[schema.GroupResource][]schema.GroupVersionKind{
		{Group: "apps", Resource: "deployments"}: {
			{Group: "apps", Version: "v1beta1", Kind: "Deployment"},
			{Group: "apps", Version: "v1", Kind: "Deployment"},
		},
		{Group: "", Resource: "configmaps"}:           {{Group: "", Version: "v1", Kind: "ConfigMap"}},
		{Group: "", Resource: "pods"}:                 {{Group: "", Version: "v1", Kind: "Pod"}},
		{Group: "keda.sh", Resource: "scaledobjects"}: {{Group: "keda.sh", Version: "v1alpha1", Kind: "ScaledObject"}},
	}

	t.Run("exact match keeps only resources with delete verb and matching rule", func(t *testing.T) {
		types := deletableTypesFromRules(
			[]rbacv1.PolicyRule{
				{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"create", "delete"}},
				{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"delete"}},
			},
			testResourceTypes,
		)
		assert.Equal(t,
			map[schema.GroupVersionKind]struct{}{
				{Group: "apps", Version: "v1beta1", Kind: "Deployment"}: {},
				{Group: "apps", Version: "v1", Kind: "Deployment"}:      {},
				{Group: "", Version: "v1", Kind: "ConfigMap"}:           {},
			},
			types,
		)
	})

	t.Run("wildcard rule matches every mapped resource", func(t *testing.T) {
		types := deletableTypesFromRules(
			[]rbacv1.PolicyRule{
				{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}},
			},
			testResourceTypes,
		)
		assert.Len(t, types, 5)
		assert.Contains(t, types, schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"})
		assert.Contains(t, types, schema.GroupVersionKind{Group: "apps", Version: "v1beta1", Kind: "Deployment"})
		assert.Contains(t, types, schema.GroupVersionKind{Group: "keda.sh", Version: "v1alpha1", Kind: "ScaledObject"})
	})

	t.Run("rule without delete verb is excluded", func(t *testing.T) {
		types := deletableTypesFromRules(
			[]rbacv1.PolicyRule{
				{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"get", "list"}},
			},
			testResourceTypes,
		)
		assert.Empty(t, types)
	})

	t.Run("group or resource mismatch is excluded", func(t *testing.T) {
		types := deletableTypesFromRules(
			[]rbacv1.PolicyRule{
				{APIGroups: []string{"apps"}, Resources: []string{"services"}, Verbs: []string{"delete"}},
				{APIGroups: []string{"networking.k8s.io"}, Resources: []string{"deployments"}, Verbs: []string{"delete"}},
			},
			testResourceTypes,
		)
		assert.Empty(t, types)
	})

	t.Run("subresource rule does not match the resource itself", func(t *testing.T) {
		types := deletableTypesFromRules(
			[]rbacv1.PolicyRule{
				{APIGroups: []string{""}, Resources: []string{"pods/exec"}, Verbs: []string{"delete"}},
			},
			testResourceTypes,
		)
		assert.Empty(t, types)
	})

	t.Run("rule for a resource missing from the scheme types is skipped", func(t *testing.T) {
		types := deletableTypesFromRules(
			[]rbacv1.PolicyRule{
				{APIGroups: []string{"apps"}, Resources: []string{"statefulsets"}, Verbs: []string{"delete"}},
			},
			testResourceTypes,
		)
		assert.Empty(t, types)
	})
}

func TestOperatorResourceTypes(t *testing.T) {
	resourceTypes, err := operatorResourceTypes()
	require.NoError(t, err)

	// Built-in types resolve: every version registered in the scheme is
	// returned, including the served one.
	assert.Contains(t,
		resourceTypes[schema.GroupResource{Group: "apps", Resource: "deployments"}],
		schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
	)
	// Types registered through the operator scheme resolve with the version
	// they are compiled with, including the duck types: no fixed value is
	// provided by the garbage collection code itself.
	assert.Contains(t,
		resourceTypes[schema.GroupResource{Group: "keda.sh", Resource: "scaledobjects"}],
		schema.GroupVersionKind{Group: "keda.sh", Version: "v1alpha1", Kind: "ScaledObject"},
	)
	assert.Contains(t,
		resourceTypes[schema.GroupResource{Group: "eventing.knative.dev", Resource: "triggers"}],
		schema.GroupVersionKind{Group: "eventing.knative.dev", Version: "v1", Kind: "Trigger"},
	)
	assert.Contains(t,
		resourceTypes[schema.GroupResource{Group: "camel.apache.org", Resource: "integrations"}],
		schema.GroupVersionKind{Group: "camel.apache.org", Version: "v1", Kind: "Integration"},
	)
	// The Gateway kind pluralizes regularly ("gateways"), which the
	// apimachinery name guesser gets wrong.
	assert.Contains(t,
		resourceTypes[schema.GroupResource{Group: "gateway.networking.k8s.io", Resource: "gateways"}],
		schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "Gateway"},
	)
	// List types are not deletable resources.
	_, hasList := resourceTypes[schema.GroupResource{Group: "", Resource: "configmaplists"}]
	assert.False(t, hasList)
	// When the scheme registers several versions of a type, all of them
	// are resolved: HorizontalPodAutoscaler is registered at both v1 and
	// v2, and garbage collection attempts both, skipping the ones the
	// cluster does not serve.
	assert.Contains(t,
		resourceTypes[schema.GroupResource{Group: "autoscaling", Resource: "horizontalpodautoscalers"}],
		schema.GroupVersionKind{Group: "autoscaling", Version: "v1", Kind: "HorizontalPodAutoscaler"},
	)
	assert.Contains(t,
		resourceTypes[schema.GroupResource{Group: "autoscaling", Resource: "horizontalpodautoscalers"}],
		schema.GroupVersionKind{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"},
	)
	// Kinds whose plural is not just a trailing "s" resolve as well.
	assert.Contains(t,
		resourceTypes[schema.GroupResource{Group: "networking.k8s.io", Resource: "networkpolicies"}],
		schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"},
	)
	// The Knative Broker, Channel and InMemoryChannel and the serving Route
	// have no Go type registered in the operator scheme, and nothing else
	// resolves them: the operator does not create those types, so it does
	// not garbage collect them either.
	for _, groupResource := range []schema.GroupResource{
		{Group: "eventing.knative.dev", Resource: "brokers"},
		{Group: "messaging.knative.dev", Resource: "channels"},
		{Group: "messaging.knative.dev", Resource: "inmemorychannels"},
		{Group: "serving.knative.dev", Resource: "routes"},
	} {
		_, ok := resourceTypes[groupResource]
		assert.False(t, ok, "%s unexpectedly resolves to a type", formatGroupResource(groupResource))
	}
}

func TestSchemeResourceTypesAllVersions(t *testing.T) {
	s := runtime.NewScheme()
	// Register the most recent version first: every registered version
	// must be resolved, whatever the registration order.
	require.NoError(t, autoscalingv2.AddToScheme(s))
	require.NoError(t, autoscalingv1.AddToScheme(s))

	resourceTypes := schemeResourceTypes(s)
	// HorizontalPodAutoscaler is registered at both v1 and v2, and both
	// versions are resolved: the cluster, not the operator, decides which
	// of them are served.
	assert.ElementsMatch(t,
		[]schema.GroupVersionKind{
			{Group: "autoscaling", Version: "v1", Kind: "HorizontalPodAutoscaler"},
			{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"},
		},
		resourceTypes[schema.GroupResource{Group: "autoscaling", Resource: "horizontalpodautoscalers"}],
	)
}

func TestKindToResource(t *testing.T) {
	for kind, resource := range map[string]string{
		"ConfigMap":           "configmaps",
		"Deployment":          "deployments",
		"Gateway":             "gateways",
		"GatewayClass":        "gatewayclasses",
		"Ingress":             "ingresses",
		"NetworkPolicy":       "networkpolicies",
		"PodDisruptionBudget": "poddisruptionbudgets",
		"Endpoints":           "endpoints",
		"ScaledObject":        "scaledobjects",
		"InMemoryChannel":     "inmemorychannels",
	} {
		assert.Equal(t, resource, kindToResource(kind), kind)
	}
}

// TestOperatorRolesDeletableCoverage is an exhaustive audit of the shipped
// operator roles: every resource the roles grant the delete verb on must
// resolve, through the same code path loadDeletableTypes uses, to one or
// more deletable types, and every resolved version must be deletable.
// Resolution can silently narrow -- a kind whose plural is computed
// wrongly, or a type dropped from the scheme, simply stops being garbage
// collected -- so this test turns any such loss into a build failure that
// names the resources involved.
func TestOperatorRolesDeletableCoverage(t *testing.T) {
	rules, err := operatorRoleRules()
	require.NoError(t, err)
	resourceTypes, err := operatorResourceTypes()
	require.NoError(t, err)
	deletable := deletableTypesFromRules(rules, resourceTypes)

	// Collect every (group, resource) pair the roles grant the delete verb
	// on, mirroring the verb matching of deletableTypesFromRules.
	granted := make(map[schema.GroupResource]struct{})
	var wildcards []string
	for _, rule := range rules {
		if !slices.Contains(rule.Verbs, "delete") && !slices.Contains(rule.Verbs, "*") {
			continue
		}
		for _, group := range rule.APIGroups {
			for _, resource := range rule.Resources {
				if group == "*" || resource == "*" {
					wildcards = append(wildcards, group+"/"+resource)
					continue
				}
				granted[schema.GroupResource{Group: group, Resource: resource}] = struct{}{}
			}
		}
	}
	// A wildcard grant cannot be enumerated: it would make every known type
	// deletable and this audit would silently stop measuring coverage. None
	// of the shipped roles use one; fail loudly if that changes, so the
	// grant is reviewed deliberately instead.
	require.Empty(t, wildcards, "wildcard delete grants cannot be audited exhaustively: %v", wildcards)

	// Subresources cannot be deleted as a whole, so the runtime never
	// resolves them as types. They are listed explicitly so that skipping
	// them is a deliberate choice: granting delete on another subresource
	// requires updating this list.
	subresources := []schema.GroupResource{
		{Group: "", Resource: "buildconfigs/webhooks"},
		{Group: "", Resource: "imagestreams/secrets"},
		{Group: "build.openshift.io", Resource: "buildconfigs/webhooks"},
		{Group: "image.openshift.io", Resource: "imagestreams/secrets"},
	}

	// The OpenShift role lists some resources under the legacy core group
	// ("") as well as under their real API group, a leftover of the
	// OpenShift 3 API layout. No such resource exists in the core group, so
	// those grants are inert; the same rules grant delete under the real
	// group, and that is the pair that must resolve. The core-group pairs
	// are listed explicitly, and below each of them is also checked to be
	// backed by a resolved grant for the same resource in another group.
	legacyCoreGroup := []schema.GroupResource{
		{Group: "", Resource: "buildconfigs"},
		{Group: "", Resource: "builds"},
		{Group: "", Resource: "imagestreamimages"},
		{Group: "", Resource: "imagestreammappings"},
		{Group: "", Resource: "imagestreams"},
		{Group: "", Resource: "imagestreamtags"},
		{Group: "", Resource: "routes"},
	}

	var foundSubresources, foundLegacy, unresolved []schema.GroupResource
	for groupResource := range granted {
		switch {
		case strings.Contains(groupResource.Resource, "/"):
			foundSubresources = append(foundSubresources, groupResource)
		case slices.Contains(legacyCoreGroup, groupResource):
			foundLegacy = append(foundLegacy, groupResource)
		default:
			GVKs, ok := resourceTypes[groupResource]
			if !ok {
				unresolved = append(unresolved, groupResource)
				continue
			}
			for _, GVK := range GVKs {
				assert.Contains(t, deletable, GVK,
					"delete-granted %s resolves to %s but is not deletable", formatGroupResource(groupResource), GVK)
			}
		}
	}
	assert.Equal(t, subresources, sortedGroupResources(foundSubresources),
		"delete-granted subresources changed: update the deliberate exclusion list in this test")
	assert.Equal(t, legacyCoreGroup, sortedGroupResources(foundLegacy),
		"legacy core-group delete grants changed: update the deliberate exclusion list in this test")

	// Each legacy core-group grant must be backed by a grant for the same
	// resource in its real API group, and that grant must resolve --
	// otherwise the exclusion above would hide a genuinely lost type.
	for _, legacy := range foundLegacy {
		backed := false
		for groupResource := range granted {
			if groupResource.Resource == legacy.Resource && groupResource.Group != legacy.Group {
				_, ok := resourceTypes[groupResource]
				backed = backed || ok
			}
		}
		assert.True(t, backed,
			"legacy core-group grant %s is not backed by a resolved grant in another group", formatGroupResource(legacy))
	}

	require.Empty(t, unresolved,
		"resources granted the delete verb by the operator roles do not resolve to a deletable type: %s",
		formatGroupResources(unresolved))

	// The Knative Broker, Channel and InMemoryChannel and the serving Route
	// have no Go type registered in the operator scheme, and the operator
	// roles no longer grant the delete verb on them: the operator does not
	// create those types, so it neither deletes them nor garbage collects
	// them. Pin their absence here, so reintroducing any of them into the
	// deletable set -- by registering a type, resolving them by other
	// means, or restoring a delete grant -- fails this test naming them.
	for groupResource, GVK := range map[schema.GroupResource]schema.GroupVersionKind{
		{Group: "eventing.knative.dev", Resource: "brokers"}: {
			Group: "eventing.knative.dev", Version: "v1", Kind: "Broker",
		},
		{Group: "messaging.knative.dev", Resource: "channels"}: {
			Group: "messaging.knative.dev", Version: "v1", Kind: "Channel",
		},
		{Group: "messaging.knative.dev", Resource: "inmemorychannels"}: {
			Group: "messaging.knative.dev", Version: "v1", Kind: "InMemoryChannel",
		},
		{Group: "serving.knative.dev", Resource: "routes"}: {
			Group: "serving.knative.dev", Version: "v1", Kind: "Route",
		},
	} {
		assert.NotContains(t, granted, groupResource,
			"%s is granted the delete verb again", formatGroupResource(groupResource))
		assert.NotContains(t, deletable, GVK,
			"%s is deletable again", GVK)
	}
}

func sortedGroupResources(groupResources []schema.GroupResource) []schema.GroupResource {
	slices.SortFunc(groupResources, func(a, b schema.GroupResource) int {
		if c := strings.Compare(a.Group, b.Group); c != 0 {
			return c
		}

		return strings.Compare(a.Resource, b.Resource)
	})

	return groupResources
}

func formatGroupResource(groupResource schema.GroupResource) string {
	return groupResource.Group + "/" + groupResource.Resource
}

func formatGroupResources(groupResources []schema.GroupResource) string {
	names := make([]string, 0, len(groupResources))
	for _, groupResource := range sortedGroupResources(groupResources) {
		names = append(names, formatGroupResource(groupResource))
	}

	return strings.Join(names, ", ")
}

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func TestCollectConfigurationValues(t *testing.T) {
	e := Environment{
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Configuration: []v1.ConfigurationSpec{
					{Type: "configmap", Value: "my-cm-integration"},
					{Type: "env", Value: "my-env-integration"},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Spec: v1.IntegrationKitSpec{
				Configuration: []v1.ConfigurationSpec{
					{Type: "configmap", Value: "my-cm-kit"},
					{Type: "property", Value: "my-p-kit"},
				},
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Configuration: []v1.ConfigurationSpec{
					{Type: "configmap", Value: "my-cm-platform"},
					{Type: "secret", Value: "my-secret-platform"},
					{Type: "property", Value: "my-p-platform"},
					{Type: "env", Value: "my-env-platform"},
				},
			},
		},
	}
	e.Platform.ResyncStatusFullConfig()
}

func TestCollectConfigurationPairs(t *testing.T) {
	e := Environment{
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Configuration: []v1.ConfigurationSpec{
					{Type: "property", Value: "p1=integration"},
					{Type: "property", Value: "p4=integration"},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Spec: v1.IntegrationKitSpec{
				Configuration: []v1.ConfigurationSpec{
					{Type: "property", Value: "p1=kit"},
					{Type: "property", Value: "p2=kit"},
				},
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Configuration: []v1.ConfigurationSpec{
					{Type: "property", Value: "p1=platform"},
					{Type: "property", Value: "p2=platform"},
					{Type: "property", Value: "p3=platform"},
					{Type: "property", Value: "p4=platform"},
				},
			},
		},
	}
	e.Platform.ResyncStatusFullConfig()

	pairs := e.collectConfigurationPairs("property")
	assert.Equal(t, pairs, []variable{
		{Name: "p1", Value: "integration"},
		{Name: "p2", Value: "kit"},
		{Name: "p3", Value: "platform"},
		{Name: "p4", Value: "integration"},
	})
}

func TestVolumeWithKeyAndPath(t *testing.T) {
	v := getVolume("SomeVolName", "secret", "SomeSecretName", "SomeKey", "SomePath")
	assert.NotNil(t, v)
	assert.Equal(t, "SomeVolName", v.Name)
	s := v.VolumeSource.Secret
	assert.NotNil(t, s)
	assert.Equal(t, "SomeSecretName", s.SecretName)
	items := s.Items
	assert.NotNil(t, items)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "SomeKey", items[0].Key)
	assert.Equal(t, "SomePath", items[0].Path)
}

func TestVolumeWithPathOnly(t *testing.T) {
	v := getVolume("SomeVolName", "secret", "SomeSecretName", "", "SomePath")
	assert.NotNil(t, v)
	assert.Equal(t, "SomeVolName", v.Name)
	s := v.VolumeSource.Secret
	assert.NotNil(t, s)
	assert.Equal(t, "SomeSecretName", s.SecretName)
	items := s.Items
	assert.Nil(t, items)
}

func TestVolumeWithKeyOnly(t *testing.T) {
	v := getVolume("SomeVolName", "secret", "SomeSecretName", "SomeKey", "")
	assert.NotNil(t, v)
	assert.Equal(t, "SomeVolName", v.Name)
	s := v.VolumeSource.Secret
	assert.NotNil(t, s)
	assert.Equal(t, "SomeSecretName", s.SecretName)
	items := s.Items
	assert.NotNil(t, items)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "SomeKey", items[0].Key)
	assert.Equal(t, "SomeKey", items[0].Path)
}

func TestCapabilityPropertyKey(t *testing.T) {
	camelPropertyKeyStatic := CapabilityPropertyKey("quarkus.camel.cluster.kubernetes.resource-name", nil)
	assert.Equal(t, "quarkus.camel.cluster.kubernetes.resource-name", camelPropertyKeyStatic)
	vars := map[string]string{
		"camel.k.master.labelKey": "org.apache.camel/integration",
	}
	camelPropertyKeyDynamic := CapabilityPropertyKey(`quarkus.camel.cluster.kubernetes.labels."${camel.k.master.labelKey}"`, vars)
	assert.Equal(t, `quarkus.camel.cluster.kubernetes.labels."org.apache.camel/integration"`, camelPropertyKeyDynamic)
}

func TestDetermineControllerStrategyDefault(t *testing.T) {
	e := createTestEnvironment(t, v1.DefaultTraitProfile)
	strategy, err := e.DetermineControllerStrategy()
	require.NoError(t, err)
	assert.Equal(t, DefaultControllerStrategy, strategy)
}

func TestDetermineControllerStrategyAutoKnative(t *testing.T) {
	e := createTestEnvironment(t, v1.TraitProfileKnative)
	strategy, err := e.DetermineControllerStrategy()
	require.NoError(t, err)
	assert.Equal(t, ControllerStrategyKnativeService, strategy)
}

func TestDetermineControllerStrategySyntheticKitDefault(t *testing.T) {
	e := createNonManagedBuildTestEnvironment(t, v1.TraitProfileKnative)
	strategy, err := e.DetermineControllerStrategy()
	require.NoError(t, err)
	assert.Equal(t, DefaultControllerStrategy, strategy)
}

func TestDetermineControllerStrategySyntheticKitForceKnative(t *testing.T) {
	e := createNonManagedBuildTestEnvironment(t, v1.TraitProfileKnative)
	e.Integration.Spec.Traits.KnativeService = &trait.KnativeServiceTrait{
		Trait: trait.Trait{
			Enabled: ptr.To(true),
		},
		Auto: ptr.To(false),
	}
	e.Platform.ResyncStatusFullConfig()
	_, _, err := e.Catalog.apply(e)
	require.NoError(t, err)

	strategy, err := e.DetermineControllerStrategy()
	require.NoError(t, err)
	assert.Equal(t, ControllerStrategyKnativeService, strategy)
}

func createTestEnvironment(t *testing.T, profile v1.TraitProfile) *Environment {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	environment := &Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: profile,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "routes.java",
							Content: `from("direct:test").log("hello");`,
						},
						Language: v1.LanguageJavaSource,
					},
					{
						DataSpec: v1.DataSpec{
							Name:    "rests.xml",
							Content: `<rest path="/test"></rest>`,
						},
						Language: v1.LanguageXML,
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterKubernetes,
				Build: v1.IntegrationPlatformBuildSpec{
					RuntimeVersion: catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}

	environment.Platform.ResyncStatusFullConfig()

	_, _, err = traitCatalog.apply(environment)
	require.NoError(t, err)

	return environment
}

func createNonManagedBuildTestEnvironment(t *testing.T, profile v1.TraitProfile) *Environment {
	t.Helper()
	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)
	environment := &Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Status: v1.IntegrationStatus{
				// default init runtime value
				RuntimeProvider: v1.RuntimeProviderQuarkus,
				RuntimeVersion:  defaults.DefaultRuntimeVersion,
				Phase:           v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: profile,
				Traits: v1.Traits{
					Container: &trait.ContainerTrait{
						Image: "my-container-image",
					},
				},
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterKubernetes,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}

	environment.Platform.ResyncStatusFullConfig()

	_, _, err = traitCatalog.apply(environment)
	require.NoError(t, err)

	return environment
}

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
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureEnabledCamelTraitSucceeds(t *testing.T) {
	trait, environment := createNominalCamelTest(false)

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.Nil(t, condition)
	assert.True(t, configured)
}

func TestApplyCamelTraitSucceeds(t *testing.T) {
	trait, environment := createNominalCamelTest(false)
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.Nil(t, condition)
	assert.True(t, configured)

	err = trait.Apply(environment)
	require.NoError(t, err)
	assert.Equal(t, "0.0.1", environment.CamelCatalog.GetRuntimeVersion())
	assert.Equal(t, "0.0.1", environment.Integration.Status.RuntimeVersion)
	assert.Equal(t, "0.0.1", environment.IntegrationKit.Status.RuntimeVersion)
	expectedCatalog := &v1.Catalog{Version: "0.0.1", Provider: v1.RuntimeProviderQuarkus}
	assert.Equal(t, expectedCatalog, environment.Integration.Status.Catalog)
	assert.Equal(t, expectedCatalog, environment.IntegrationKit.Status.Catalog)

	// Test regex as well
	assert.True(t, exactVersionRegexp.MatchString("1.2.3"))
	assert.True(t, exactVersionRegexp.MatchString("1.0.0-SNAPSHOT"))
	assert.False(t, exactVersionRegexp.MatchString("wroong"))
}

func TestApplyCamelTraitNonManagedBuild(t *testing.T) {
	trait, environment := createNominalCamelTest(false)
	environment.Integration.Spec.Traits.Container = &traitv1.ContainerTrait{
		Image: "my-image",
	}

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	expectedCondition := NewIntegrationCondition(
		"Camel",
		v1.IntegrationConditionTraitInfo,
		corev1.ConditionTrue,
		traitConfigurationReason,
		fmt.Sprintf(
			"Operated with CamelCatalog version %s which may be different from the runtime used in the container",
			"0.0.1",
		),
	)
	assert.Equal(t, expectedCondition, condition)
	assert.True(t, configured)
	err = trait.Apply(environment)
	require.NoError(t, err)
	assert.Equal(t, "", environment.Integration.Status.RuntimeVersion)
	assert.Equal(t, v1.RuntimeProvider(""), environment.Integration.Status.RuntimeProvider)
	expectedCatalog := &v1.Catalog{Version: "0.0.1", Provider: v1.RuntimeProviderQuarkus}
	assert.Equal(t, expectedCatalog, environment.Integration.Status.Catalog)
}

func TestApplyCamelTraitWithoutEnvironmentCatalogAndUnmatchableVersionFails(t *testing.T) {
	trait, environment := createNominalCamelTest(false)
	environment.CamelCatalog = nil
	environment.Integration.Status.RuntimeVersion = "Unmatchable version"
	environment.Integration.Status.RuntimeProvider = v1.RuntimeProviderQuarkus

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.Nil(t, condition)
	assert.True(t, configured)

	err = trait.Apply(environment)
	require.Error(t, err)
	assert.Equal(t, "unable to find catalog matching version requirement: runtime=Unmatchable version, provider=quarkus", err.Error())
}

func createNominalCamelTest(withSources bool) (*camelTrait, *Environment) {
	client, _ := test.NewFakeClient()

	trait, _ := newCamelTrait().(*camelTrait)
	var sources []v1.SourceSpec
	if withSources {
		sources = []v1.SourceSpec{
			{
				DataSpec: v1.DataSpec{
					Name:    "source1.java",
					Content: "Java Source Code",
				},
				Type: "data",
			},
			{
				DataSpec: v1.DataSpec{
					Name:    "source2.xml",
					Content: "XML Source Code",
				},
				Type: "data",
			},
			{
				DataSpec: v1.DataSpec{
					Name:       "source3.xml",
					ContentRef: "my-cm1",
				},
				Type: "data",
			},
		}
	} else {
		sources = []v1.SourceSpec{}
	}
	environment := &Environment{
		CamelCatalog: &camel.RuntimeCatalog{
			CamelCatalogSpec: v1.CamelCatalogSpec{
				Loaders: map[string]v1.CamelLoader{
					"java": {
						Metadata: map[string]string{
							"native":                         boolean.TrueString,
							"sources-required-at-build-time": boolean.TrueString,
						},
					},
				},
				Runtime: v1.RuntimeSpec{
					Version:  "0.0.1",
					Provider: v1.RuntimeProviderQuarkus,
				},
			},
		},
		Catalog: NewEnvironmentTestCatalog(),
		Ctx:     context.TODO(),
		Client:  client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-integration",
				Namespace: "namespace",
			},
			Spec: v1.IntegrationSpec{
				Traits:  v1.Traits{},
				Sources: sources,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					v1.IntegrationKitLayoutLabel: v1.IntegrationKitLayoutNativeSources,
				},
				Namespace: "namespace",
			},
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
			},
			Status: v1.IntegrationPlatformStatus{
				IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
					Build: v1.IntegrationPlatformBuildSpec{
						RuntimeProvider: v1.RuntimeProviderQuarkus,
						RuntimeVersion:  "0.0.1",
					},
				},
			},
		},
		Resources:             kubernetes.NewCollection(),
		ApplicationProperties: make(map[string]string),
	}

	return trait, environment
}

func TestApplyCamelTraitWithProperties(t *testing.T) {
	trait, environment := createNominalCamelTest(false)
	trait.Properties = []string{"a=b", "c=d"}

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.Nil(t, condition)
	assert.True(t, configured)

	err = trait.Apply(environment)
	require.NoError(t, err)

	userPropertiesCm := environment.Resources.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Labels["camel.apache.org/properties.type"] == "user"
	})
	assert.NotNil(t, userPropertiesCm)
	assert.Equal(t, map[string]string{
		"application.properties": "a=b\nc=d\n",
	}, userPropertiesCm.Data)
}

func TestApplyCamelTraitNonManagedBuildWithProperties(t *testing.T) {
	trait, environment := createNominalCamelTest(false)
	trait.Properties = []string{"a=b", "c=d"}
	environment.Integration.Spec.Traits.Container = &traitv1.ContainerTrait{
		Image: "my-image",
	}

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	expectedCondition := NewIntegrationCondition(
		"Camel",
		v1.IntegrationConditionTraitInfo,
		corev1.ConditionTrue,
		traitConfigurationReason,
		fmt.Sprintf(
			"Operated with CamelCatalog version %s which may be different from the runtime used in the container",
			"0.0.1",
		),
	)
	assert.Equal(t, expectedCondition, condition)
	assert.True(t, configured)

	err = trait.Apply(environment)
	require.NoError(t, err)

	userPropertiesCm := environment.Resources.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Labels["camel.apache.org/properties.type"] == "user"
	})
	assert.NotNil(t, userPropertiesCm)
	assert.Equal(t, map[string]string{
		"application.properties": "a=b\nc=d\n",
	}, userPropertiesCm.Data)
}

func TestApplyCamelTraitWithSources(t *testing.T) {
	trait, environment := createNominalCamelTest(true)

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.Nil(t, condition)
	assert.True(t, configured)

	err = trait.Apply(environment)
	require.NoError(t, err)

	assert.Equal(t, 1, environment.Resources.Size())
	sourceCm := environment.Resources.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == "some-integration-source-000" && cm.Annotations[sourceLanguageAnnotation] == "xml" && cm.Annotations[sourceNameAnnotation] == "source2.xml"
	})
	assert.NotNil(t, sourceCm)
	assert.Equal(t, map[string]string{
		"content": "XML Source Code",
	}, sourceCm.Data)
}

func TestCamelMatches(t *testing.T) {
	t1 := camelTrait{
		BasePlatformTrait: NewBasePlatformTrait("camel", 600),
		CamelTrait: traitv1.CamelTrait{
			RuntimeVersion: "1.2.3",
		},
	}
	t2 := camelTrait{
		BasePlatformTrait: NewBasePlatformTrait("camel", 600),
		CamelTrait: traitv1.CamelTrait{
			RuntimeVersion: "1.2.3",
		},
	}

	assert.True(t, t1.Matches(&t2))
	t1.Properties = []string{"hello=world"}
	assert.True(t, t1.Matches(&t2))
	t2.RuntimeVersion = "3.2.1"
	assert.False(t, t1.Matches(&t2))
}

func TestCamelCatalogSemver(t *testing.T) {
	trait, environment := createNominalCamelTest(true)
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit
	environment.CamelCatalog.Runtime.Version = "2.16.1"
	trait.RuntimeVersion = "2.x"

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.Nil(t, condition)
	assert.True(t, configured)

	err = trait.Apply(environment)
	require.NoError(t, err)
	// 2.x will translate with 2.16.1 as it is already existing
	assert.Equal(t, "2.16.1", environment.CamelCatalog.GetRuntimeVersion())
}

func TestCamelTraitSyntheticIntegration(t *testing.T) {
	trait, environment := createNominalCamelTest(true)
	environment.Integration.Status = v1.IntegrationStatus{}
	environment.Integration.Annotations = make(map[string]string)
	environment.Integration.Annotations[v1.IntegrationSyntheticLabel] = boolean.TrueString

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.Equal(t, "explicitly disabled by the platform: synthetic integration", condition.message)
	assert.False(t, configured)

	assert.Equal(t, v1.RuntimeProvider(""), environment.Integration.Status.RuntimeProvider)
	assert.Equal(t, "", environment.Integration.Status.RuntimeVersion)
}

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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"

	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToTraitMap(t *testing.T) {
	traits := v1.Traits{
		Container: &traitv1.ContainerTrait{
			PlatformBaseTrait: traitv1.PlatformBaseTrait{},
			Name:              "test-container",
			Auto:              ptr.To(false),
			Expose:            ptr.To(true),
			Port:              8081,
			PortName:          "http-8081",
			ServicePort:       81,
			ServicePortName:   "http-81",
		},
		Service: &traitv1.ServiceTrait{
			Trait: traitv1.Trait{
				Enabled: ptr.To(true),
			},
		},
		Addons: map[string]v1.AddonTrait{
			"telemetry": toAddonTrait(t, map[string]interface{}{
				"enabled": true,
			}),
		},
	}
	expected := Options{
		"container": {
			"auto":            false,
			"expose":          true,
			"port":            float64(8081),
			"portName":        "http-8081",
			"servicePort":     float64(81),
			"servicePortName": "http-81",
			"name":            "test-container",
		},
		"service": {
			"enabled": true,
		},
		"addons": {
			"telemetry": map[string]interface{}{
				"enabled": true,
			},
		},
	}

	traitMap, err := ToTraitMap(traits)

	require.NoError(t, err)
	assert.Equal(t, expected, traitMap)
}

func TestMigrateLegacyConfiguration(t *testing.T) {
	trait := map[string]interface{}{
		"enabled":         true,
		"auto":            false,
		"port":            float64(8081),
		"portName":        "http-8081",
		"servicePortName": "http-81",
		"expose":          true,
		"name":            "test-container",
		"servicePort":     float64(81),
	}
	expected := map[string]interface{}{
		"enabled":         true,
		"auto":            false,
		"port":            float64(8081),
		"portName":        "http-8081",
		"servicePortName": "http-81",
		"expose":          true,
		"name":            "test-container",
		"servicePort":     float64(81),
	}

	err := MigrateLegacyConfiguration(trait)

	require.NoError(t, err)
	assert.Equal(t, expected, trait)
}

func TestMigrateLegacyConfiguration_invalidConfiguration(t *testing.T) {
	trait := map[string]interface{}{
		"enabled":       true,
		"configuration": "It should not be a string!",
	}

	err := MigrateLegacyConfiguration(trait)

	require.Error(t, err)
}

func TestToTrait(t *testing.T) {
	config := map[string]interface{}{
		"auto":            false,
		"expose":          true,
		"port":            8081,
		"portName":        "http-8081",
		"servicePort":     81,
		"servicePortName": "http-81",
		"name":            "test-container",
	}
	expected := traitv1.ContainerTrait{
		PlatformBaseTrait: traitv1.PlatformBaseTrait{},
		Name:              "test-container",
		Auto:              ptr.To(false),
		Expose:            ptr.To(true),
		Port:              8081,
		PortName:          "http-8081",
		ServicePort:       81,
		ServicePortName:   "http-81",
	}

	trait := traitv1.ContainerTrait{}
	err := ToTrait(config, &trait)

	require.NoError(t, err)
	assert.Equal(t, expected, trait)
}

func TestSameTraits(t *testing.T) {
	c, err := internal.NewFakeClient()
	require.NoError(t, err)

	t.Run("same traits with annotations only", func(t *testing.T) {
		oldKlb := &v1.Pipe{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1.TraitAnnotationPrefix + "container.image": "foo/bar:1",
				},
			},
		}
		newKlb := &v1.Pipe{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1.TraitAnnotationPrefix + "container.image": "foo/bar:1",
				},
			},
		}

		ok, err := PipesHaveSameTraits(c, oldKlb, newKlb)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("not same traits with annotations only", func(t *testing.T) {
		oldKlb := &v1.Pipe{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1.TraitAnnotationPrefix + "container.image": "foo/bar:1",
				},
			},
		}
		newKlb := &v1.Pipe{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1.TraitAnnotationPrefix + "container.image": "foo/bar:2",
				},
			},
		}

		ok, err := PipesHaveSameTraits(c, oldKlb, newKlb)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestHasMathchingTraitsEmpty(t *testing.T) {
	opt1 := Options{
		"builder": {},
		"camel": {
			"runtimeVersion": "1.2.3",
		},
		"quarkus": {},
	}
	opt2 := Options{
		"camel": {
			"runtimeVersion": "1.2.3",
		},
	}
	opt3 := Options{
		"camel": {
			"runtimeVersion": "1.2.3",
		},
	}
	opt4 := Options{
		"camel": {
			"runtimeVersion": "3.2.1",
		},
	}
	b1, err := HasMatchingTraits(opt1, opt2)
	assert.Nil(t, err)
	assert.True(t, b1)
	b2, err := HasMatchingTraits(opt1, opt4)
	assert.Nil(t, err)
	assert.False(t, b2)
	b3, err := HasMatchingTraits(opt2, opt3)
	assert.Nil(t, err)
	assert.True(t, b3)
}

func TestHasMathchingTraitsMissing(t *testing.T) {
	opt1 := Options{}
	opt2 := Options{
		"camel": {
			"properties": []string{"a=1"},
		},
	}
	b1, err := HasMatchingTraits(opt1, opt2)
	assert.Nil(t, err)
	assert.True(t, b1)
}

func TestIntegrationAndPipeSameTraits(t *testing.T) {
	pipe := &v1.Pipe{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				v1.TraitAnnotationPrefix + "camel.runtime-version": "1.2.3",
			},
		},
	}

	integration := &v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Camel: &traitv1.CamelTrait{
					RuntimeVersion: "1.2.3",
				},
			},
		},
	}
	c, err := internal.NewFakeClient(pipe, integration)
	require.NoError(t, err)

	result, err := IntegrationAndPipeSameTraits(c, integration, pipe)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestMergePlatformTraits(t *testing.T) {
	integration := &v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Camel: &traitv1.CamelTrait{
					Properties: []string{"hello=world"},
				},
			},
		},
	}
	platform := &v1.IntegrationPlatform{
		Status: v1.IntegrationPlatformStatus{
			IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
				Traits: v1.Traits{
					Camel: &traitv1.CamelTrait{
						RuntimeVersion: "1.2.3",
					},
				},
			},
		},
	}

	expectedOptions := Options{
		"camel": {
			"properties":     []any{"hello=world"},
			"runtimeVersion": "1.2.3",
		},
	}

	c, err := internal.NewFakeClient()
	require.NoError(t, err)
	mergedOptions, err := NewSpecTraitsOptionsForIntegrationAndPlatform(c, integration, nil, platform)
	require.NoError(t, err)
	assert.Equal(t, expectedOptions, mergedOptions)
}

func TestMergePlatformTraitsIntegrationPriority(t *testing.T) {
	integration := &v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Camel: &traitv1.CamelTrait{
					Properties:     []string{"hello=world"},
					RuntimeVersion: "0.0.0",
				},
			},
		},
	}
	platform := &v1.IntegrationPlatform{
		Status: v1.IntegrationPlatformStatus{
			IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
				Traits: v1.Traits{
					Camel: &traitv1.CamelTrait{
						RuntimeVersion: "1.2.3",
					},
				},
			},
		},
	}

	expectedOptions := Options{
		"camel": {
			"properties":     []any{"hello=world"},
			"runtimeVersion": "0.0.0",
		},
	}

	c, err := internal.NewFakeClient()
	require.NoError(t, err)
	mergedOptions, err := NewSpecTraitsOptionsForIntegrationAndPlatform(c, integration, nil, platform)
	require.NoError(t, err)
	assert.Equal(t, expectedOptions, mergedOptions)
}

func TestMergeIntegrationProfileTraits(t *testing.T) {
	integration := &v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Camel: &traitv1.CamelTrait{
					Properties: []string{"hello=world"},
				},
			},
		},
	}
	profile := &v1.IntegrationProfile{
		Spec: v1.IntegrationProfileSpec{
			Traits: v1.Traits{
				Camel: &traitv1.CamelTrait{
					RuntimeVersion: "1.2.3",
				},
			},
		},
	}

	expectedOptions := Options{
		"camel": {
			"properties":     []any{"hello=world"},
			"runtimeVersion": "1.2.3",
		},
	}

	c, err := internal.NewFakeClient()
	require.NoError(t, err)
	mergedOptions, err := NewSpecTraitsOptionsForIntegrationAndPlatform(c, integration, profile, nil)
	require.NoError(t, err)
	assert.Equal(t, expectedOptions, mergedOptions)
}

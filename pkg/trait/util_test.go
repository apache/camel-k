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
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToTraitMap(t *testing.T) {
	traits := v1.Traits{
		Container: &traitv1.ContainerTrait{
			PlatformBaseTrait: traitv1.PlatformBaseTrait{},
			Name:              "test-container",
			Auto:              pointer.Bool(false),
			Expose:            pointer.Bool(true),
			Port:              8081,
			PortName:          "http-8081",
			ServicePort:       81,
			ServicePortName:   "http-81",
		},
		Service: &traitv1.ServiceTrait{
			Trait: traitv1.Trait{
				Enabled: pointer.Bool(true),
			},
		},
		Addons: map[string]v1.AddonTrait{
			"telemetry": ToAddonTrait(t, map[string]interface{}{
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

func TestToPropertyMap(t *testing.T) {
	trait := traitv1.ContainerTrait{
		PlatformBaseTrait: traitv1.PlatformBaseTrait{},
		Name:              "test-container",
		Auto:              pointer.Bool(false),
		Expose:            pointer.Bool(true),
		Port:              8081,
		PortName:          "http-8081",
		ServicePort:       81,
		ServicePortName:   "http-81",
	}
	expected := map[string]interface{}{
		"auto":            false,
		"expose":          true,
		"port":            float64(8081),
		"portName":        "http-8081",
		"servicePort":     float64(81),
		"servicePortName": "http-81",
		"name":            "test-container",
	}

	propMap, err := ToPropertyMap(trait)

	require.NoError(t, err)
	assert.Equal(t, expected, propMap)
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
		Auto:              pointer.Bool(false),
		Expose:            pointer.Bool(true),
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
	t.Run("empty traits", func(t *testing.T) {
		oldKlb := &v1.Pipe{
			Spec: v1.PipeSpec{
				Integration: &v1.IntegrationSpec{
					Traits: v1.Traits{},
				},
			},
		}
		newKlb := &v1.Pipe{
			Spec: v1.PipeSpec{
				Integration: &v1.IntegrationSpec{
					Traits: v1.Traits{},
				},
			},
		}

		ok, err := PipesHaveSameTraits(oldKlb, newKlb)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("same traits", func(t *testing.T) {
		oldKlb := &v1.Pipe{
			Spec: v1.PipeSpec{
				Integration: &v1.IntegrationSpec{
					Traits: v1.Traits{
						Container: &traitv1.ContainerTrait{
							Image: "foo/bar:1",
						},
					},
				},
			},
		}
		newKlb := &v1.Pipe{
			Spec: v1.PipeSpec{
				Integration: &v1.IntegrationSpec{
					Traits: v1.Traits{
						Container: &traitv1.ContainerTrait{
							Image: "foo/bar:1",
						},
					},
				},
			},
		}

		ok, err := PipesHaveSameTraits(oldKlb, newKlb)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("not same traits", func(t *testing.T) {
		oldKlb := &v1.Pipe{
			Spec: v1.PipeSpec{
				Integration: &v1.IntegrationSpec{
					Traits: v1.Traits{
						Container: &traitv1.ContainerTrait{
							Image: "foo/bar:1",
						},
					},
				},
			},
		}
		newKlb := &v1.Pipe{
			Spec: v1.PipeSpec{
				Integration: &v1.IntegrationSpec{
					Traits: v1.Traits{
						Owner: &traitv1.OwnerTrait{
							TargetAnnotations: []string{"foo"},
						},
					},
				},
			},
		}

		ok, err := PipesHaveSameTraits(oldKlb, newKlb)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("same traits with annotations", func(t *testing.T) {
		oldKlb := &v1.Pipe{
			Spec: v1.PipeSpec{
				Integration: &v1.IntegrationSpec{
					Traits: v1.Traits{
						Container: &traitv1.ContainerTrait{
							Image: "foo/bar:1",
						},
					},
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

		ok, err := PipesHaveSameTraits(oldKlb, newKlb)
		require.NoError(t, err)
		assert.True(t, ok)
	})

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

		ok, err := PipesHaveSameTraits(oldKlb, newKlb)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("not same traits with annotations", func(t *testing.T) {
		oldKlb := &v1.Pipe{
			Spec: v1.PipeSpec{
				Integration: &v1.IntegrationSpec{
					Traits: v1.Traits{
						Container: &traitv1.ContainerTrait{
							Image: "foo/bar:1",
						},
					},
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

		ok, err := PipesHaveSameTraits(oldKlb, newKlb)
		require.NoError(t, err)
		assert.False(t, ok)
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

		ok, err := PipesHaveSameTraits(oldKlb, newKlb)
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

func TestFromAnnotationsPlain(t *testing.T) {
	meta := metav1.ObjectMeta{
		Annotations: map[string]string{
			"trait.camel.apache.org/trait.prop1": "hello1",
			"trait.camel.apache.org/trait.prop2": "hello2",
		},
	}
	opt, err := FromAnnotations(&meta)
	require.NoError(t, err)
	tt, ok := opt.Get("trait")
	assert.True(t, ok)
	assert.Equal(t, "hello1", tt["prop1"])
	assert.Equal(t, "hello2", tt["prop2"])
}

func TestFromAnnotationsArray(t *testing.T) {
	meta := metav1.ObjectMeta{
		Annotations: map[string]string{
			"trait.camel.apache.org/trait.prop1": "[hello,world]",
			// The func should trim empty spaces as well
			"trait.camel.apache.org/trait.prop2": "[\"hello=1\", \"world=2\"]",
		},
	}
	opt, err := FromAnnotations(&meta)
	require.NoError(t, err)
	tt, ok := opt.Get("trait")
	assert.True(t, ok)
	assert.Equal(t, []string{"hello", "world"}, tt["prop1"])
	assert.Equal(t, []string{"\"hello=1\"", "\"world=2\""}, tt["prop2"])
}

func TestFromAnnotationsArrayEmpty(t *testing.T) {
	meta := metav1.ObjectMeta{
		Annotations: map[string]string{
			"trait.camel.apache.org/trait.prop": "[]",
		},
	}
	opt, err := FromAnnotations(&meta)
	require.NoError(t, err)
	tt, ok := opt.Get("trait")
	assert.True(t, ok)
	assert.Equal(t, []string{}, tt["prop"])
}

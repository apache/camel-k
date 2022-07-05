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

	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"

	"github.com/stretchr/testify/assert"
)

func TestToTraitMap(t *testing.T) {
	traits := v1.Traits{
		Container: &traitv1.ContainerTrait{
			Trait: traitv1.Trait{
				Enabled: pointer.Bool(true),
				Configuration: configurationFromMap(t, map[string]interface{}{
					"name": "test-container",
					"port": 8082,
				}),
			},
			Auto:            pointer.Bool(false),
			Expose:          pointer.Bool(true),
			Port:            8081,
			PortName:        "http-8081",
			ServicePort:     81,
			ServicePortName: "http-81",
		},
		Service: &traitv1.ServiceTrait{
			Trait: traitv1.Trait{
				Enabled: pointer.Bool(true),
			},
		},
		Addons: map[string]v1.AddonTrait{
			"tracing": ToAddonTrait(t, map[string]interface{}{
				"enabled": true,
			}),
		},
	}
	expected := map[string]map[string]interface{}{
		"container": {
			"enabled":         true,
			"auto":            false,
			"expose":          true,
			"port":            float64(8081),
			"portName":        "http-8081",
			"servicePort":     float64(81),
			"servicePortName": "http-81",
			"configuration": map[string]interface{}{
				"name": "test-container",
				"port": float64(8082),
			},
		},
		"service": {
			"enabled": true,
		},
		"addons": {
			"tracing": map[string]interface{}{
				"enabled": true,
			},
		},
	}

	traitMap, err := ToTraitMap(traits)

	assert.NoError(t, err)
	assert.Equal(t, expected, traitMap)
}

func TestToPropertyMap(t *testing.T) {
	trait := traitv1.ContainerTrait{
		Trait: traitv1.Trait{
			Enabled: pointer.Bool(true),
			Configuration: configurationFromMap(t, map[string]interface{}{
				"name": "test-container",
				"port": 8082,
			}),
		},
		Auto:            pointer.Bool(false),
		Expose:          pointer.Bool(true),
		Port:            8081,
		PortName:        "http-8081",
		ServicePort:     81,
		ServicePortName: "http-81",
	}
	expected := map[string]interface{}{
		"enabled":         true,
		"auto":            false,
		"expose":          true,
		"port":            float64(8081),
		"portName":        "http-8081",
		"servicePort":     float64(81),
		"servicePortName": "http-81",
		"configuration": map[string]interface{}{
			"name": "test-container",
			"port": float64(8082),
		},
	}

	propMap, err := ToPropertyMap(trait)

	assert.NoError(t, err)
	assert.Equal(t, expected, propMap)
}

func TestMigrateLegacyConfiguration(t *testing.T) {
	trait := map[string]interface{}{
		"enabled":         true,
		"auto":            false,
		"port":            float64(8081),
		"portName":        "http-8081",
		"servicePortName": "http-81",
		"configuration": map[string]interface{}{
			"auto":        true,
			"expose":      true,
			"name":        "test-container",
			"port":        float64(8082),
			"servicePort": float64(81),
		},
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

	assert.NoError(t, err)
	assert.Equal(t, expected, trait)
}

func TestMigrateLegacyConfiguration_invalidConfiguration(t *testing.T) {
	trait := map[string]interface{}{
		"enabled":       true,
		"configuration": "It should not be a string!",
	}

	err := MigrateLegacyConfiguration(trait)

	assert.Error(t, err)
}

func TestToTrait(t *testing.T) {
	config := map[string]interface{}{
		"enabled":         true,
		"auto":            false,
		"expose":          true,
		"port":            8081,
		"portName":        "http-8081",
		"servicePort":     81,
		"servicePortName": "http-81",
		"configuration": map[string]interface{}{
			"name": "test-container",
			"port": float64(8082),
		},
	}
	expected := traitv1.ContainerTrait{
		Trait: traitv1.Trait{
			Enabled: pointer.Bool(true),
			Configuration: configurationFromMap(t, map[string]interface{}{
				"name": "test-container",
				"port": 8082,
			}),
		},
		Auto:            pointer.Bool(false),
		Expose:          pointer.Bool(true),
		Port:            8081,
		PortName:        "http-8081",
		ServicePort:     81,
		ServicePortName: "http-81",
	}

	trait := traitv1.ContainerTrait{}
	err := ToTrait(config, &trait)

	assert.NoError(t, err)
	assert.Equal(t, expected, trait)
}

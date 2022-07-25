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

package v1

import (
	"encoding/json"
	"testing"

	"github.com/apache/camel-k/pkg/apis/camel/v1/trait"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/pointer"
)

func TestTraitsMerge(t *testing.T) {
	t1 := Traits{
		Container: &trait.ContainerTrait{
			Trait: trait.Trait{
				Configuration: configurationFromMap(t, map[string]interface{}{
					"name": "test-container",
				}),
			},
			Auto:        pointer.Bool(false),
			ServicePort: 81,
		},
		Logging: &trait.LoggingTrait{
			Color: pointer.Bool(false),
			Level: "INFO",
		},
		Addons: map[string]AddonTrait{
			"master": toAddonTrait(t, map[string]interface{}{
				"resourceName": "test-lock",
			}),
			"tracing": toAddonTrait(t, map[string]interface{}{
				"enabled": true,
			}),
		},
	}
	t2 := Traits{
		Container: &trait.ContainerTrait{
			Trait: trait.Trait{
				Configuration: configurationFromMap(t, map[string]interface{}{
					"port": 8081,
				}),
			},
			PortName: "http-8081",
		},
		Logging: &trait.LoggingTrait{
			Color: pointer.Bool(true),
			Level: "DEBUG",
		},
		Addons: map[string]AddonTrait{
			"tracing": toAddonTrait(t, map[string]interface{}{
				"serviceName": "test-integration",
			}),
		},
	}

	err := t1.Merge(t2)

	require.NoError(t, err)

	assert.NotNil(t, t1.Container)
	assert.False(t, pointer.BoolDeref(t1.Container.Auto, true))
	assert.Equal(t, "http-8081", t1.Container.PortName)
	assert.Equal(t, 81, t1.Container.ServicePort)
	assert.Equal(t,
		configurationFromMap(t, map[string]interface{}{
			"name": "test-container",
			"port": 8081,
		}),
		t1.Container.Configuration)

	// values from merged trait take precedence over the original ones
	assert.NotNil(t, t1.Logging)
	assert.True(t, pointer.BoolDeref(t1.Logging.Color, false))
	assert.Equal(t, "DEBUG", t1.Logging.Level)

	assert.NotNil(t, t1.Addons)
	assert.Equal(t,
		toAddonTrait(t, map[string]interface{}{
			"resourceName": "test-lock",
		}),
		t1.Addons["master"])
	assert.Equal(t,
		toAddonTrait(t, map[string]interface{}{
			"enabled":     true,
			"serviceName": "test-integration",
		}),
		t1.Addons["tracing"])
}

func configurationFromMap(t *testing.T, configMap map[string]interface{}) *trait.Configuration {
	t.Helper()

	data, err := json.Marshal(configMap)
	require.NoError(t, err)

	return &trait.Configuration{
		RawMessage: data,
	}
}

func toAddonTrait(t *testing.T, config map[string]interface{}) AddonTrait {
	t.Helper()

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var addon AddonTrait
	err = json.Unmarshal(data, &addon)
	require.NoError(t, err)

	return addon
}

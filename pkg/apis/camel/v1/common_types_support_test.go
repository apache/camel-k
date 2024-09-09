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
	"fmt"
	"testing"

	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestTraitsMerge(t *testing.T) {
	t1 := Traits{
		Container: &trait.ContainerTrait{
			Name:        "test-container",
			Auto:        ptr.To(false),
			ServicePort: 81,
		},
		Logging: &trait.LoggingTrait{
			Color: ptr.To(false),
			Level: "INFO",
		},
		Addons: map[string]AddonTrait{
			"master": toAddonTrait(t, map[string]interface{}{
				"resourceName": "test-lock",
			}),
			"telemetry": toAddonTrait(t, map[string]interface{}{
				"enabled": true,
			}),
		},
	}
	t2 := Traits{
		Container: &trait.ContainerTrait{
			Port:     8081,
			PortName: "http-8081",
		},
		Logging: &trait.LoggingTrait{
			Color: ptr.To(true),
			Level: "DEBUG",
		},
		Addons: map[string]AddonTrait{
			"telemetry": toAddonTrait(t, map[string]interface{}{
				"serviceName": "test-integration",
			}),
		},
	}

	err := t1.Merge(t2)

	require.NoError(t, err)

	assert.NotNil(t, t1.Container)
	assert.False(t, ptr.Deref(t1.Container.Auto, true))
	assert.Equal(t, "http-8081", t1.Container.PortName)
	assert.Equal(t, 81, t1.Container.ServicePort)

	// values from merged trait take precedence over the original ones
	assert.NotNil(t, t1.Logging)
	assert.True(t, ptr.Deref(t1.Logging.Color, false))
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
		t1.Addons["telemetry"])
}

func TestIntegrationKitTraitsMerge(t *testing.T) {
	t1 := IntegrationKitTraits{
		Builder: &trait.BuilderTrait{
			Properties: []string{
				"b1=value_b1",
			},
		},
		Camel: &trait.CamelTrait{
			RuntimeVersion: "0.99.0",
			Properties: []string{
				"c1=value_c1",
			},
		},
		Addons: map[string]AddonTrait{
			"master": toAddonTrait(t, map[string]interface{}{
				"resourceName": "test-lock",
			}),
			"telemetry": toAddonTrait(t, map[string]interface{}{
				"enabled": true,
			}),
		},
	}
	t2 := IntegrationKitTraits{
		Builder: &trait.BuilderTrait{
			Properties: []string{
				"b2=value_b2",
			},
		},
		Quarkus: &trait.QuarkusTrait{
			NativeBaseImage: "quay.io/quarkus/quarkus-micro-image:2.0",
		},
		Addons: map[string]AddonTrait{
			"telemetry": toAddonTrait(t, map[string]interface{}{
				"serviceName": "test-integration",
			}),
		},
	}

	err := t1.Merge(t2)

	require.NoError(t, err)

	assert.NotNil(t, t1.Builder)
	assert.Equal(t, 1, len(t1.Builder.Properties))
	assert.Equal(t, "b2=value_b2", t1.Builder.Properties[0])

	assert.NotNil(t, t1.Camel)
	assert.Equal(t, "0.99.0", t1.Camel.RuntimeVersion)
	assert.Equal(t, 1, len(t1.Camel.Properties))
	assert.Equal(t, "c1=value_c1", t1.Camel.Properties[0])

	assert.NotNil(t, t1.Quarkus)
	assert.Equal(t, "quay.io/quarkus/quarkus-micro-image:2.0", t1.Quarkus.NativeBaseImage)

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
		t1.Addons["telemetry"])
}

func TestDecodeValueSourceValid(t *testing.T) {
	res, err := DecodeValueSource("configmap:my-configmap", "defaultkey", "errorMessage")
	require.NoError(t, err)

	assert.NotNil(t, res)
	assert.Nil(t, res.SecretKeyRef)
	assert.NotNil(t, res.ConfigMapKeyRef)
	assert.Equal(t, "defaultkey", res.ConfigMapKeyRef.Key)

	res, err = DecodeValueSource("configmap:my-configmap/my-key", "defaultkey", "errorMessage")
	require.NoError(t, err)

	assert.NotNil(t, res)
	assert.Nil(t, res.SecretKeyRef)
	assert.NotNil(t, res.ConfigMapKeyRef)
	assert.Equal(t, "my-key", res.ConfigMapKeyRef.Key)

	res, err = DecodeValueSource("secret:my-secret/mykey", "defaultkey", "errorMessage")
	require.NoError(t, err)

	assert.NotNil(t, res)
	assert.Nil(t, res.ConfigMapKeyRef)
	assert.NotNil(t, res.SecretKeyRef)
	assert.Equal(t, "mykey", res.SecretKeyRef.Key)

	res, err = DecodeValueSource("secret:my-secret", "defaultkey", "errorMessage")
	require.NoError(t, err)

	assert.NotNil(t, res)
	assert.Nil(t, res.ConfigMapKeyRef)
	assert.NotNil(t, res.SecretKeyRef)
	assert.Equal(t, "defaultkey", res.SecretKeyRef.Key)
}

func TestDecodeValueSourceInvalid(t *testing.T) {
	testcases := []struct {
		name         string
		input        string
		defaultKey   string
		errorMessage string
	}{
		{
			name:         "invalidResource",
			input:        "invalid:my-resource",
			defaultKey:   "defaultKey",
			errorMessage: "invalidResource",
		},
		{
			name:         "noResourceName",
			input:        "secret:",
			defaultKey:   "defaultKey",
			errorMessage: "noResourceName",
		},
		{
			name:         "invalidResourceName",
			input:        "configmap:***",
			defaultKey:   "defaultKey",
			errorMessage: "errorMessage",
		},
		{
			name:         "invalidResourceKey",
			input:        "configmap:my-cm/-",
			defaultKey:   "defaultKey",
			errorMessage: "invalidResourceKey",
		},
		{
			name:         "invalidResourceNameWithKey",
			input:        "configmap:/my-key",
			defaultKey:   "defaultKey",
			errorMessage: "invalidResourceNameWithKey",
		}}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.name), func(t *testing.T) {
			res, err := DecodeValueSource(tc.input, tc.defaultKey, tc.errorMessage)
			require.Error(t, err)
			assert.Equal(t, ValueSource{}, res)
			assert.Equal(t, err.Error(), tc.errorMessage)
		})
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

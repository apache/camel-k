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

package bindings

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindingConverter(t *testing.T) {
	testcases := []struct {
		name                  string
		endpointType          v1.EndpointType
		uri                   string
		step                  map[string]interface{}
		endpointProperties    map[string]string
		applicationProperties map[string]string
	}{
		{
			name:         "source",
			endpointType: v1.EndpointTypeSource,
			uri:          "kamelet:mykamelet/source-0",
			step:         nil,
		},
		{
			name:         "source-properties",
			endpointType: v1.EndpointTypeSource,
			uri:          "kamelet:mykamelet/source-1",
			step:         nil,
			endpointProperties: map[string]string{
				"foo": "bar",
			},
			applicationProperties: map[string]string{
				"camel.kamelet.mykamelet.source-1.foo": "bar",
			},
		},
		{
			name:         "action",
			endpointType: v1.EndpointTypeAction,
			uri:          "",
			step: map[string]interface{}{
				"kamelet": map[string]interface{}{
					"name": "mykamelet/action-2",
				},
			},
		},
		{
			name:         "sink",
			endpointType: v1.EndpointTypeSink,
			uri:          "kamelet:mykamelet/sink-3",
			step:         nil,
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.name), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := internal.NewFakeClient()
			require.NoError(t, err)

			endpoint := v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kamelet",
					APIVersion: "camel.apache.org/v1any1",
					Name:       "mykamelet",
				},
			}

			if len(tc.endpointProperties) > 0 {
				endpoint.Properties = asEndpointProperties(tc.endpointProperties)
			}

			binding, err := BindingConverter{}.Translate(
				BindingContext{
					Ctx:       ctx,
					Client:    client,
					Namespace: "test",
					Profile:   v1.TraitProfileKubernetes,
				},
				EndpointContext{
					Type:     tc.endpointType,
					Position: &i,
				},
				endpoint)

			require.NoError(t, err)
			assert.NotNil(t, binding)
			assert.Equal(t, tc.step, binding.Step)
			assert.Equal(t, tc.uri, binding.URI)

			if len(tc.applicationProperties) > 0 {
				assert.Equal(t, tc.applicationProperties, binding.ApplicationProperties)
			} else {
				assert.Empty(t, binding.ApplicationProperties)
			}
		})
	}
}

func TestBindingConverterWithDataTypes(t *testing.T) {
	testcases := []struct {
		name                  string
		endpointType          v1.EndpointType
		uri                   string
		step                  map[string]interface{}
		endpointProperties    map[string]string
		applicationProperties map[string]string
		inputScheme           string
		inputFormat           string
		outputScheme          string
		outputFormat          string
	}{
		{
			name:         "action-input",
			endpointType: v1.EndpointTypeAction,
			uri:          "",
			step:         getExpectedStep(true, false, defaultDataTypeActionKamelet),
			endpointProperties: map[string]string{
				"foo": "bar",
			},
			applicationProperties: map[string]string{
				"camel.kamelet.mykamelet.action-0.foo":              "bar",
				"camel.kamelet.data-type-action.action-0-in.scheme": "camel",
				"camel.kamelet.data-type-action.action-0-in.format": "string",
			},
			inputFormat: "string",
		},
		{
			name:         "action-input-scheme-prefix",
			endpointType: v1.EndpointTypeAction,
			uri:          "",
			step:         getExpectedStep(true, false, defaultDataTypeActionKamelet),
			endpointProperties: map[string]string{
				"foo": "bar",
			},
			applicationProperties: map[string]string{
				"camel.kamelet.mykamelet.action-0.foo":              "bar",
				"camel.kamelet.data-type-action.action-0-in.scheme": "foo",
				"camel.kamelet.data-type-action.action-0-in.format": "string",
			},
			inputFormat: "foo:string",
		},
		{
			name:         "action-output",
			endpointType: v1.EndpointTypeAction,
			uri:          "",
			step:         getExpectedStep(false, true, defaultDataTypeActionKamelet),
			endpointProperties: map[string]string{
				"foo": "bar",
			},
			applicationProperties: map[string]string{
				"camel.kamelet.mykamelet.action-0.foo":               "bar",
				"camel.kamelet.data-type-action.action-0-out.scheme": "camel",
				"camel.kamelet.data-type-action.action-0-out.format": "string",
			},
			outputFormat: "string",
		},
		{
			name:         "action-output-scheme-prefix",
			endpointType: v1.EndpointTypeAction,
			uri:          "",
			step:         getExpectedStep(false, true, defaultDataTypeActionKamelet),
			endpointProperties: map[string]string{
				"foo": "bar",
			},
			applicationProperties: map[string]string{
				"camel.kamelet.mykamelet.action-0.foo":               "bar",
				"camel.kamelet.data-type-action.action-0-out.scheme": "foo",
				"camel.kamelet.data-type-action.action-0-out.format": "string",
			},
			outputFormat: "foo:string",
		},
		{
			name:         "action-input-output",
			endpointType: v1.EndpointTypeAction,
			uri:          "",
			step:         getExpectedStep(true, true, defaultDataTypeActionKamelet),
			endpointProperties: map[string]string{
				"foo": "bar",
			},
			applicationProperties: map[string]string{
				"camel.kamelet.mykamelet.action-0.foo":               "bar",
				"camel.kamelet.data-type-action.action-0-in.scheme":  "foo",
				"camel.kamelet.data-type-action.action-0-in.format":  "string",
				"camel.kamelet.data-type-action.action-0-out.scheme": "foo",
				"camel.kamelet.data-type-action.action-0-out.format": "binary",
			},
			inputScheme:  "foo",
			inputFormat:  "string",
			outputScheme: "foo",
			outputFormat: "binary",
		},
		{
			name:         "action-input-output-schema-and-prefix",
			endpointType: v1.EndpointTypeAction,
			uri:          "",
			step:         getExpectedStep(true, true, defaultDataTypeActionKamelet),
			endpointProperties: map[string]string{
				"foo": "bar",
			},
			applicationProperties: map[string]string{
				"camel.kamelet.mykamelet.action-0.foo":               "bar",
				"camel.kamelet.data-type-action.action-0-in.scheme":  "camel",
				"camel.kamelet.data-type-action.action-0-in.format":  "foo:string",
				"camel.kamelet.data-type-action.action-0-out.scheme": "camel",
				"camel.kamelet.data-type-action.action-0-out.format": "foo:binary",
			},
			inputScheme:  "camel",
			inputFormat:  "foo:string",
			outputScheme: "camel",
			outputFormat: "foo:binary",
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.name), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := internal.NewFakeClient()
			require.NoError(t, err)

			endpoint := v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kamelet",
					APIVersion: "camel.apache.org/v1any1",
					Name:       "mykamelet",
				},
			}

			if len(tc.endpointProperties) > 0 {
				endpoint.Properties = asEndpointProperties(tc.endpointProperties)
			}

			endpoint.DataTypes = make(map[v1.TypeSlot]v1.DataTypeReference)
			if tc.inputFormat != "" {
				endpoint.DataTypes[v1.TypeSlotIn] = v1.DataTypeReference{
					Scheme: tc.inputScheme,
					Format: tc.inputFormat,
				}
			}

			if tc.outputFormat != "" {
				endpoint.DataTypes[v1.TypeSlotOut] = v1.DataTypeReference{
					Scheme: tc.outputScheme,
					Format: tc.outputFormat,
				}
			}

			pos := 0
			binding, err := BindingConverter{}.Translate(
				BindingContext{
					Ctx:       ctx,
					Client:    client,
					Namespace: "test",
					Profile:   v1.TraitProfileKubernetes,
				},
				EndpointContext{
					Type:     tc.endpointType,
					Position: &pos,
				},
				endpoint)

			require.NoError(t, err)
			assert.NotNil(t, binding)
			assert.Equal(t, tc.step, binding.Step)
			assert.Equal(t, tc.uri, binding.URI)

			if len(tc.applicationProperties) > 0 {
				assert.Equal(t, tc.applicationProperties, binding.ApplicationProperties)
			} else {
				assert.Empty(t, binding.ApplicationProperties)
			}
		})
	}
}

func TestBindingConverterWithDataTypesOverridden(t *testing.T) {
	testcases := []struct {
		name                  string
		endpointType          v1.EndpointType
		uri                   string
		step                  map[string]interface{}
		endpointProperties    map[string]string
		applicationProperties map[string]string
		inputScheme           string
		inputFormat           string
		outputScheme          string
		outputFormat          string
	}{
		{
			name:         "action-input",
			endpointType: v1.EndpointTypeAction,
			uri:          "",
			step:         getExpectedStep(true, false, "data-type-action-v2"),
			endpointProperties: map[string]string{
				"foo": "bar",
			},
			applicationProperties: map[string]string{
				"camel.kamelet.mykamelet.action-0.foo":                 "bar",
				"camel.kamelet.data-type-action-v2.action-0-in.scheme": "camel",
				"camel.kamelet.data-type-action-v2.action-0-in.format": "string",
			},
			inputFormat: "string",
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.name), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := internal.NewFakeClient()
			require.NoError(t, err)

			endpoint := v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kamelet",
					APIVersion: "camel.apache.org/v1any1",
					Name:       "mykamelet",
				},
			}

			if len(tc.endpointProperties) > 0 {
				endpoint.Properties = asEndpointProperties(tc.endpointProperties)
			}

			endpoint.DataTypes = make(map[v1.TypeSlot]v1.DataTypeReference)
			if tc.inputFormat != "" {
				endpoint.DataTypes[v1.TypeSlotIn] = v1.DataTypeReference{
					Scheme: tc.inputScheme,
					Format: tc.inputFormat,
				}
			}

			if tc.outputFormat != "" {
				endpoint.DataTypes[v1.TypeSlotOut] = v1.DataTypeReference{
					Scheme: tc.outputScheme,
					Format: tc.outputFormat,
				}
			}

			pos := 0
			binding, err := BindingConverter{}.Translate(
				BindingContext{
					Ctx:       ctx,
					Client:    client,
					Namespace: "test",
					Profile:   v1.TraitProfileKubernetes,
					Metadata: map[string]string{
						v1.KameletDataTypeLabel: "data-type-action-v2",
					},
				},
				EndpointContext{
					Type:     tc.endpointType,
					Position: &pos,
				},
				endpoint)

			require.NoError(t, err)
			assert.NotNil(t, binding)
			assert.Equal(t, tc.step, binding.Step)
			assert.Equal(t, tc.uri, binding.URI)

			if len(tc.applicationProperties) > 0 {
				assert.Equal(t, tc.applicationProperties, binding.ApplicationProperties)
			} else {
				assert.Empty(t, binding.ApplicationProperties)
			}
		})
	}
}

func getExpectedStep(withIn bool, withOut bool, dataTypeActionKamelet string) map[string]interface{} {
	var steps []map[string]interface{}

	if withIn {
		steps = append(steps, map[string]interface{}{
			"kamelet": map[string]interface{}{
				"name": dataTypeActionKamelet + "/action-0-in",
			},
		})
	}

	steps = append(steps, map[string]interface{}{
		"kamelet": map[string]interface{}{
			"name": "mykamelet/action-0",
		},
	})

	if withOut {
		steps = append(steps, map[string]interface{}{
			"kamelet": map[string]interface{}{
				"name": dataTypeActionKamelet + "/action-0-out",
			},
		})
	}

	return map[string]interface{}{
		"pipeline": map[string]interface{}{
			"id":    "action-0-pipeline",
			"steps": steps,
		},
	}
}

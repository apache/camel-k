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

	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestKameletBindingProvider(t *testing.T) {
	testcases := []struct {
		name                  string
		endpointType          v1alpha1.EndpointType
		uri                   string
		step                  map[string]interface{}
		endpointProperties    map[string]string
		applicationProperties map[string]string
	}{
		{
			name:         "source",
			endpointType: v1alpha1.EndpointTypeSource,
			uri:          "kamelet:mykamelet/source-0",
			step:         nil,
		},
		{
			name:         "source-properties",
			endpointType: v1alpha1.EndpointTypeSource,
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
			endpointType: v1alpha1.EndpointTypeAction,
			uri:          "",
			step: map[string]interface{}{
				"kamelet": map[string]interface{}{
					"name": "mykamelet/action-2",
				},
			},
		},
		{
			name:         "sink",
			endpointType: v1alpha1.EndpointTypeSink,
			uri:          "kamelet:mykamelet/sink-3",
			step:         nil,
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.name), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := test.NewFakeClient()
			assert.NoError(t, err)

			endpoint := v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kamelet",
					APIVersion: "camel.apache.org/v1any1",
					Name:       "mykamelet",
				},
			}

			if len(tc.endpointProperties) > 0 {
				endpoint.Properties = asEndpointProperties(tc.endpointProperties)
			}

			binding, err := KameletBindingProvider{}.Translate(
				BindingContext{
					Ctx:       ctx,
					Client:    client,
					Namespace: "test",
					Profile:   camelv1.TraitProfileKubernetes,
				},
				EndpointContext{
					Type:     tc.endpointType,
					Position: &i,
				},
				endpoint)

			assert.NoError(t, err)
			assert.NotNil(t, binding)
			assert.Equal(t, tc.step, binding.Step)
			assert.Equal(t, tc.uri, binding.URI)

			if len(tc.applicationProperties) > 0 {
				assert.Equal(t, tc.applicationProperties, binding.ApplicationProperties)
			} else {
				assert.True(t, len(binding.ApplicationProperties) == 0)
			}
		})
	}
}

func TestKameletBindingProviderWithDataTypes(t *testing.T) {
	testcases := []struct {
		name                  string
		endpointType          v1alpha1.EndpointType
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
			endpointType: v1alpha1.EndpointTypeAction,
			uri:          "",
			step: map[string]interface{}{
				"pipeline": map[string]interface{}{
					"id": "action-0-pipeline",
					"steps": []map[string]interface{}{
						{
							"kamelet": map[string]interface{}{
								"name": "data-type-action/action-0-in",
							},
						},
						{
							"kamelet": map[string]interface{}{
								"name": "mykamelet/action-0",
							},
						},
					},
				},
			},
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
			name:         "action-output",
			endpointType: v1alpha1.EndpointTypeAction,
			uri:          "",
			step: map[string]interface{}{
				"pipeline": map[string]interface{}{
					"id": "action-1-pipeline",
					"steps": []map[string]interface{}{
						{
							"kamelet": map[string]interface{}{
								"name": "mykamelet/action-1",
							},
						},
						{
							"kamelet": map[string]interface{}{
								"name": "data-type-action/action-1-out",
							},
						},
					},
				},
			},
			endpointProperties: map[string]string{
				"foo": "bar",
			},
			applicationProperties: map[string]string{
				"camel.kamelet.mykamelet.action-1.foo":               "bar",
				"camel.kamelet.data-type-action.action-1-out.scheme": "camel",
				"camel.kamelet.data-type-action.action-1-out.format": "string",
			},
			outputFormat: "string",
		},
		{
			name:         "action-input-output",
			endpointType: v1alpha1.EndpointTypeAction,
			uri:          "",
			step: map[string]interface{}{
				"pipeline": map[string]interface{}{
					"id": "action-2-pipeline",
					"steps": []map[string]interface{}{
						{
							"kamelet": map[string]interface{}{
								"name": "data-type-action/action-2-in",
							},
						},
						{
							"kamelet": map[string]interface{}{
								"name": "mykamelet/action-2",
							},
						},
						{
							"kamelet": map[string]interface{}{
								"name": "data-type-action/action-2-out",
							},
						},
					},
				},
			},
			endpointProperties: map[string]string{
				"foo": "bar",
			},
			applicationProperties: map[string]string{
				"camel.kamelet.mykamelet.action-2.foo":               "bar",
				"camel.kamelet.data-type-action.action-2-in.scheme":  "foo",
				"camel.kamelet.data-type-action.action-2-in.format":  "string",
				"camel.kamelet.data-type-action.action-2-out.scheme": "foo",
				"camel.kamelet.data-type-action.action-2-out.format": "binary",
			},
			inputScheme:  "foo",
			inputFormat:  "string",
			outputScheme: "foo",
			outputFormat: "binary",
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.name), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := test.NewFakeClient()
			assert.NoError(t, err)

			endpoint := v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kamelet",
					APIVersion: "camel.apache.org/v1any1",
					Name:       "mykamelet",
				},
			}

			if len(tc.endpointProperties) > 0 {
				endpoint.Properties = asEndpointProperties(tc.endpointProperties)
			}

			endpoint.DataTypes = make(map[v1alpha1.TypeSlot]v1alpha1.DataTypeReference)
			if tc.inputFormat != "" {
				endpoint.DataTypes[v1alpha1.TypeSlotIn] = v1alpha1.DataTypeReference{
					Scheme: tc.inputScheme,
					Format: tc.inputFormat,
				}
			}

			if tc.outputFormat != "" {
				endpoint.DataTypes[v1alpha1.TypeSlotOut] = v1alpha1.DataTypeReference{
					Scheme: tc.outputScheme,
					Format: tc.outputFormat,
				}
			}

			binding, err := KameletBindingProvider{}.Translate(
				BindingContext{
					Ctx:       ctx,
					Client:    client,
					Namespace: "test",
					Profile:   camelv1.TraitProfileKubernetes,
				},
				EndpointContext{
					Type:     tc.endpointType,
					Position: &i,
				},
				endpoint)

			assert.NoError(t, err)
			assert.NotNil(t, binding)
			assert.Equal(t, tc.step, binding.Step)
			assert.Equal(t, tc.uri, binding.URI)

			if len(tc.applicationProperties) > 0 {
				assert.Equal(t, tc.applicationProperties, binding.ApplicationProperties)
			} else {
				assert.True(t, len(binding.ApplicationProperties) == 0)
			}
		})
	}
}

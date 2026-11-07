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
	"encoding/json"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindings(t *testing.T) {
	testcases := []struct {
		endpointType v1.EndpointType
		endpoint     v1.Endpoint
		uri          string
		traits       v1.Traits
		props        map[string]string
	}{
		{
			endpointType: v1.EndpointTypeSource,
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kamelet",
					APIVersion: "camel.apache.org/v1any1",
					Name:       "mykamelet",
				},
			},
			uri:   "kamelet:mykamelet/source",
			props: map[string]string{},
		},
		{
			endpointType: v1.EndpointTypeSink,
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kamelet",
					APIVersion: "camel.apache.org/v1any1",
					Name:       "mykamelet",
				},
				Properties: asEndpointProperties(map[string]string{
					"mymessage":   "myval",
					"encodedkey?": "encoded=val",
				}),
			},
			uri: "kamelet:mykamelet/sink",
			props: map[string]string{
				"camel.kamelet.mykamelet.sink.encodedkey?": "encoded=val",
				"camel.kamelet.mykamelet.sink.mymessage":   "myval",
			},
		},
		{
			endpointType: v1.EndpointTypeSink,
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kamelet",
					APIVersion: "camel.apache.org/v1any1",
					Name:       "mykamelet",
				},
				Properties: asEndpointProperties(map[string]string{
					"mymessage":      "myval",
					"kameletVersion": "v1",
				}),
			},
			uri: "kamelet:mykamelet/sink?kameletVersion=v1",
			props: map[string]string{
				"camel.kamelet.mykamelet.sink.mymessage": "myval",
			},
		},
		{
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kamelet",
					APIVersion: "camel.apache.org/v1any1",
					Name:       "mykamelet",
				},
				Properties: asEndpointProperties(map[string]string{
					"id":        "myid?",
					"mymessage": "myval",
				}),
			},
			uri: "kamelet:mykamelet/myid%3F",
			props: map[string]string{
				"camel.kamelet.mykamelet.myid?.mymessage": "myval",
			},
		},
		{
			endpointType: v1.EndpointTypeSink,
			endpoint: v1.Endpoint{
				URI: ptr.To("https://myurl/hey"),
			},
			uri: "https://myurl/hey",
		},
		{
			endpointType: v1.EndpointTypeSink,
			endpoint: v1.Endpoint{
				URI: ptr.To("docker://xxx"),
			},
			uri: "docker://xxx",
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.uri), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := internal.NewFakeClient()
			require.NoError(t, err)

			bindingContext := BindingContext{
				Ctx:       ctx,
				Client:    client,
				Namespace: "test",
			}

			binding, err := Translate(bindingContext, EndpointContext{
				Type: tc.endpointType,
			}, tc.endpoint)
			require.NoError(t, err)
			assert.NotNil(t, binding)
			assert.Equal(t, tc.uri, binding.URI)
			assert.Equal(t, tc.props, binding.ApplicationProperties)
		})
	}
}

func asEndpointProperties(props map[string]string) *v1.EndpointProperties {
	serialized, err := json.Marshal(props)
	if err != nil {
		panic(err)
	}
	return &v1.EndpointProperties{
		RawMessage: serialized,
	}
}

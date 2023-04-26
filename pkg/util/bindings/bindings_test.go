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
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"

	"github.com/apache/camel-k/v2/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

func TestBindings(t *testing.T) {
	testcases := []struct {
		endpointType v1.EndpointType
		endpoint     v1.Endpoint
		profile      v1.TraitProfile
		uri          string
		traits       v1.Traits
		props        map[string]string
	}{
		{
			endpointType: v1.EndpointTypeSink,
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
					Name:       "myservice",
				},
			},
			uri: "knative:endpoint/myservice?apiVersion=serving.knative.dev%2Fv1&kind=Service",
		},
		{
			endpointType: v1.EndpointTypeAction,
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
					Name:       "myservice",
				},
			},
			uri: "knative:endpoint/myservice?apiVersion=serving.knative.dev%2Fv1&kind=Service",
		},
		{
			endpointType: v1.EndpointTypeSink,
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
					Name:       "myservice",
				},
				Properties: asEndpointProperties(map[string]string{
					"ce.override.ce-type": "mytype",
				}),
			},
			uri: "knative:endpoint/myservice?apiVersion=serving.knative.dev%2Fv1&ce.override.ce-type=mytype&kind=Service",
		},
		{
			endpointType: v1.EndpointTypeSink,
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Channel",
					APIVersion: "messaging.knative.dev/v1",
					Name:       "mychannel",
				},
			},
			uri: "knative:channel/mychannel?apiVersion=messaging.knative.dev%2Fv1&kind=Channel",
		},
		{
			endpointType: v1.EndpointTypeSource,
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Channel",
					APIVersion: "messaging.knative.dev/v1",
					Name:       "mychannel",
				},
			},
			uri: "knative:channel/mychannel?apiVersion=messaging.knative.dev%2Fv1&kind=Channel",
		},
		{
			endpointType: v1.EndpointTypeSource,
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "KafkaChannel",
					APIVersion: "messaging.knative.dev/v1beta1",
					Name:       "mychannel",
				},
			},
			uri: "knative:channel/mychannel?apiVersion=messaging.knative.dev%2Fv1beta1&kind=KafkaChannel",
		},
		{
			endpointType: v1.EndpointTypeSource,
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Broker",
					APIVersion: "eventing.knative.dev/v1",
					Name:       "default",
				},
				Properties: asEndpointProperties(map[string]string{
					"type": "myeventtype",
				}),
			},
			uri: "knative:event/myeventtype?apiVersion=eventing.knative.dev%2Fv1&kind=Broker&name=default",
		},
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
				URI: pointer.String("https://myurl/hey"),
				Properties: asEndpointProperties(map[string]string{
					"ce.override.ce-type": "mytype",
				}),
			},
			uri: "knative:endpoint/sink?ce.override.ce-type=mytype",
			traits: v1.Traits{
				Knative: &traitv1.KnativeTrait{
					SinkBinding:   pointer.Bool(false),
					Configuration: asKnativeConfig("https://myurl/hey"),
				},
			},
		},
		{
			endpointType: v1.EndpointTypeSink,
			endpoint: v1.Endpoint{
				URI: pointer.String("https://myurl/hey"),
			},
			profile: v1.TraitProfileKubernetes,
			uri:     "https://myurl/hey",
		},
		{
			endpointType: v1.EndpointTypeSink,
			endpoint: v1.Endpoint{
				URI: pointer.String("docker://xxx"),
			},
			uri: "docker://xxx",
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.uri), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := test.NewFakeClient()
			assert.NoError(t, err)

			profile := tc.profile
			if profile == "" {
				profile = v1.TraitProfileKnative
			}

			bindingContext := BindingContext{
				Ctx:       ctx,
				Client:    client,
				Namespace: "test",
				Profile:   profile,
			}

			binding, err := Translate(bindingContext, EndpointContext{
				Type: tc.endpointType,
			}, tc.endpoint)
			assert.NoError(t, err)
			assert.NotNil(t, binding)
			assert.Equal(t, tc.uri, binding.URI)
			assert.Equal(t, tc.traits, binding.Traits)
			assert.Equal(t, tc.props, binding.ApplicationProperties)
		})
	}
}

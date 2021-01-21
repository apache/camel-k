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
	"net/url"
	"testing"

	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	knativeapis "github.com/apache/camel-k/pkg/apis/camel/v1/knative"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestBindings(t *testing.T) {
	testcases := []struct {
		endpointType v1alpha1.EndpointType
		endpoint     v1alpha1.Endpoint
		profile      camelv1.TraitProfile
		uri          string
		traits       map[string]camelv1.TraitSpec
	}{
		{
			endpointType: v1alpha1.EndpointTypeSink,
			endpoint: v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
					Name:       "myservice",
				},
			},
			uri: "knative:endpoint/myservice?apiVersion=serving.knative.dev%2Fv1&kind=Service",
		},
		{
			endpointType: v1alpha1.EndpointTypeSink,
			endpoint: v1alpha1.Endpoint{
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
			endpointType: v1alpha1.EndpointTypeSink,
			endpoint: v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Channel",
					APIVersion: "messaging.knative.dev/v1",
					Name:       "mychannel",
				},
			},
			uri: "knative:channel/mychannel?apiVersion=messaging.knative.dev%2Fv1&kind=Channel",
		},
		{
			endpointType: v1alpha1.EndpointTypeSource,
			endpoint: v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Channel",
					APIVersion: "messaging.knative.dev/v1",
					Name:       "mychannel",
				},
			},
			uri: "knative:channel/mychannel?apiVersion=messaging.knative.dev%2Fv1&kind=Channel",
		},
		{
			endpointType: v1alpha1.EndpointTypeSource,
			endpoint: v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "KafkaChannel",
					APIVersion: "messaging.knative.dev/v1beta1",
					Name:       "mychannel",
				},
			},
			uri: "knative:channel/mychannel?apiVersion=messaging.knative.dev%2Fv1beta1&kind=KafkaChannel",
		},
		{
			endpointType: v1alpha1.EndpointTypeSource,
			endpoint: v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Broker",
					APIVersion: "eventing.knative.dev/v1beta1",
					Name:       "default",
				},
				Properties: asEndpointProperties(map[string]string{
					"type": "myeventtype",
				}),
			},
			uri: "knative:event/myeventtype?apiVersion=eventing.knative.dev%2Fv1beta1&kind=Broker",
		},
		{
			endpoint: v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kamelet",
					APIVersion: "camel.apache.org/v1any1",
					Name:       "mykamelet",
				},
			},
			uri: "kamelet:mykamelet",
		},
		{
			endpoint: v1alpha1.Endpoint{
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
			uri: "kamelet:mykamelet?encodedkey%3F=encoded%3Dval&mymessage=myval",
		},
		{
			endpoint: v1alpha1.Endpoint{
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
			uri: "kamelet:mykamelet/myid%3F?mymessage=myval",
		},
		{
			endpointType: v1alpha1.EndpointTypeSink,
			endpoint: v1alpha1.Endpoint{
				URI: asStringPointer("https://myurl/hey"),
				Properties: asEndpointProperties(map[string]string{
					"ce.override.ce-type": "mytype",
				}),
			},
			uri: "knative:endpoint/sink?ce.override.ce-type=mytype",
			traits: asTraitSpec("knative", map[string]interface{}{
				"sinkBinding":   false,
				"configuration": asKnativeConfig("https://myurl/hey"),
			}),
		},
		{
			endpointType: v1alpha1.EndpointTypeSink,
			endpoint: v1alpha1.Endpoint{
				URI: asStringPointer("https://myurl/hey"),
			},
			profile: camelv1.TraitProfileKubernetes,
			uri:     "https://myurl/hey",
		},
		{
			endpointType: v1alpha1.EndpointTypeSink,
			endpoint: v1alpha1.Endpoint{
				URI: asStringPointer("docker://xxx"),
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
				profile = camelv1.TraitProfileKnative
			}

			bindingContext := BindingContext{
				Ctx:       ctx,
				Client:    client,
				Namespace: "test",
				Profile:   profile,
			}

			binding, err := Translate(bindingContext, tc.endpointType, tc.endpoint)
			assert.NoError(t, err)
			assert.NotNil(t, binding)
			assert.Equal(t, tc.uri, binding.URI)
			assert.Equal(t, tc.traits, binding.Traits)
		})
	}
}

func asEndpointProperties(props map[string]string) *v1alpha1.EndpointProperties {
	serialized, err := json.Marshal(props)
	if err != nil {
		panic(err)
	}
	return &v1alpha1.EndpointProperties{
		RawMessage: serialized,
	}
}

func asTraitSpec(key string, data map[string]interface{}) map[string]camelv1.TraitSpec {
	res := make(map[string]camelv1.TraitSpec)
	serialized, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	res[key] = camelv1.TraitSpec{
		Configuration: camelv1.TraitConfiguration{
			RawMessage: serialized,
		},
	}
	return res
}

func asStringPointer(str string) *string {
	return &str
}

func asKnativeConfig(endpointURL string) string {
	serviceURL, err := url.Parse(endpointURL)
	if err != nil {
		panic(err)
	}
	def, err := knativeapis.BuildCamelServiceDefinition("sink", knativeapis.CamelEndpointKindSink, knativeapis.CamelServiceTypeEndpoint, *serviceURL, "", "")
	if err != nil {
		panic(err)
	}
	env := knativeapis.NewCamelEnvironment()
	env.Services = append(env.Services, def)
	serialized, err := json.Marshal(env)
	if err != nil {
		panic(err)
	}
	return string(serialized)
}

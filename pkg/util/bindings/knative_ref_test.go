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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/test"
)

func TestKnativeRefAsSource(t *testing.T) {
	testcases := []struct {
		name            string
		endpoint        camelv1.Endpoint
		uri             string
		filters         []string
		filterEventType *bool
	}{
		{
			name: "broker",
			endpoint: camelv1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Broker",
					Name:       "default",
					APIVersion: "eventing.knative.dev/v1",
				},
			},
			uri:             "knative:event?apiVersion=eventing.knative.dev%2Fv1&kind=Broker&name=default",
			filterEventType: ptr.To(false),
		},
		{
			name: "broker-name-property",
			endpoint: camelv1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Broker",
					Name:       "default",
					APIVersion: "eventing.knative.dev/v1",
				},
				Properties: asEndpointProperties(map[string]string{"name": "my-broker"}),
			},
			uri:             "knative:event?apiVersion=eventing.knative.dev%2Fv1&kind=Broker&name=my-broker",
			filterEventType: ptr.To(false),
		},
		{
			name: "event-type-filter",
			endpoint: camelv1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Broker",
					Name:       "default",
					APIVersion: "eventing.knative.dev/v1",
				},
				Properties: asEndpointProperties(map[string]string{"type": "org.apache.camel.myevent"}),
			},
			uri:     "knative:event/org.apache.camel.myevent?apiVersion=eventing.knative.dev%2Fv1&kind=Broker&name=default",
			filters: []string{"type=org.apache.camel.myevent"},
		},
		{
			name: "cloud-events-type-filter",
			endpoint: camelv1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Broker",
					Name:       "default",
					APIVersion: "eventing.knative.dev/v1",
				},
				Properties: asEndpointProperties(map[string]string{"cloudEventsType": "org.apache.camel.cloudevent"}),
			},
			uri:     "knative:event/org.apache.camel.cloudevent?apiVersion=eventing.knative.dev%2Fv1&cloudEventsType=org.apache.camel.cloudevent&kind=Broker&name=default",
			filters: []string{"type=org.apache.camel.cloudevent"},
		},
		{
			name: "event-filters",
			endpoint: camelv1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Broker",
					Name:       "default",
					APIVersion: "eventing.knative.dev/v1",
				},
				Properties: asEndpointProperties(map[string]string{"source": "my-source", "subject": "mySubject"}),
			},
			uri:             "knative:event?apiVersion=eventing.knative.dev%2Fv1&kind=Broker&name=default",
			filters:         []string{"source=my-source", "subject=mySubject"},
			filterEventType: ptr.To(false),
		},
		{
			name: "event-extension-filters",
			endpoint: camelv1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Broker",
					Name:       "default",
					APIVersion: "eventing.knative.dev/v1",
				},
				Properties: asEndpointProperties(map[string]string{"myextension": "foo"}),
			},
			uri:             "knative:event?apiVersion=eventing.knative.dev%2Fv1&kind=Broker&name=default",
			filters:         []string{"myextension=foo"},
			filterEventType: ptr.To(false),
		},
		{
			name: "channel",
			endpoint: camelv1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Channel",
					Name:       "mychannel",
					APIVersion: "messaging.knative.dev/v1",
				},
			},
			uri: "knative:channel/mychannel?apiVersion=messaging.knative.dev%2Fv1&kind=Channel",
		},
		{
			name: "service",
			endpoint: camelv1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Service",
					Name:       "myservice",
					APIVersion: "serving.knative.dev/v1",
				},
			},
			uri: "knative:endpoint/myservice?apiVersion=serving.knative.dev%2Fv1&kind=Service",
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.name), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := test.NewFakeClient()
			require.NoError(t, err)

			bindingContext := BindingContext{
				Ctx:       ctx,
				Client:    client,
				Namespace: "test",
				Profile:   camelv1.TraitProfileKnative,
			}

			binding, err := KnativeRefBindingProvider{}.Translate(bindingContext, EndpointContext{
				Type: camelv1.EndpointTypeSource,
			}, tc.endpoint)
			require.NoError(t, err)
			assert.NotNil(t, binding)
			assert.Equal(t, tc.uri, binding.URI)

			if tc.filters != nil || !ptr.Deref(tc.filterEventType, true) {
				assert.NotNil(t, binding.Traits.Knative)
				assert.Len(t, binding.Traits.Knative.Filters, len(tc.filters))

				for _, filterExpression := range tc.filters {
					assert.Contains(t, binding.Traits.Knative.Filters, filterExpression)
				}

				assert.Equal(t, ptr.Deref(binding.Traits.Knative.FilterEventType, true), ptr.Deref(tc.filterEventType, true))
			}
		})
	}
}

func TestKnativeRefAsSink(t *testing.T) {
	testcases := []struct {
		name     string
		endpoint camelv1.Endpoint
		uri      string
	}{
		{
			name: "broker",
			endpoint: camelv1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Broker",
					Name:       "default",
					APIVersion: "eventing.knative.dev/v1",
				},
			},
			uri: "knative:event?apiVersion=eventing.knative.dev%2Fv1&kind=Broker&name=default",
		},
		{
			name: "broker-name-property",
			endpoint: camelv1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Broker",
					Name:       "default",
					APIVersion: "eventing.knative.dev/v1",
				},
				Properties: asEndpointProperties(map[string]string{"name": "my-broker"}),
			},
			uri: "knative:event?apiVersion=eventing.knative.dev%2Fv1&kind=Broker&name=my-broker",
		},
		{
			name: "event-type",
			endpoint: camelv1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Broker",
					Name:       "default",
					APIVersion: "eventing.knative.dev/v1",
				},
				Properties: asEndpointProperties(map[string]string{"type": "org.apache.camel.myevent"}),
			},
			uri: "knative:event/org.apache.camel.myevent?apiVersion=eventing.knative.dev%2Fv1&kind=Broker&name=default",
		},
		{
			name: "channel",
			endpoint: camelv1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Channel",
					Name:       "mychannel",
					APIVersion: "messaging.knative.dev/v1",
				},
			},
			uri: "knative:channel/mychannel?apiVersion=messaging.knative.dev%2Fv1&kind=Channel",
		},
		{
			name: "service",
			endpoint: camelv1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Service",
					Name:       "myservice",
					APIVersion: "serving.knative.dev/v1",
				},
			},
			uri: "knative:endpoint/myservice?apiVersion=serving.knative.dev%2Fv1&kind=Service",
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.name), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := test.NewFakeClient()
			require.NoError(t, err)

			bindingContext := BindingContext{
				Ctx:       ctx,
				Client:    client,
				Namespace: "test",
				Profile:   camelv1.TraitProfileKnative,
			}

			binding, err := KnativeRefBindingProvider{}.Translate(bindingContext, EndpointContext{
				Type: camelv1.EndpointTypeSink,
			}, tc.endpoint)
			require.NoError(t, err)
			assert.NotNil(t, binding)
			assert.Equal(t, tc.uri, binding.URI)
			assert.Equal(t, camelv1.Traits{}, binding.Traits)
		})
	}
}

func TestUnsupportedKnativeResource(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := test.NewFakeClient()
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKnative,
	}

	endpoint := camelv1.Endpoint{
		Ref: &v1.ObjectReference{
			Kind:       "Unknown",
			Name:       "default",
			APIVersion: "eventing.knative.dev/v1",
		},
	}

	binding, err := KnativeRefBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	assert.Nil(t, binding)
}

func TestKnativeNotInstalled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := test.NewFakeClient()
	require.NoError(t, err)

	// disable the knative eventing api
	fakeClient := client.(*test.FakeClient) //nolint
	fakeClient.DisableAPIGroupDiscovery("eventing.knative.dev/v1")

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKnative,
	}

	endpoint := camelv1.Endpoint{
		Ref: &v1.ObjectReference{
			Kind:       "Broker",
			Name:       "default",
			APIVersion: "eventing.knative.dev/v1",
		},
	}

	binding, err := KnativeRefBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.Error(t, err, "integration referencing Knative endpoint 'default' that cannot run, because Knative is not installed on the cluster")
	assert.Nil(t, binding)
}

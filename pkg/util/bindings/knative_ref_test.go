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

	"github.com/apache/camel-k/pkg/util/test"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"

	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

func TestKnativeRefBinding(t *testing.T) {
	testcases := []struct {
		endpoint v1alpha1.Endpoint
		uri      string
	}{
		{
			endpoint: v1alpha1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Broker",
					Name:       "default",
					APIVersion: "eventing.knative.dev/v1",
				},
			},
			uri: "knative:event?apiVersion=eventing.knative.dev%2Fv1&kind=Broker&name=default",
		},
		{
			endpoint: v1alpha1.Endpoint{
				Ref: &v1.ObjectReference{
					Kind:       "Channel",
					Name:       "mychannel",
					APIVersion: "messaging.knative.dev/v1",
				},
			},
			uri: "knative:channel/mychannel?apiVersion=messaging.knative.dev%2Fv1&kind=Channel",
		},
		{
			endpoint: v1alpha1.Endpoint{
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
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.endpoint.Ref.Kind), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := test.NewFakeClient()
			assert.NoError(t, err)

			bindingContext := BindingContext{
				Ctx:       ctx,
				Client:    client,
				Namespace: "test",
				Profile:   camelv1.TraitProfileKnative,
			}

			binding, err := KnativeRefBindingProvider{}.Translate(bindingContext, EndpointContext{
				Type: v1alpha1.EndpointTypeSink,
			}, tc.endpoint)
			assert.NoError(t, err)
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
	assert.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKnative,
	}

	endpoint := v1alpha1.Endpoint{
		Ref: &v1.ObjectReference{
			Kind:       "Unknown",
			Name:       "default",
			APIVersion: "eventing.knative.dev/v1",
		},
	}

	binding, err := KnativeRefBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: v1alpha1.EndpointTypeSink,
	}, endpoint)
	assert.NoError(t, err)
	assert.Nil(t, binding)
}

func TestKnativeNotInstalled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := test.NewFakeClient()
	assert.NoError(t, err)

	// disable the knative service api
	fakeClient := client.(*test.FakeClient) //nolint
	fakeClient.DisableAPIGroupDiscovery("serving.knative.dev/v1")

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKnative,
	}

	endpoint := v1alpha1.Endpoint{
		Ref: &v1.ObjectReference{
			Kind:       "Broker",
			Name:       "default",
			APIVersion: "eventing.knative.dev/v1",
		},
	}

	binding, err := KnativeRefBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: v1alpha1.EndpointTypeSink,
	}, endpoint)
	assert.Error(t, err, "integration referencing Knative endpoint 'default' that cannot run, because Knative is not installed on the cluster")
	assert.Nil(t, binding)
}

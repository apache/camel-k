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
	"testing"

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestKnativeURI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKnative,
	}

	endpoint := camelv1.Endpoint{
		URI: ptr.To("http://my-domain"),
	}

	binding, err := KnativeURIBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	require.NotNil(t, binding)
	assert.Equal(t, "knative:endpoint/sink", binding.URI)
	require.NotNil(t, binding.Traits.Knative)
	assert.Equal(t,
		`{"services":[{"type":"endpoint","name":"sink","url":"http://my-domain","metadata":{"camel.endpoint.kind":"sink","knative.apiVersion":"","knative.kind":"","knative.name":"sink"}}]}`,
		binding.Traits.Knative.Configuration,
	)
}

func TestKnativeURISkip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	endpoint := camelv1.Endpoint{
		// Missing URI
	}

	binding, err := KnativeURIBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	require.Nil(t, binding)

	endpoint.URI = ptr.To("non-http:my")
	binding, err = KnativeURIBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	require.Nil(t, binding)

	endpoint.URI = ptr.To("http:my")
	binding, err = KnativeURIBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSource,
	}, endpoint)
	require.NoError(t, err)
	require.Nil(t, binding)
}

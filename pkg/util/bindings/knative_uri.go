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
	"net/url"
	"strings"

	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	knativeapis "github.com/apache/camel-k/pkg/apis/camel/v1/knative"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/uri"
)

// KnativeURIBindingProvider converts a HTTP/HTTPS URI into a Camel Knative endpoint (to call it via CloudEvents).
type KnativeURIBindingProvider struct{}

func (k KnativeURIBindingProvider) ID() string {
	return "knative-uri"
}

func (k KnativeURIBindingProvider) Translate(ctx BindingContext, endpointCtx EndpointContext, e v1alpha1.Endpoint) (*Binding, error) {
	if e.URI == nil {
		// works only on uris
		return nil, nil
	}
	if ctx.Profile != v1.TraitProfileKnative {
		// use cloudevent binding only in Knative trait profile
		return nil, nil
	}
	if !strings.HasPrefix(*e.URI, "http:") && !strings.HasPrefix(*e.URI, "https:") {
		// only translates http/https uri to Knative calls
		return nil, nil
	}
	if endpointCtx.Type == v1alpha1.EndpointTypeSource {
		// HTTP/HTTPS uri are translated to Knative endpoints only when used as sinks
		return nil, nil
	}

	originalURI, err := url.Parse(*e.URI)
	if err != nil {
		return nil, err
	}
	env := knativeapis.NewCamelEnvironment()
	svc, err := knativeapis.BuildCamelServiceDefinition("sink",
		knativeapis.CamelEndpointKindSink,
		knativeapis.CamelServiceTypeEndpoint,
		*originalURI, "", "")
	if err != nil {
		return nil, err
	}
	env.Services = append(env.Services, svc)
	config, err := env.Serialize()
	if err != nil {
		return nil, err
	}

	// Rewrite URI to match the service definition
	serviceURI := "knative:endpoint/sink"
	props, err := e.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}
	serviceURI = uri.AppendParameters(serviceURI, props)

	return &Binding{
		URI: serviceURI,
		Traits: v1.Traits{
			Knative: &traitv1.KnativeTrait{
				Configuration: config,
				SinkBinding:   pointer.Bool(false),
			},
		},
	}, nil
}

func (k KnativeURIBindingProvider) Order() int {
	return OrderStandard
}

func init() {
	RegisterBindingProvider(KnativeURIBindingProvider{})
}

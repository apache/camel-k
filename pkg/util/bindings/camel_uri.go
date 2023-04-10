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
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	v1alpha1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/v2/pkg/util/uri"
)

// CamelURIBindingProvider converts an explicit URI into a Camel endpoint.
// It's used as fallback if the URI scheme is not known by other providers.
type CamelURIBindingProvider struct{}

// ID --.
func (k CamelURIBindingProvider) ID() string {
	return "camel-uri"
}

// Translate --.
func (k CamelURIBindingProvider) Translate(ctx BindingContext, endpointCtx EndpointContext, e v1.Endpoint) (*Binding, error) {
	if e.URI == nil {
		// works only on uris
		return nil, nil
	}

	endpointURI := *e.URI
	props, err := e.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}
	endpointURI = uri.AppendParameters(endpointURI, props)

	return &Binding{
		URI: endpointURI,
	}, nil
}

// Order --.
func (k CamelURIBindingProvider) Order() int {
	// Using it as fallback
	return OrderLast
}

// V1alpha1CamelURIBindingProvider --.
// Deprecated .
type V1alpha1CamelURIBindingProvider struct{}

// ID --.
// Deprecated .
func (k V1alpha1CamelURIBindingProvider) ID() string {
	return "camel-uri"
}

// Translate --.
// Deprecated .
func (k V1alpha1CamelURIBindingProvider) Translate(ctx V1alpha1BindingContext, endpointCtx V1alpha1EndpointContext, e v1alpha1.Endpoint) (*Binding, error) {
	if e.URI == nil {
		// works only on uris
		return nil, nil
	}

	endpointURI := *e.URI
	props, err := e.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}
	endpointURI = uri.AppendParameters(endpointURI, props)

	return &Binding{
		URI: endpointURI,
	}, nil
}

// Order --
// Deprecated .
func (k V1alpha1CamelURIBindingProvider) Order() int {
	// Using it as fallback
	return OrderLast
}

func init() {
	RegisterBindingProvider(CamelURIBindingProvider{})
	V1alpha1RegisterBindingProvider(V1alpha1CamelURIBindingProvider{})
}

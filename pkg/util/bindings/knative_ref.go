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
	"errors"
	"fmt"
	"net/url"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	knativeapis "github.com/apache/camel-k/v2/pkg/apis/camel/v1/knative"
	v1alpha1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1alpha1"

	"github.com/apache/camel-k/v2/pkg/util/knative"
	"github.com/apache/camel-k/v2/pkg/util/uri"
)

// KnativeRefBindingProvider converts a reference to a Kubernetes object into a Camel URI.
// It's used as fallback if no other providers can decode the object reference.
type KnativeRefBindingProvider struct{}

// ID --.
func (k KnativeRefBindingProvider) ID() string {
	return "knative-ref"
}

// Translate --.
func (k KnativeRefBindingProvider) Translate(ctx BindingContext, endpointCtx EndpointContext, e v1.Endpoint) (*Binding, error) {
	if e.Ref == nil {
		// works only on refs
		return nil, nil
	}

	serviceType, err := knative.GetServiceType(*e.Ref)
	if err != nil {
		return nil, err
	}

	if serviceType == nil {
		endpointType := knativeapis.CamelServiceTypeEndpoint
		serviceType = &endpointType
	}

	props, err := e.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}
	if props == nil {
		props = make(map[string]string)
	}
	if props["apiVersion"] == "" {
		props["apiVersion"] = e.Ref.APIVersion
	}
	if props["kind"] == "" {
		props["kind"] = e.Ref.Kind
	}

	var serviceURI string
	if *serviceType == knativeapis.CamelServiceTypeEvent {
		if props["name"] == "" {
			props["name"] = e.Ref.Name
		}
		if eventType, ok := props["type"]; ok {
			// consume prop
			delete(props, "type")
			serviceURI = fmt.Sprintf("knative:%s/%s", *serviceType, eventType)
		} else {
			if endpointCtx.Type == v1.EndpointTypeSink || endpointCtx.Type == v1.EndpointTypeAction {
				// Allowing no event type, but it can fail. See https://github.com/apache/camel-k/v2-runtime/issues/536
				serviceURI = fmt.Sprintf("knative:%s", *serviceType)
			} else {
				return nil, errors.New(`property "type" must be provided when reading from the Broker`)
			}
		}
	} else {
		serviceURI = fmt.Sprintf("knative:%s/%s", *serviceType, url.PathEscape(e.Ref.Name))
	}

	serviceURI = uri.AppendParameters(serviceURI, props)
	return &Binding{
		URI: serviceURI,
	}, nil
}

// Order --.
func (k KnativeRefBindingProvider) Order() int {
	// Executes as last, as it can be used as fallback for all unknown object references
	return OrderLast
}

// V1alpha1KnativeRefBindingProvider converts a reference to a Kubernetes object into a Camel URI.
// It's used as fallback if no other providers can decode the object reference.
// Deprecated.
type V1alpha1KnativeRefBindingProvider struct{}

// ID --.
// Deprecated.
func (k V1alpha1KnativeRefBindingProvider) ID() string {
	return "knative-ref"
}

// Translate --.
// Deprecated.
func (k V1alpha1KnativeRefBindingProvider) Translate(ctx V1alpha1BindingContext, endpointCtx V1alpha1EndpointContext, e v1alpha1.Endpoint) (*Binding, error) {
	if e.Ref == nil {
		// works only on refs
		return nil, nil
	}

	serviceType, err := knative.GetServiceType(*e.Ref)
	if err != nil {
		return nil, err
	}

	if serviceType == nil {
		endpointType := knativeapis.CamelServiceTypeEndpoint
		serviceType = &endpointType
	}

	props, err := e.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}
	if props == nil {
		props = make(map[string]string)
	}
	if props["apiVersion"] == "" {
		props["apiVersion"] = e.Ref.APIVersion
	}
	if props["kind"] == "" {
		props["kind"] = e.Ref.Kind
	}

	var serviceURI string
	if *serviceType == knativeapis.CamelServiceTypeEvent {
		if props["name"] == "" {
			props["name"] = e.Ref.Name
		}
		if eventType, ok := props["type"]; ok {
			// consume prop
			delete(props, "type")
			serviceURI = fmt.Sprintf("knative:%s/%s", *serviceType, eventType)
		} else {
			if endpointCtx.Type == v1alpha1.EndpointTypeSink || endpointCtx.Type == v1alpha1.EndpointTypeAction {
				// Allowing no event type, but it can fail. See https://github.com/apache/camel-k/v2-runtime/issues/536
				serviceURI = fmt.Sprintf("knative:%s", *serviceType)
			} else {
				return nil, errors.New(`property "type" must be provided when reading from the Broker`)
			}
		}
	} else {
		serviceURI = fmt.Sprintf("knative:%s/%s", *serviceType, url.PathEscape(e.Ref.Name))
	}

	serviceURI = uri.AppendParameters(serviceURI, props)
	return &Binding{
		URI: serviceURI,
	}, nil
}

// Order --.
// Deprecated.
func (k V1alpha1KnativeRefBindingProvider) Order() int {
	// Executes as last, as it can be used as fallback for all unknown object references
	return OrderLast
}

func init() {
	RegisterBindingProvider(KnativeRefBindingProvider{})
	V1alpha1RegisterBindingProvider(V1alpha1KnativeRefBindingProvider{})
}

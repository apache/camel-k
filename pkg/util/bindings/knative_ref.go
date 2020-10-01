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
	"encoding/json"
	"fmt"
	"net/url"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	knativeapis "github.com/apache/camel-k/pkg/apis/camel/v1/knative"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/knative"
	"github.com/apache/camel-k/pkg/util/uri"
)

// KnativeRefBindingProvider converts a reference to a Kubernetes object into a Camel URI.
// It's used as fallback if no other providers can decode the object reference.
type KnativeRefBindingProvider struct{}

func (k KnativeRefBindingProvider) ID() string {
	return "knative-ref"
}

func (k KnativeRefBindingProvider) Translate(endpointType v1alpha1.EndpointType, e v1alpha1.Endpoint) (*Binding, error) {
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
		if eventType, ok := props["type"]; ok {
			// consume prop
			delete(props, "type")
			serviceURI = fmt.Sprintf("knative:%s/%s", *serviceType, eventType)
		} else {
			serviceURI = fmt.Sprintf("knative:%s", *serviceType)
		}
	} else {
		serviceURI = fmt.Sprintf("knative:%s/%s", *serviceType, url.PathEscape(e.Ref.Name))
	}

	serviceURI = uri.AppendParameters(serviceURI, props)

	var traits map[string]v1.TraitSpec
	if endpointType == v1alpha1.EndpointTypeSink {
		knativeConfig := make(map[string]interface{})
		// TODO remove this after making sinkbinding the default (https://github.com/apache/camel-k/issues/1654)
		knativeConfig["sinkBinding"] = true
		knativeConfigJSON, err := json.Marshal(knativeConfig)
		if err != nil {
			return nil, err
		}
		traits = map[string]v1.TraitSpec{
			"knative": {
				Configuration: v1.TraitConfiguration{
					RawMessage: knativeConfigJSON,
				},
			},
		}
	}

	return &Binding{
		URI:    serviceURI,
		Traits: traits,
	}, nil
}

func (k KnativeRefBindingProvider) Order() int {
	// Executes as last, as it can be used as fallback for all unknown object references
	return OrderLast
}

func init() {
	RegisterBindingProvider(KnativeRefBindingProvider{})
}

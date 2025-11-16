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
	"fmt"
	"net/url"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	knativeapis "github.com/apache/camel-k/v2/pkg/internal/knative"
	"github.com/apache/camel-k/v2/pkg/util/property"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

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

	if ok, err := isKnownKnativeResource(e.Ref); !ok {
		// only operates on known Knative endpoint resources (e.g. channels, brokers)
		return nil, err
	}

	if refInstalled, _ := knative.IsRefKindInstalled(ctx.Client, *e.Ref); !refInstalled {
		// works only when Knative specific API Kind is installed
		return nil, fmt.Errorf("integration referencing Knative endpoint '%s' that cannot run, "+
			"because Knative is not installed on the cluster", e.Ref.Name)
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

	var filterEventType = true
	var filterExpressions = make([]string, 0)
	var serviceURI string

	// TODO: refactor
	//nolint:nestif
	if *serviceType == knativeapis.CamelServiceTypeEvent {
		if props["name"] == "" {
			props["name"] = e.Ref.Name
		}

		if endpointCtx.Type == v1.EndpointTypeSource {
			// Configure trigger filter attributes for the Knative event source
			for key, value := range props {
				if key == "cloudEventsType" {
					// cloudEventsType is a synonym for type filter attribute
					filterExpressions = append(filterExpressions, "type="+value)
				} else if key != "name" {
					filterExpressions = append(filterExpressions, fmt.Sprintf("%s=%s", key, value))
				}
			}
		}

		if eventType, ok := props["type"]; ok {
			// consume the type property and set it as URI path parameter
			delete(props, "type")
			serviceURI = fmt.Sprintf("knative:%s/%s", *serviceType, eventType)
		} else if cloudEventsType, found := props["cloudEventsType"]; found && endpointCtx.Type == v1.EndpointTypeSource {
			// set the cloud events type as URI path parameter, but keep it also as URI query param
			serviceURI = fmt.Sprintf("knative:%s/%s", *serviceType, cloudEventsType)
		} else {
			filterEventType = false
			serviceURI = fmt.Sprintf("knative:%s", *serviceType)
		}
	} else {
		serviceURI = fmt.Sprintf("knative:%s/%s", *serviceType, url.PathEscape(e.Ref.Name))
	}

	// Remove filter attributes from props to avoid adding them to the service URI query params
	for _, exp := range filterExpressions {
		key, _ := property.SplitPropertyFileEntry(exp)
		delete(props, key)
	}

	// Enrich service URI query params if not set
	if props["apiVersion"] == "" {
		props["apiVersion"] = e.Ref.APIVersion
	}
	if props["kind"] == "" {
		props["kind"] = e.Ref.Kind
	}

	serviceURI = uri.AppendParameters(serviceURI, props)
	var binding = Binding{
		URI: serviceURI,
	}

	if endpointCtx.Type == v1.EndpointTypeSource && (len(filterExpressions) > 0 || !filterEventType) {
		binding.Traits = v1.Traits{
			Knative: &trait.KnativeTrait{
				Filters:         filterExpressions,
				FilterEventType: &filterEventType,
			},
		}
	}

	return &binding, nil
}

func isKnownKnativeResource(ref *corev1.ObjectReference) (bool, error) {
	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return false, err
	}

	for _, endpoint := range knative.KnownEndpointKinds {
		if endpoint.Group == gv.Group && endpoint.Kind == ref.Kind {
			return true, nil
		}
	}

	return false, nil
}

// Order --.
func (k KnativeRefBindingProvider) Order() int {
	// Executes as last, as it can be used as fallback for all unknown object references
	return OrderLast
}

func init() {
	RegisterBindingProvider(KnativeRefBindingProvider{})
}

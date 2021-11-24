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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// KameletBindingProvider converts a reference to a Kamelet into a Camel URI.
type KameletBindingProvider struct{}

func (k KameletBindingProvider) ID() string {
	return "kamelet"
}

func (k KameletBindingProvider) Translate(ctx BindingContext, endpointCtx EndpointContext, e v1alpha1.Endpoint) (*Binding, error) {
	if e.Ref == nil {
		// works only on refs
		return nil, nil
	}
	gv, err := schema.ParseGroupVersion(e.Ref.APIVersion)
	if err != nil {
		return nil, err
	}
	// it translates only Kamelet refs
	if e.Ref.Kind == v1alpha1.KameletKind && gv.Group == v1alpha1.SchemeGroupVersion.Group {
		kameletName := url.PathEscape(e.Ref.Name)

		props, err := e.Properties.GetPropertyMap()
		if err != nil {
			return nil, err
		}

		id, idPresent := props[v1alpha1.KameletIDProperty]
		if idPresent {
			delete(props, v1alpha1.KameletIDProperty)
		} else {
			id = endpointCtx.GenerateID()
		}

		binding := Binding{}
		if endpointCtx.Type == v1alpha1.EndpointTypeAction {
			binding.Step = map[string]interface{}{
				"kamelet": map[string]interface{}{
					"name": fmt.Sprintf("%s/%s", kameletName, url.PathEscape(id)),
				},
			}
		} else {
			binding.URI = fmt.Sprintf("kamelet:%s/%s", kameletName, url.PathEscape(id))
		}

		if len(props) > 0 {
			binding.ApplicationProperties = make(map[string]string, len(props))
			for k, v := range props {
				propKey := fmt.Sprintf("camel.kamelet.%s.%s.%s", kameletName, id, k)
				binding.ApplicationProperties[propKey] = v
			}
		}

		return &binding, nil
	}
	return nil, nil
}

func (k KameletBindingProvider) Order() int {
	return OrderStandard
}

func init() {
	RegisterBindingProvider(KameletBindingProvider{})
}

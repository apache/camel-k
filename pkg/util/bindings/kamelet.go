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
	v1alpha1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	datTypeActionKamelet = "data-type-action"
)

// BindingConverter converts a reference to a Kamelet into a Camel URI.
type BindingConverter struct{}

// ID --.
func (k BindingConverter) ID() string {
	return "kamelet"
}

// Translate --.
func (k BindingConverter) Translate(ctx BindingContext, endpointCtx EndpointContext, e v1.Endpoint) (*Binding, error) {
	if e.Ref == nil {
		// works only on refs
		return nil, nil
	}
	gv, err := schema.ParseGroupVersion(e.Ref.APIVersion)
	if err != nil {
		return nil, err
	}
	// it translates only Kamelet refs
	if e.Ref.Kind == v1.KameletKind && gv.Group == v1.SchemeGroupVersion.Group {
		kameletName := url.PathEscape(e.Ref.Name)

		props, err := e.Properties.GetPropertyMap()
		if err != nil {
			return nil, err
		}

		id, idPresent := props[v1.KameletIDProperty]
		if idPresent {
			delete(props, v1.KameletIDProperty)
		} else {
			id = endpointCtx.GenerateID()
		}

		binding := Binding{}
		binding.ApplicationProperties = make(map[string]string)
		for k, v := range props {
			propKey := fmt.Sprintf("camel.kamelet.%s.%s.%s", kameletName, id, k)
			binding.ApplicationProperties[propKey] = v
		}

		switch endpointCtx.Type {
		case v1.EndpointTypeAction:
			steps := make([]map[string]interface{}, 0)

			if in, applicationProperties := k.DataTypeStep(e, id, v1.TypeSlotIn); in != nil {
				steps = append(steps, in)
				for k, v := range applicationProperties {
					binding.ApplicationProperties[k] = v
				}
			}

			steps = append(steps, map[string]interface{}{
				"kamelet": map[string]interface{}{
					"name": fmt.Sprintf("%s/%s", kameletName, url.PathEscape(id)),
				},
			})

			if out, applicationProperties := k.DataTypeStep(e, id, v1.TypeSlotOut); out != nil {
				steps = append(steps, out)
				for k, v := range applicationProperties {
					binding.ApplicationProperties[k] = v
				}
			}

			if len(steps) > 1 {
				binding.Step = map[string]interface{}{
					"pipeline": map[string]interface{}{
						"id":    fmt.Sprintf("%s-pipeline", id),
						"steps": steps,
					},
				}
			} else if len(steps) == 1 {
				// nolint: gosec // steps[0] not out of bounds
				binding.Step = steps[0]
			}
		case v1.EndpointTypeSource:
			if out, applicationProperties := k.DataTypeStep(e, id, v1.TypeSlotOut); out != nil {
				binding.Step = out
				for k, v := range applicationProperties {
					binding.ApplicationProperties[k] = v
				}
			}

			binding.URI = fmt.Sprintf("kamelet:%s/%s", kameletName, url.PathEscape(id))
		case v1.EndpointTypeSink:
			if in, applicationProperties := k.DataTypeStep(e, id, v1.TypeSlotIn); in != nil {
				binding.Step = in
				for k, v := range applicationProperties {
					binding.ApplicationProperties[k] = v
				}
			}

			binding.URI = fmt.Sprintf("kamelet:%s/%s", kameletName, url.PathEscape(id))
		default:
			binding.URI = fmt.Sprintf("kamelet:%s/%s", kameletName, url.PathEscape(id))
		}

		return &binding, nil
	}
	return nil, nil
}

// DataTypeStep --.
func (k BindingConverter) DataTypeStep(e v1.Endpoint, id string, typeSlot v1.TypeSlot) (map[string]interface{}, map[string]string) {
	if e.DataTypes == nil {
		return nil, nil
	}

	if inDataType, ok := e.DataTypes[typeSlot]; ok {
		scheme := "camel"
		if inDataType.Scheme != "" {
			scheme = inDataType.Scheme
		}

		props := make(map[string]string, 2)
		props[fmt.Sprintf("camel.kamelet.%s.%s-%s.scheme", datTypeActionKamelet, id, typeSlot)] = scheme
		props[fmt.Sprintf("camel.kamelet.%s.%s-%s.format", datTypeActionKamelet, id, typeSlot)] = inDataType.Format

		stepDsl := map[string]interface{}{
			"kamelet": map[string]interface{}{
				"name": fmt.Sprintf("%s/%s-%s", datTypeActionKamelet, url.PathEscape(id), typeSlot),
			},
		}

		return stepDsl, props
	}

	return nil, nil
}

// Order --.
func (k BindingConverter) Order() int {
	return OrderStandard
}

// V1alpha1BindingConverter converts a reference to a Kamelet into a Camel URI.
// Deprecated.
type V1alpha1BindingConverter struct{}

// ID -- .
// Deprecated.
func (k V1alpha1BindingConverter) ID() string {
	return "kamelet"
}

// Translate -- .
// Deprecated.
func (k V1alpha1BindingConverter) Translate(ctx V1alpha1BindingContext, endpointCtx V1alpha1EndpointContext, e v1alpha1.Endpoint) (*Binding, error) {
	if e.Ref == nil {
		// works only on refs
		return nil, nil
	}
	gv, err := schema.ParseGroupVersion(e.Ref.APIVersion)
	if err != nil {
		return nil, err
	}
	// it translates only Kamelet refs
	if e.Ref.Kind == v1.KameletKind && gv.Group == v1.SchemeGroupVersion.Group {
		kameletName := url.PathEscape(e.Ref.Name)

		props, err := e.Properties.GetPropertyMap()
		if err != nil {
			return nil, err
		}

		id, idPresent := props[v1.KameletIDProperty]
		if idPresent {
			delete(props, v1.KameletIDProperty)
		} else {
			id = endpointCtx.GenerateID()
		}

		binding := Binding{}
		binding.ApplicationProperties = make(map[string]string)
		for k, v := range props {
			propKey := fmt.Sprintf("camel.kamelet.%s.%s.%s", kameletName, id, k)
			binding.ApplicationProperties[propKey] = v
		}

		switch endpointCtx.Type {
		case v1alpha1.EndpointTypeAction:
			steps := make([]map[string]interface{}, 0)

			if in, applicationProperties := k.DataTypeStep(e, id, v1alpha1.TypeSlotIn); in != nil {
				steps = append(steps, in)
				for k, v := range applicationProperties {
					binding.ApplicationProperties[k] = v
				}
			}

			steps = append(steps, map[string]interface{}{
				"kamelet": map[string]interface{}{
					"name": fmt.Sprintf("%s/%s", kameletName, url.PathEscape(id)),
				},
			})

			if out, applicationProperties := k.DataTypeStep(e, id, v1alpha1.TypeSlotOut); out != nil {
				steps = append(steps, out)
				for k, v := range applicationProperties {
					binding.ApplicationProperties[k] = v
				}
			}

			if len(steps) > 1 {
				binding.Step = map[string]interface{}{
					"pipeline": map[string]interface{}{
						"id":    fmt.Sprintf("%s-pipeline", id),
						"steps": steps,
					},
				}
			} else if len(steps) == 1 {
				// nolint: gosec // steps[0] not out of bounds
				binding.Step = steps[0]
			}
		case v1alpha1.EndpointTypeSource:
			if out, applicationProperties := k.DataTypeStep(e, id, v1alpha1.TypeSlotOut); out != nil {
				binding.Step = out
				for k, v := range applicationProperties {
					binding.ApplicationProperties[k] = v
				}
			}

			binding.URI = fmt.Sprintf("kamelet:%s/%s", kameletName, url.PathEscape(id))
		case v1alpha1.EndpointTypeSink:
			if in, applicationProperties := k.DataTypeStep(e, id, v1alpha1.TypeSlotIn); in != nil {
				binding.Step = in
				for k, v := range applicationProperties {
					binding.ApplicationProperties[k] = v
				}
			}

			binding.URI = fmt.Sprintf("kamelet:%s/%s", kameletName, url.PathEscape(id))
		default:
			binding.URI = fmt.Sprintf("kamelet:%s/%s", kameletName, url.PathEscape(id))
		}

		return &binding, nil
	}
	return nil, nil
}

// DataTypeStep -- .
// Deprecated.
func (k V1alpha1BindingConverter) DataTypeStep(e v1alpha1.Endpoint, id string, typeSlot v1alpha1.TypeSlot) (map[string]interface{}, map[string]string) {
	if e.DataTypes == nil {
		return nil, nil
	}

	if inDataType, ok := e.DataTypes[typeSlot]; ok {
		scheme := "camel"
		if inDataType.Scheme != "" {
			scheme = inDataType.Scheme
		}

		props := make(map[string]string, 2)
		props[fmt.Sprintf("camel.kamelet.%s.%s-%s.scheme", datTypeActionKamelet, id, typeSlot)] = scheme
		props[fmt.Sprintf("camel.kamelet.%s.%s-%s.format", datTypeActionKamelet, id, typeSlot)] = inDataType.Format

		stepDsl := map[string]interface{}{
			"kamelet": map[string]interface{}{
				"name": fmt.Sprintf("%s/%s-%s", datTypeActionKamelet, url.PathEscape(id), typeSlot),
			},
		}

		return stepDsl, props
	}

	return nil, nil
}

// Order -- .
// Deprecated.
func (k V1alpha1BindingConverter) Order() int {
	return OrderStandard
}

func init() {
	RegisterBindingProvider(BindingConverter{})
	V1alpha1RegisterBindingProvider(V1alpha1BindingConverter{})
}

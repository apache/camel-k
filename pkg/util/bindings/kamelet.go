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
	"maps"
	"net/url"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/source"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	defaultDataTypeActionKamelet = "data-type-action"
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
	if e.Ref.Kind != v1.KameletKind || gv.Group != v1.SchemeGroupVersion.Group {
		return nil, nil
	}

	kameletName := url.PathEscape(e.Ref.Name)

	props, err := e.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}

	// Set id, if specified
	id, idPresent := props[v1.KameletIDProperty]
	if idPresent {
		delete(props, v1.KameletIDProperty)
	} else {
		id = endpointCtx.GenerateID()
	}
	// Set version, if specified
	version, versionPresent := props[v1.KameletVersionProperty]
	if versionPresent {
		delete(props, v1.KameletVersionProperty)
	}

	// Set namespace, if specified and different from the actual context
	namespace := ""
	if e.Ref.Namespace != "" && e.Ref.Namespace != ctx.Namespace {
		namespace = e.Ref.Namespace
	}

	kameletTranslated := getKameletName(kameletName, id, version, namespace)

	binding := Binding{}
	binding.ApplicationProperties = make(map[string]string)
	for k, v := range props {
		propKey := fmt.Sprintf("camel.kamelet.%s.%s.%s", kameletName, id, k)
		binding.ApplicationProperties[propKey] = v
	}

	dataTypeActionKamelet := ctx.Metadata[v1.KameletDataTypeLabel]
	if dataTypeActionKamelet == "" {
		dataTypeActionKamelet = defaultDataTypeActionKamelet
	}

	switch endpointCtx.Type {
	case v1.EndpointTypeAction:
		steps := make([]map[string]any, 0)

		if in, applicationProperties := k.DataTypeStep(e, id, v1.TypeSlotIn, dataTypeActionKamelet); in != nil {
			steps = append(steps, in)
			maps.Copy(binding.ApplicationProperties, applicationProperties)
		}

		steps = append(steps, map[string]any{
			"kamelet": map[string]any{
				"name": kameletTranslated,
			},
		})

		if out, applicationProperties := k.DataTypeStep(e, id, v1.TypeSlotOut, dataTypeActionKamelet); out != nil {
			steps = append(steps, out)
			maps.Copy(binding.ApplicationProperties, applicationProperties)
		}

		if len(steps) > 1 {
			binding.Step = map[string]any{
				"pipeline": map[string]any{
					"id":    id + "-pipeline",
					"steps": steps,
				},
			}
		} else if len(steps) == 1 {
			binding.Step = steps[0]
		}
	case v1.EndpointTypeSource:
		if out, applicationProperties := k.DataTypeStep(e, id, v1.TypeSlotOut, dataTypeActionKamelet); out != nil {
			binding.Step = out
			maps.Copy(binding.ApplicationProperties, applicationProperties)
		}

		binding.URI = "kamelet:" + kameletTranslated
	case v1.EndpointTypeSink:
		if in, applicationProperties := k.DataTypeStep(e, id, v1.TypeSlotIn, dataTypeActionKamelet); in != nil {
			binding.Step = in
			maps.Copy(binding.ApplicationProperties, applicationProperties)
		}

		binding.URI = "kamelet:" + kameletTranslated
	default:
		binding.URI = "kamelet:" + kameletTranslated
	}

	return &binding, nil
}

// getKameletName returns the kamelet with it's name and querystring attached.
func getKameletName(name, id, version, namespace string) string {
	kamelet := fmt.Sprintf("%s/%s", name, url.PathEscape(id))

	return source.GetKameletQuerystring(kamelet, version, namespace)
}

// DataTypeStep --.
func (k BindingConverter) DataTypeStep(e v1.Endpoint, id string, typeSlot v1.TypeSlot, dataTypeActionKamelet string) (map[string]any, map[string]string) {
	if e.DataTypes == nil {
		return nil, nil
	}

	if dataType, ok := e.DataTypes[typeSlot]; ok {
		scheme := "camel"
		format := dataType.Format
		if dataType.Scheme != "" {
			scheme = dataType.Scheme
		} else if strings.Contains(format, ":") {
			tuple := strings.SplitN(format, ":", 2)
			scheme = tuple[0]
			format = tuple[1]
		}

		props := make(map[string]string, 2)
		props[fmt.Sprintf("camel.kamelet.%s.%s-%s.scheme", dataTypeActionKamelet, id, typeSlot)] = scheme
		props[fmt.Sprintf("camel.kamelet.%s.%s-%s.format", dataTypeActionKamelet, id, typeSlot)] = format

		stepDsl := map[string]any{
			"kamelet": map[string]any{
				"name": fmt.Sprintf("%s/%s-%s", dataTypeActionKamelet, url.PathEscape(id), typeSlot),
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

func init() {
	RegisterBindingProvider(BindingConverter{})
}

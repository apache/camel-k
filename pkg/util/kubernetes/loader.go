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

package kubernetes

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

// LoadResourceFromYaml returns a Kubernetes resource from its serialized YAML definition.
func LoadResourceFromYaml(scheme *runtime.Scheme, data string) (ctrl.Object, error) {
	source := []byte(data)
	jsonSource, err := yaml.ToJSON(source)
	if err != nil {
		return nil, err
	}
	u := unstructured.Unstructured{}
	err = u.UnmarshalJSON(jsonSource)
	if err != nil {
		return nil, err
	}
	ro, err := runtimeObjectFromUnstructured(scheme, &u)
	if err != nil {
		return nil, err
	}
	o, ok := ro.(ctrl.Object)
	if !ok {
		return nil, fmt.Errorf("type assertion failed: %v", ro)
	}

	return o, nil
}

// LoadUnstructuredFromYaml returns an unstructured resource from its serialized YAML definition.
func LoadUnstructuredFromYaml(data string) (ctrl.Object, error) {
	source, err := yaml.ToJSON([]byte(data))
	if err != nil {
		return nil, err
	}
	var obj map[string]interface{}
	if err = json.Unmarshal(source, &obj); err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{
		Object: obj,
	}, nil
}

func runtimeObjectFromUnstructured(scheme *runtime.Scheme, u *unstructured.Unstructured) (runtime.Object, error) {
	gvk := u.GroupVersionKind()
	codecs := serializer.NewCodecFactory(scheme)
	decoder := codecs.UniversalDecoder(gvk.GroupVersion())

	b, err := u.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("error running MarshalJSON on unstructured object: %w", err)
	}
	ro, _, err := decoder.Decode(b, &gvk, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode json data with gvk(%v): %w", gvk.String(), err)
	}
	return ro, nil
}

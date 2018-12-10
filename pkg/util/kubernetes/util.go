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

	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// ToJSON --
func ToJSON(value runtime.Object) ([]byte, error) {
	u, err := k8sutil.UnstructuredFromRuntimeObject(value)
	if err != nil {
		return nil, fmt.Errorf("error creating unstructured data: %v", err)
	}
	data, err := runtime.Encode(unstructured.UnstructuredJSONScheme, u)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to json: %v", err)
	}
	return data, nil
}

// ToYAML --
func ToYAML(value runtime.Object) ([]byte, error) {
	data, err := ToJSON(value)
	if err != nil {
		return nil, err
	}

	return JSONToYAML(data)
}

// JSONToYAML --
func JSONToYAML(src []byte) ([]byte, error) {
	jsondata := map[string]interface{}{}
	err := json.Unmarshal(src, &jsondata)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json: %v", err)
	}
	yamldata, err := yaml.Marshal(&jsondata)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to yaml: %v", err)
	}

	return yamldata, nil
}

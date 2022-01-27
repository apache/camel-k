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

package dsl

import (
	"encoding/json"
	"fmt"

	yaml2 "gopkg.in/yaml.v2"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

// TemplateToYamlDSL converts a kamelet template into its Camel YAML DSL equivalent.
func TemplateToYamlDSL(template v1.Template, id string) ([]byte, error) {
	data, err := json.Marshal(&template)
	if err != nil {
		return nil, err
	}
	jsondata := make(map[string]interface{})
	err = json.Unmarshal(data, &jsondata)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json: %w", err)
	}
	if _, present := jsondata["id"]; !present {
		jsondata["id"] = id
	}
	templateWrapper := make(map[string]interface{}, 2)
	templateWrapper["template"] = jsondata
	listWrapper := make([]interface{}, 0, 1)
	listWrapper = append(listWrapper, templateWrapper)
	yamldata, err := yaml2.Marshal(listWrapper)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to yaml: %w", err)
	}

	return yamldata, nil
}

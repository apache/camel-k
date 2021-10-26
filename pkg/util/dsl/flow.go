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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	yaml2 "gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/util/yaml"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// FromYamlDSLString creates a slice of flows from a Camel YAML DSL string
func FromYamlDSLString(flowsString string) ([]v1.Flow, error) {
	return FromYamlDSL(bytes.NewReader([]byte(flowsString)))
}

// FromYamlDSL creates a slice of flows from a Camel YAML DSL stream
func FromYamlDSL(reader io.Reader) ([]v1.Flow, error) {
	buffered, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var flows []v1.Flow
	// Using the Kubernetes decoder to turn them into JSON before unmarshal.
	// This avoids having map[interface{}]interface{} objects which are not JSON compatible.
	jsonData, err := yaml.ToJSON(buffered)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(jsonData, &flows); err != nil {
		return nil, err
	}
	return flows, err
}

// ToYamlDSL converts a flow into its Camel YAML DSL equivalent
func ToYamlDSL(flows []v1.Flow) ([]byte, error) {
	data, err := json.Marshal(&flows)
	if err != nil {
		return nil, err
	}
	jsondata := make([]map[string]interface{}, 0)
	err = json.Unmarshal(data, &jsondata)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json: %w", err)
	}
	yamldata, err := yaml2.Marshal(&jsondata)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to yaml: %w", err)
	}

	return yamldata, nil
}

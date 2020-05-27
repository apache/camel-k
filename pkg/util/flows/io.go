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

package flows

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	yaml2 "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// UnmarshalString reads flows contained in a string
func UnmarshalString(flowsString string) ([]v1.Flow, error) {
	return Unmarshal(bytes.NewReader([]byte(flowsString)))
}

// Unmarshal flows from a stream
func Unmarshal(reader io.Reader) ([]v1.Flow, error) {
	buffered, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var flows []v1.Flow
	// Using the Kubernetes decoder to turn them into JSON before unmarshal.
	// This avoids having map[interface{}]interface{} objects which are not JSON compatible.
	jsonData, err := yaml.ToJSON(buffered)
	if err = json.Unmarshal(jsonData, &flows); err != nil {
		return nil, err
	}
	return flows, err
}

// Marshal flows as byte array
func Marshal(flows []v1.Flow) ([]byte, error) {
	return yaml2.Marshal(flows)
}

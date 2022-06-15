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

package cmd

import (
	"encoding/json"
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/stretchr/testify/assert"
)

func TestEditContainerTrait(t *testing.T) {
	var containerTrait v1.TraitSpec
	m := make(map[string]interface{})
	m["configuration"] = map[string]interface{}{
		"name":  "myName",
		"image": "myImage",
	}
	data, _ := json.Marshal(m)
	_ = json.Unmarshal(data, &containerTrait)

	editedContainerTrait, err := editContainerImage(containerTrait, "editedImage")
	assert.Nil(t, err)

	mappedTrait := make(map[string]map[string]interface{})
	newData, _ := json.Marshal(editedContainerTrait)
	_ = json.Unmarshal(newData, &mappedTrait)

	assert.Equal(t, "myName", mappedTrait["configuration"]["name"])
	assert.Equal(t, "editedImage", mappedTrait["configuration"]["image"])
}

func TestEditMissingContainerTrait(t *testing.T) {
	var containerTrait v1.TraitSpec

	editedContainerTrait, err := editContainerImage(containerTrait, "editedImage")
	assert.Nil(t, err)

	mappedTrait := make(map[string]map[string]interface{})
	newData, _ := json.Marshal(editedContainerTrait)
	_ = json.Unmarshal(newData, &mappedTrait)

	assert.Equal(t, "editedImage", mappedTrait["configuration"]["image"])
}

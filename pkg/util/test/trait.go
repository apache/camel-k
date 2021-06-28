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

package test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TraitSpecFromMap(t *testing.T, spec map[string]interface{}) v1.TraitSpec {
	var trait v1.TraitSpec

	data, err := json.Marshal(spec)
	assert.Nil(t, err)

	err = json.Unmarshal(data, &trait.Configuration)
	assert.Nil(t, err)

	return trait
}

func TraitSpecToMap(t *testing.T, spec v1.TraitSpec) map[string]string {
	trait := make(map[string]string)

	data, err := json.Marshal(spec.Configuration)
	assert.Nil(t, err)

	err = json.Unmarshal(data, &trait)
	assert.Nil(t, err)

	return trait
}

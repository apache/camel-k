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

package deploy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetResource(t *testing.T) {
	assert.NotEmpty(t, Resource("operator-service-account.yaml"))
	assert.NotEmpty(t, Resource("/operator-service-account.yaml"))
	assert.NotEmpty(t, ResourceAsString("operator-service-account.yaml"))
	assert.NotEmpty(t, ResourceAsString("/operator-service-account.yaml"))
	assert.Contains(t, Resources("/"), "operator-service-account.yaml")
}

func TestGetNoResource(t *testing.T) {
	assert.Empty(t, Resource("operator-service-account.json"))
	assert.Empty(t, Resource("/operator-service-account.json"))
	assert.Empty(t, ResourceAsString("operator-service-account.json"))
	assert.Empty(t, ResourceAsString("/operator-service-account.json"))
	assert.NotContains(t, Resources("/"), "operator-service-account.json")
}

func TestResources(t *testing.T) {
	assert.Contains(t, Resources("/"), "operator-service-account.yaml")
	assert.NotContains(t, Resources("/"), "resources.go")
	for _, res := range Resources("/") {
		if strings.Contains(res, "java.tmpl") {
			assert.Fail(t, "Resources should not return nested files")
		}
		if strings.Contains(res, "templates") {
			assert.Fail(t, "Resources should not return nested dirs")
		}
	}
	assert.Contains(t, Resources("/templates"), "java.tmpl")
	assert.Empty(t, Resources("/olm-catalog"))
}

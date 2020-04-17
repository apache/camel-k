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

package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
)

func TestPlatformHttpCapabilities(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name:    "Request.java",
			Content: `from("platform-http:/test").to("log:test");`,
		},
		Language: v1.LanguageJavaSource,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:platform-http",
			"camel:log",
		},
		meta.Dependencies.List())

	assert.ElementsMatch(
		t,
		[]string{
			v1.CapabilityPlatformHTTP,
		},
		meta.RequiredCapabilities.List())
}

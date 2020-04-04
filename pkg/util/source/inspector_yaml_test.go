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

package source

import (
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/stretchr/testify/assert"
)

func NewtestYAMLInspector(t *testing.T) YAMLInspector {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	return YAMLInspector{
		baseInspector: baseInspector{
			catalog: catalog,
		},
	}
}

const YAMLRestDSL = `
- rest:
    verb: "post"
    uri: "/api/flow"
    accepts: "text/plain"
    binding-mode: "off"
    steps:
      - convert-body:
          type: "java.lang.String"
      - to:
          uri: "log:in"
      - filter:
          simple: "${body.contains(\",\")}"
      - split:
          tokenize: ";"
      - set-body:
          simple: "${body.toLowerCase()}"
      - to:
          uri: "log:out"
`

const YAMLRestDSLWithRoute = `
- route:
    id: "flow"
    group: "routes"
    rest:
      verb: "post"
      uri: "/api/flow"
      accepts: "text/plain"
      binding-mode: "off"
      steps:
        - convert-body:
            type: "java.lang.String"
        - to:
            uri: "log:in"
        - filter:
            simple: "${body.contains(\",\")}"
        - split:
            tokenize: ";"
        - set-body:
            simple: "${body.toLowerCase()}"
        - to:
            uri: "log:out"
`

func TestYAMLRestDSL(t *testing.T) {
	for name, content := range map[string]string{"YAMLRestDSL": YAMLRestDSL, "YAMLRestDSLWithRoute": YAMLRestDSLWithRoute} {
		sourceContent := content
		t.Run(name, func(t *testing.T) {
			code := v1.SourceSpec{
				DataSpec: v1.DataSpec{
					Name:    "route.yaml",
					Content: sourceContent,
				},
				Language: v1.LanguageYaml,
			}

			meta := NewMetadata()
			inspector := NewtestYAMLInspector(t)

			err := inspector.Extract(code, &meta)
			assert.Nil(t, err)
			assert.True(t, meta.RequiredCapabilities.Has(v1.CapabilityRest))
			assert.True(t, meta.Dependencies.Has("camel:log"))
			assert.True(t, meta.ExposesHTTPServices)
		})
	}
}

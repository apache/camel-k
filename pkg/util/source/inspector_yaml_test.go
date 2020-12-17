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
	"fmt"
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

const YAMLRouteConsumer = `
- from:
    uri: knative:endpoint/default
    steps:
      - to:
          uri: "log:out"
`

const YAMLRouteProducer = `
- from:
    uri: timer:tick
    steps:
      - to:
          uri: knative:endpoint/service
`

const YAMLRouteTransformer = `
- from:
    uri: knative:channel/mychannel
    steps:
      - to:
          uri: knative:endpoint/service
`

func TestYAMLDependencies(t *testing.T) {
	tests := []struct {
		name                string
		source              string
		dependencies        []string
		missingDependencies []string
	}{
		{
			name:                "consumer",
			source:              YAMLRouteConsumer,
			dependencies:        []string{`mvn:org.apache.camel.k/camel-k-knative-consumer`},
			missingDependencies: []string{`mvn:org.apache.camel.k/camel-k-knative-producer`},
		},
		{
			name:                "producer",
			source:              YAMLRouteProducer,
			dependencies:        []string{`mvn:org.apache.camel.k/camel-k-knative-producer`},
			missingDependencies: []string{`mvn:org.apache.camel.k/camel-k-knative-consumer`},
		},
		{
			name:         "transformer",
			source:       YAMLRouteTransformer,
			dependencies: []string{`mvn:org.apache.camel.k/camel-k-knative-producer`, `mvn:org.apache.camel.k/camel-k-knative-consumer`},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code := v1.SourceSpec{
				DataSpec: v1.DataSpec{
					Name:    "route.yaml",
					Content: test.source,
				},
				Language: v1.LanguageYaml,
			}

			meta := NewMetadata()
			inspector := NewtestYAMLInspector(t)

			err := inspector.Extract(code, &meta)
			assert.Nil(t, err)
			for _, dependency := range test.dependencies {
				assert.Contains(t, meta.Dependencies.List(), dependency)
			}
			for _, missingDependency := range test.missingDependencies {
				assert.NotContains(t, meta.Dependencies.List(), missingDependency)
			}
		})
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
			assert.True(t, meta.Dependencies.Has("camel-quarkus:log"))
			assert.True(t, meta.ExposesHTTPServices)
		})
	}
}

const YAMLJSONMarshal = `
- from:
    uri: timer:tick
    steps:
    - marshal:
        json: {}
`

const YAMLJSONUnmarshal = `
- from:
    uri: timer:tick
    steps:
    - unmarshal:
        json: {}
`

const YAMLJSONGsonMarshal = `
- from:
    uri: timer:tick
    steps:
    - marshal:
        json:
          library: Gson
`

const YAMLJSONUnknownMarshal = `
- from:
    uri: timer:tick
    steps:
    - marshal:
        json:
          library: Unknown
`

func TestYAMLJson(t *testing.T) {
	tc := []struct {
		source     string
		dependency string
	}{
		{
			source:     YAMLJSONMarshal,
			dependency: "camel-quarkus:jackson",
		},
		{
			source:     YAMLJSONUnmarshal,
			dependency: "camel-quarkus:jackson",
		},
		{
			source:     YAMLJSONGsonMarshal,
			dependency: "camel-quarkus:gson",
		},
		{
			source:     YAMLJSONUnknownMarshal,
			dependency: "camel-quarkus:timer",
		},
	}

	for i, test := range tc {
		t.Run(fmt.Sprintf("%s-%d", test.dependency, i), func(t *testing.T) {
			code := v1.SourceSpec{
				DataSpec: v1.DataSpec{
					Name:    "route.yaml",
					Content: test.source,
				},
				Language: v1.LanguageYaml,
			}

			meta := NewMetadata()
			inspector := NewtestYAMLInspector(t)

			err := inspector.Extract(code, &meta)
			assert.Nil(t, err)
			assert.True(t, meta.RequiredCapabilities.IsEmpty())
			assert.Contains(t, meta.Dependencies.List(), test.dependency)
		})
	}
}

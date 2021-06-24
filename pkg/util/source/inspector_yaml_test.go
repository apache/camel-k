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

	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
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

const YAMLInvalid = `
- from:
    uri: knative:endpoint/default
    steps:
      - "log:out"
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
			dependencies:        []string{`mvn:org.apache.camel.k:camel-k-knative-consumer`},
			missingDependencies: []string{`mvn:org.apache.camel.k:camel-k-knative-producer`},
		},
		{
			name:                "producer",
			source:              YAMLRouteProducer,
			dependencies:        []string{`mvn:org.apache.camel.k:camel-k-knative-producer`},
			missingDependencies: []string{`mvn:org.apache.camel.k:camel-k-knative-consumer`},
		},
		{
			name:   "transformer",
			source: YAMLRouteTransformer,
			dependencies: []string{
				`mvn:org.apache.camel.k:camel-k-knative-producer`,
				`mvn:org.apache.camel.k:camel-k-knative-consumer`,
			},
		},
		{
			name:   "invalid",
			source: YAMLInvalid,
			dependencies: []string{
				`mvn:org.apache.camel.k:camel-k-knative-consumer`,
			},
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
			assert.True(t, meta.Dependencies.Has("camel:log"))
			assert.True(t, meta.ExposesHTTPServices)
		})
	}
}

const YAMLFromDSL = `
- from:
    uri: "timer:tick"
    parameters:
      period: "5000"
    steps:
      - set-body:
          constant: "Hello Yaml !!!"
      - transform:
          simple: "${body.toUpperCase()}"
      - to: "log:info"
`

const YAMLFromDSLWithRoute = `
- route:
    id: route1
    from:
      uri: "timer:tick"
      parameters:
        period: "5000"
    steps:
      - set-body:
          constant: "Hello Yaml !!!"
      - transform:
          simple: "${body.toUpperCase()}"
      - to: "log:info"
`

func TestYAMLRouteAndFromEquivalence(t *testing.T) {
	for name, content := range map[string]string{"YAMLFromDSL": YAMLFromDSL, "YAMLFromDSLWithRoute": YAMLFromDSLWithRoute} {
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
			assert.Equal(t, meta.FromURIs, []string{"timer:tick?period=5000"})
			assert.Equal(t, meta.ToURIs, []string{"log:info"})
			assert.True(t, meta.Dependencies.Has("camel:log"))
			assert.True(t, meta.Dependencies.Has("camel:timer"))
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
			dependency: "camel:jackson",
		},
		{
			source:     YAMLJSONUnmarshal,
			dependency: "camel:jackson",
		},
		{
			source:     YAMLJSONGsonMarshal,
			dependency: "camel:gson",
		},
		{
			source:     YAMLJSONUnknownMarshal,
			dependency: "camel:timer",
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

const YAMLKameletEipNoId = `
- from:
    uri: timer:tick
    steps:
    - kamelet: "foo"
`

const YAMLKameletEipInline = `
- from:
    uri: timer:tick
    steps:
    - kamelet: "foo/bar?baz=test"
`
const YAMLKameletEipMap = `
- from:
    uri: timer:tick
    steps:
    - kamelet: 
        name: "foo/bar?baz=test"
`
const YAMLKameletEipMapWithParams = `
- from:
    uri: timer:tick
    steps:
    - kamelet: 
        name: "foo/bar"
        parameters:
          baz:test
`
const YAMLKameletEndpoint = `
- from:
    uri: timer:tick
    steps:
    - to: "kamelet:foo/bar?baz=test"
`

func TestYAMLKamelet(t *testing.T) {
	tc := []struct {
		source   string
		kamelets []string
	}{
		{
			source:   YAMLKameletEipNoId,
			kamelets: []string{"foo"},
		},
		{
			source:   YAMLKameletEipInline,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   YAMLKameletEipMap,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   YAMLKameletEipMapWithParams,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   YAMLKameletEndpoint,
			kamelets: []string{"foo/bar"},
		},
	}

	for i, test := range tc {
		t.Run(fmt.Sprintf("TestYAMLKamelet-%d", i), func(t *testing.T) {
			code := v1.SourceSpec{
				DataSpec: v1.DataSpec{
					Content: test.source,
				},
			}

			catalog, err := camel.DefaultCatalog()
			assert.Nil(t, err)

			meta := NewMetadata()
			inspector := YAMLInspector{
				baseInspector: baseInspector{
					catalog: catalog,
				},
			}

			err = inspector.Extract(code, &meta)
			assert.Nil(t, err)
			assert.True(t, meta.RequiredCapabilities.IsEmpty())

			for _, k := range test.kamelets {
				assert.Contains(t, meta.Kamelets, k)
			}
		})
	}
}

const YAMLKameletExplicitParams = `
- from:
    uri: cron:tab
    parameters:
      schedule: "* * * * ?"
    steps:
    - to:
        uri: knative:channel/a
        parameters:
          cloudEventsSpecVersion: "1.0"
`

const YAMLKameletExplicitNumericParams = `
- from:
    uri: timer:tick
    parameters:
      period: 1000
    steps:
    - log: "hello"
`

func TestYAMLExplicitParameters(t *testing.T) {
	tc := []struct {
		source   string
		fromURIs []string
		toURIs   []string
	}{
		{
			source:   YAMLKameletExplicitParams,
			fromURIs: []string{"cron:tab?schedule=%2A+%2A+%2A+%2A+%3F"},
			toURIs:   []string{"knative:channel/a?cloudEventsSpecVersion=1.0"},
		},
		{
			source:   YAMLKameletExplicitNumericParams,
			fromURIs: []string{"timer:tick?period=1000"},
		},
	}

	for i, test := range tc {
		t.Run(fmt.Sprintf("TestYAMLExplicitParameters-%d", i), func(t *testing.T) {
			code := v1.SourceSpec{
				DataSpec: v1.DataSpec{
					Content: test.source,
				},
			}

			catalog, err := camel.DefaultCatalog()
			assert.Nil(t, err)

			meta := NewMetadata()
			inspector := YAMLInspector{
				baseInspector: baseInspector{
					catalog: catalog,
				},
			}

			err = inspector.Extract(code, &meta)
			assert.Nil(t, err)
			assert.True(t, meta.RequiredCapabilities.IsEmpty())

			assert.Len(t, meta.FromURIs, len(test.fromURIs))
			for _, k := range test.fromURIs {
				assert.Contains(t, meta.FromURIs, k)
			}
			assert.Len(t, meta.ToURIs, len(test.toURIs))
			for _, k := range test.toURIs {
				assert.Contains(t, meta.ToURIs, k)
			}
		})
	}
}

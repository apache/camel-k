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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
)

func newTestYAMLInspector(t *testing.T) YAMLInspector {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	return YAMLInspector{
		baseInspector: baseInspector{
			catalog: catalog,
		},
	}
}

const yamlRouteConsumer = `
- from:
    uri: knative:endpoint/default
    steps:
      - to:
          uri: "log:out"
`

const yamlRouteProducer = `
- from:
    uri: timer:tick
    steps:
      - to:
          uri: knative:endpoint/service
`

const yamlRouteTransformer = `
- from:
    uri: knative:channel/mychannel
    steps:
      - to:
          uri: knative:endpoint/service
`

const yamlInvalid = `
- from:
    uri: knative:endpoint/default
    steps:
      - "log:out"
`

const yamlInDepthChannel = `
- from:
    uri: knative:channel/mychannel
    steps:
      - choice:
          when:
          - simple: "${body}"
            steps:
            - to:
                uri: knative:endpoint/service
`

const yamlWireTapKnativeEIP = `
- from:
    uri: knative:channel/mychannel
    parameters:
      period: "1000"
    steps:
      - wireTap:
          uri: knative:channel/mychannel
`

const yamlWireTapJmsEIP = `
- from:
    uri: knative:channel/mychannel
    parameters:
      period: "1000"
    steps:
      - wireTap:
          uri: jms:queue:foo
`

func TestYAMLDependencies(t *testing.T) {
	tests := []struct {
		name                string
		source              string
		dependencies        []string
		missingDependencies []string
	}{
		{
			name:         "consumer",
			source:       yamlRouteConsumer,
			dependencies: []string{`camel:knative`},
		},
		{
			name:         "producer",
			source:       yamlRouteProducer,
			dependencies: []string{`camel:knative`},
		},
		{
			name:   "transformer",
			source: yamlRouteTransformer,
			dependencies: []string{
				`camel:knative`,
			},
		},
		{
			name:   "invalid",
			source: yamlInvalid,
			dependencies: []string{
				`camel:knative`,
			},
		},
		{
			name:   "in-depth",
			source: yamlInDepthChannel,
			dependencies: []string{
				`camel:knative`,
			},
		},
		{
			name:   "wire-tap-knative",
			source: yamlWireTapKnativeEIP,
			dependencies: []string{
				`camel:knative`,
			},
		},
		{
			name:   "wire-tap-jms",
			source: yamlWireTapJmsEIP,
			dependencies: []string{
				`camel:knative`,
				`camel:jms`,
			},
		},
	}

	inspector := newTestYAMLInspector(t)
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			assertExtractYAML(t, inspector, test.source, func(meta *Metadata) {
				for _, dependency := range test.dependencies {
					assert.Contains(t, meta.Dependencies.List(), dependency)
				}
				for _, missingDependency := range test.missingDependencies {
					assert.NotContains(t, meta.Dependencies.List(), missingDependency)
				}
			})
		})
	}
}

const yamlRestDSL = `
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

const yamlRestDSLWithRoute = `
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
	inspector := newTestYAMLInspector(t)
	for name, content := range map[string]string{
		"yamlRestDSL":          yamlRestDSL,
		"yamlRestDSLWithRoute": yamlRestDSLWithRoute,
	} {
		sourceContent := content
		t.Run(name, func(t *testing.T) {
			assertExtractYAML(t, inspector, sourceContent, func(meta *Metadata) {
				assert.True(t, meta.RequiredCapabilities.Has(v1.CapabilityRest))
				assert.True(t, meta.Dependencies.Has("camel:log"))
				assert.True(t, meta.ExposesHTTPServices)
			})
		})
	}
}

const yamlFromDSL = `
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

const yamlFromDSLWithRoute = `
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
	inspector := newTestYAMLInspector(t)
	for name, content := range map[string]string{
		"yamlFromDSL":          yamlFromDSL,
		"yamlFromDSLWithRoute": yamlFromDSLWithRoute,
	} {
		sourceContent := content
		t.Run(name, func(t *testing.T) {
			assertExtractYAML(t, inspector, sourceContent, func(meta *Metadata) {
				assert.Equal(t, meta.FromURIs, []string{"timer:tick?period=5000"})
				assert.Equal(t, meta.ToURIs, []string{"log:info"})
				assert.True(t, meta.Dependencies.Has("camel:log"))
				assert.True(t, meta.Dependencies.Has("camel:timer"))
			})
		})
	}
}

const yamlJSONMarshal = `
- from:
    uri: timer:tick
    steps:
    - marshal:
        json: {}
`

const yamlJSONUnmarshal = `
- from:
    uri: timer:tick
    steps:
    - unmarshal:
        json: {}
`

const yamlJSONGsonMarshal = `
- from:
    uri: timer:tick
    steps:
    - marshal:
        json:
          library: Gson
`

const yamlJSONUnknownMarshal = `
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
			source:     yamlJSONMarshal,
			dependency: "camel:jackson",
		},
		{
			source:     yamlJSONUnmarshal,
			dependency: "camel:jackson",
		},
		{
			source:     yamlJSONGsonMarshal,
			dependency: "camel:gson",
		},
		{
			source:     yamlJSONUnknownMarshal,
			dependency: "camel:timer",
		},
	}

	inspector := newTestYAMLInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("%s-%d", test.dependency, i), func(t *testing.T) {
			assertExtractYAML(t, inspector, test.source, func(meta *Metadata) {
				assert.True(t, meta.RequiredCapabilities.IsEmpty())
				assert.Contains(t, meta.Dependencies.List(), test.dependency)
			})
		})
	}
}

const yamlAvroEndpoint = `
- from:
    uri: direct:start
    steps:
    - to:
        uri: dataformat:avro:marshal
`

const yamlJacksonEndpoint = `
- from:
    uri: direct:start
    steps:
    - to:
        uri: dataformat:jackson:marshal
`

const yamlProtobufEndpoint = `
- from:
    uri: direct:start
    steps:
    - to:
        uri: dataformat:protobuf:marshal
`

func TestYAMLDataFormat(t *testing.T) {
	tc := []struct {
		source string
		deps   []string
	}{
		{
			source: yamlAvroEndpoint,
			deps:   []string{"camel:dataformat", "camel:avro"},
		},
		{
			source: yamlJacksonEndpoint,
			deps:   []string{"camel:dataformat", "camel:jackson"},
		},
		{
			source: yamlProtobufEndpoint,
			deps:   []string{"camel:dataformat", "camel:protobuf"},
		},
	}

	inspector := newTestYAMLInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestYAMLDataFormat-%d", i), func(t *testing.T) {
			assertExtract(t, inspector, test.source, func(meta *Metadata) {
				for _, d := range test.deps {
					assert.Contains(t, meta.Dependencies.List(), d)
				}
			})
		})
	}
}

const yamlKameletEipNoID = `
- from:
    uri: timer:tick
    steps:
    - kamelet: "foo"
`

const yamlKameletEipInline = `
- from:
    uri: timer:tick
    steps:
    - kamelet: "foo/bar?baz=test"
`

const yamlKameletEipMap = `
- from:
    uri: timer:tick
    steps:
    - kamelet: 
        name: "foo/bar?baz=test"
`

// #nosec G101
const yamlKameletEipMapWithParams = `
- from:
    uri: timer:tick
    steps:
    - kamelet: 
        name: "foo/bar"
        parameters:
          baz:test
`

const yamlKameletEndpoint = `
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
			source:   yamlKameletEipNoID,
			kamelets: []string{"foo"},
		},
		{
			source:   yamlKameletEipInline,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   yamlKameletEipMap,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   yamlKameletEipMapWithParams,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   yamlKameletEndpoint,
			kamelets: []string{"foo/bar"},
		},
	}

	inspector := newTestYAMLInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestYAMLKamelet-%d", i), func(t *testing.T) {
			assertExtract(t, inspector, test.source, func(meta *Metadata) {
				assert.True(t, meta.RequiredCapabilities.IsEmpty())
				for _, k := range test.kamelets {
					assert.Contains(t, meta.Kamelets, k)
				}
			})
		})
	}
}

const yamlKameletExplicitParams = `
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

const yamlKameletExplicitNumericParams = `
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
			source:   yamlKameletExplicitParams,
			fromURIs: []string{"cron:tab?schedule=%2A+%2A+%2A+%2A+%3F"},
			toURIs:   []string{"knative:channel/a?cloudEventsSpecVersion=1.0"},
		},
		{
			source:   yamlKameletExplicitNumericParams,
			fromURIs: []string{"timer:tick?period=1000"},
		},
	}

	inspector := newTestYAMLInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestYAMLExplicitParameters-%d", i), func(t *testing.T) {
			assertExtract(t, inspector, test.source, func(meta *Metadata) {
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
		})
	}
}

const yamlFromDSLWithPropertyPlaceholder = `
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
      - to: "{{url}}"
`

const yamlFromDSLWithPropertyPlaceholderScheme = `
- route:
    id: route2
    from:
      uri: "timer:tick"
      parameters:
        period: "5000"
    steps:
      - set-body:
          constant: "Hello Yaml !!!"
      - transform:
          simple: "${body.toUpperCase()}"
      - to: "{{scheme}}:{{resource}}"
`

func TestYAMLRouteWithPropertyPlaceholder(t *testing.T) {
	tc := []struct {
		source   string
		fromURIs []string
		toURIs   []string
	}{
		{
			source:   yamlFromDSLWithPropertyPlaceholder,
			fromURIs: []string{"timer:tick?period=5000"},
			toURIs:   []string{"{{url}}"},
		},
		{
			source:   yamlFromDSLWithPropertyPlaceholderScheme,
			fromURIs: []string{"timer:tick?period=5000"},
			toURIs:   []string{"{{scheme}}:{{resource}}"},
		},
	}

	inspector := newTestYAMLInspector(t)
	for i, test := range tc {
		t.Run(fmt.Sprintf("TestYAMLRouteWithPropertyPlaceholder-%d", i), func(t *testing.T) {
			assertExtractYAML(t, inspector, test.source, func(meta *Metadata) {
				assert.Len(t, meta.FromURIs, len(test.fromURIs))
				for _, k := range test.fromURIs {
					assert.Contains(t, meta.FromURIs, k)
				}
				assert.Len(t, meta.ToURIs, len(test.toURIs))
				for _, k := range test.toURIs {
					assert.Contains(t, meta.ToURIs, k)
				}
				assert.Equal(t, meta.Dependencies.Size(), 2)
				assert.True(t, meta.Dependencies.Has("camel:core"))
				assert.True(t, meta.Dependencies.Has("camel:timer"))
			})
		})
	}
}

const yamlFromDSLWithUnknownFromScheme = `
- route:
    id: route1
    from:
      uri: "unknown:foo"
    steps:
      - to: "log:info"
`

const yamlFromDSLWithUnknownToScheme = `
- route:
    id: route2
    from:
      uri: "timer:tick"
    steps:
      - to: "unknown:foo"
`

func TestYAMLRouteWithUnknownScheme(t *testing.T) {
	inspector := newTestYAMLInspector(t)
	for i, source := range []string{yamlFromDSLWithUnknownFromScheme, yamlFromDSLWithUnknownToScheme} {
		t.Run(fmt.Sprintf("TestYAMLRouteWithUnknownScheme-%d", i), func(t *testing.T) {
			assertExtractYAMLError(t, inspector, source, func(err error) {
				assert.True(t, strings.HasPrefix(err.Error(), fmt.Sprintf("component not found for uri %q", "unknown:foo")))
			})
		})
	}
}

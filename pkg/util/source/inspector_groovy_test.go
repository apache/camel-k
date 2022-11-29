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
	"github.com/stretchr/testify/require"

	"github.com/apache/camel-k/pkg/util/camel"
)

func newTestGroovyInspector(t *testing.T) GroovyInspector {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	return GroovyInspector{
		baseInspector: baseInspector{
			catalog: catalog,
		},
	}
}

const groovyKameletEip = `
from("direct:start")
    .kamelet("foo/bar?baz=test")
`

const groovyKameletEipSingleQuote = `
from("direct:start")
    .kamelet('foo/bar?baz=test')
`

const groovyKameletEndpoint = `
from("direct:start")
    .to("kamelet:foo/bar?baz=test")
`

const groovyKameletEndpointSingleQuote = `
from("direct:start")
    .to('kamelet:foo/bar?baz=test')
`

const groovyWireTapEipSingleQuote = `
from("direct:start")
    .wireTap('kamelet:foo/bar?baz=test')
`

func TestGroovyKamelet(t *testing.T) {
	tc := []struct {
		source   string
		kamelets []string
	}{
		{
			source:   groovyKameletEip,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   groovyKameletEndpoint,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   groovyKameletEipSingleQuote,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   groovyKameletEndpointSingleQuote,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   groovyWireTapEipSingleQuote,
			kamelets: []string{"foo/bar"},
		},
	}

	inspector := newTestGroovyInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestGroovyKamelet-%d", i), func(t *testing.T) {
			assertExtract(t, inspector, test.source, func(meta *Metadata) {
				assert.True(t, meta.RequiredCapabilities.IsEmpty())
				for _, k := range test.kamelets {
					assert.Contains(t, meta.Kamelets, k)
				}
			})
		})
	}
}

const groovyJSONEip = `
from("direct:start")
    .unmarshal().json()
`

const groovyJSONJacksonEip = `
from("direct:start")
    .unmarshal().json(JsonLibrary.Jackson)
`

const groovyAvroEndpoint = `
from("direct:start")
    .to("dataformat:avro:marshal")
`

const groovyJacksonEndpoint = `
from("direct:start")
    .to("dataformat:jackson:marshal")
`

const groovyProtobufEndpoint = `
from("direct:start")
    .to("dataformat:protobuf:marshal")
`

func TestGroovyDataFormat(t *testing.T) {
	tc := []struct {
		source string
		deps   []string
	}{
		{
			source: groovyJSONEip,
			deps:   []string{"camel:jackson"},
		},
		{
			source: groovyJSONJacksonEip,
			deps:   []string{"camel:jackson"},
		},
		{
			source: groovyAvroEndpoint,
			deps:   []string{"camel:dataformat", "camel:avro"},
		},
		{
			source: groovyJacksonEndpoint,
			deps:   []string{"camel:dataformat", "camel:jackson"},
		},
		{
			source: groovyProtobufEndpoint,
			deps:   []string{"camel:dataformat", "camel:protobuf"},
		},
	}

	inspector := newTestGroovyInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestGroovyDataFormat-%d", i), func(t *testing.T) {
			assertExtract(t, inspector, test.source, func(meta *Metadata) {
				for _, d := range test.deps {
					assert.Contains(t, meta.Dependencies.List(), d)
				}
			})
		})
	}
}

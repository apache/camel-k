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

	"github.com/apache/camel-k/pkg/util/camel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestJavaScriptInspector(t *testing.T) JavaScriptInspector {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	return JavaScriptInspector{
		baseInspector: baseInspector{
			catalog: catalog,
		},
	}
}

const javaScriptKameletEip = `
from("direct:start")
    .kamelet("foo/bar?baz=test")
`

const javaScriptKameletEndpoint = `
from("direct:start")
    .to("kamelet:foo/bar?baz=test")
`

const javaScriptWireTapEipSingleQuote = `
from("direct:start")
    .wireTap('kamelet:foo/bar?baz=test')
`

func TestJavaScriptKamelet(t *testing.T) {
	tc := []struct {
		source   string
		kamelets []string
	}{
		{
			source:   javaScriptKameletEip,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   javaScriptKameletEndpoint,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   javaScriptWireTapEipSingleQuote,
			kamelets: []string{"foo/bar"},
		},
	}

	inspector := newTestJavaScriptInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestJavaScriptKamelet-%d", i), func(t *testing.T) {
			assertExtract(t, inspector, test.source, func(meta *Metadata) {
				assert.True(t, meta.RequiredCapabilities.IsEmpty())
				for _, k := range test.kamelets {
					assert.Contains(t, meta.Kamelets, k)
				}
			})
		})
	}
}

const javaScriptJSONEip = `
from('direct:start')
    .unmarshal().json()
`

const javaScriptJSONJacksonEip = `
from('direct:start')
    .unmarshal().json(JsonLibrary.Jackson)
`

const javaScriptAvroEndpoint = `
from('direct:start')
    .to('dataformat:avro:marshal');
`

const javaScriptJacksonEndpoint = `
from('direct:start')
    .to('dataformat:jackson:marshal')
`

const javaScriptProtobufEndpoint = `
from('direct:start')
    .to('dataformat:protobuf:marshal')
`

func TestJavaScriptDataFormat(t *testing.T) {
	tc := []struct {
		source string
		deps   []string
	}{
		{
			source: javaScriptJSONEip,
			deps:   []string{"camel:jackson"},
		},
		{
			source: javaScriptJSONJacksonEip,
			deps:   []string{"camel:jackson"},
		},
		{
			source: javaScriptAvroEndpoint,
			deps:   []string{"camel:dataformat", "camel:avro"},
		},
		{
			source: javaScriptJacksonEndpoint,
			deps:   []string{"camel:dataformat", "camel:jackson"},
		},
		{
			source: javaScriptProtobufEndpoint,
			deps:   []string{"camel:dataformat", "camel:protobuf"},
		},
	}

	inspector := newTestJavaScriptInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestJavaScriptDataFormat-%d", i), func(t *testing.T) {
			assertExtract(t, inspector, test.source, func(meta *Metadata) {
				for _, d := range test.deps {
					assert.Contains(t, meta.Dependencies.List(), d)
				}
			})
		})
	}
}

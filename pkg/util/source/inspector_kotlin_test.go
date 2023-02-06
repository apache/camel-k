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

func newTestKotlinInspector(t *testing.T) KotlinInspector {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	return KotlinInspector{
		baseInspector: baseInspector{
			catalog: catalog,
		},
	}
}

const kotlinKameletEip = `
from("direct:start")
    .kamelet("foo/bar?baz=test")
`

const kotlinKameletEndpoint = `
from("direct:start")
    .to("kamelet:foo/bar?baz=test")
`

const kotlinWireTapEip = `
from("direct:start")
    .wireTap("kamelet:foo/bar?baz=test")
`

func TestKotlinKamelet(t *testing.T) {
	tc := []struct {
		source   string
		kamelets []string
	}{
		{
			source:   kotlinKameletEip,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   kotlinKameletEndpoint,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   kotlinWireTapEip,
			kamelets: []string{"foo/bar"},
		},
	}

	inspector := newTestKotlinInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestKotlinKamelet-%d", i), func(t *testing.T) {
			assertExtract(t, inspector, test.source, func(meta *Metadata) {
				assert.True(t, meta.RequiredCapabilities.IsEmpty())
				for _, k := range test.kamelets {
					assert.Contains(t, meta.Kamelets, k)
				}
			})
		})
	}
}

const kotlinJSONEip = `
from("direct:start")
    .unmarshal().json()
`

const kotlinJSONJacksonEip = `
from("direct:start")
    .unmarshal().json(JsonLibrary.Jackson)
`

const kotlinAvroEndpoint = `
from("direct:start")
    .to("dataformat:avro:marshal")
`

const kotlinJacksonEndpoint = `
from("direct:start")
    .to("dataformat:jackson:marshal")
`

const kotlinProtobufEndpoint = `
from("direct:start")
    .to("dataformat:protobuf:marshal")
`

func TestKotlinDataFormat(t *testing.T) {
	tc := []struct {
		source string
		deps   []string
	}{
		{
			source: kotlinJSONEip,
			deps:   []string{"camel:jackson"},
		},
		{
			source: kotlinJSONJacksonEip,
			deps:   []string{"camel:jackson"},
		},
		{
			source: kotlinAvroEndpoint,
			deps:   []string{"camel:dataformat", "camel:avro"},
		},
		{
			source: kotlinJacksonEndpoint,
			deps:   []string{"camel:dataformat", "camel:jackson"},
		},
		{
			source: kotlinProtobufEndpoint,
			deps:   []string{"camel:dataformat", "camel:protobuf"},
		},
	}

	inspector := newTestKotlinInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestKotlinDataFormat-%d", i), func(t *testing.T) {
			assertExtract(t, inspector, test.source, func(meta *Metadata) {
				for _, d := range test.deps {
					assert.Contains(t, meta.Dependencies.List(), d)
				}
			})
		})
	}
}

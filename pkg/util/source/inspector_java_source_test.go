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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/camel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestJavaSourceInspector(t *testing.T) JavaSourceInspector {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	return JavaSourceInspector{
		baseInspector: baseInspector{
			catalog: catalog,
		},
	}
}

const javaSourceKameletEip = `
from("direct:start")
    .kamelet("foo/bar?baz=test");
`

const javaSourceKameletEndpoint = `
from("direct:start")
    .to("kamelet:foo/bar?baz=test");
`

const javaSourceWireTapEip = `
from("direct:start")
    .wireTap("kamelet:foo/bar?baz=test");
`

func TestJavaSourceKamelet(t *testing.T) {
	tc := []struct {
		source   string
		kamelets []string
	}{
		{
			source:   javaSourceKameletEip,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   javaSourceKameletEndpoint,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   javaSourceWireTapEip,
			kamelets: []string{"foo/bar"},
		},
	}

	inspector := newTestJavaSourceInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestJavaSourceKamelet-%d", i), func(t *testing.T) {
			assertExtract(t, inspector, test.source, func(meta *Metadata) {
				assert.True(t, meta.RequiredCapabilities.IsEmpty())
				for _, k := range test.kamelets {
					assert.Contains(t, meta.Kamelets, k)
				}
			})
		})
	}
}

const javaSourceJSONEip = `
from("direct:start")
    .unmarshal().json();
`

const javaSourceJSONJacksonEip = `
from("direct:start")
    .unmarshal().json(JsonLibrary.Jackson);
`

const javaSourceAvroEndpoint = `
from("direct:start")
    .to("dataformat:avro:marshal");
`

const javaSourceJacksonEndpoint = `
from("direct:start")
    .to("dataformat:jackson:marshal");
`

const javaSourceProtobufEndpoint = `
from("direct:start")
    .to("dataformat:protobuf:marshal");
`

func TestJavaSourceDataFormat(t *testing.T) {
	tc := []struct {
		source string
		deps   []string
	}{
		{
			source: javaSourceJSONEip,
			deps:   []string{"camel:jackson"},
		},
		{
			source: javaSourceJSONJacksonEip,
			deps:   []string{"camel:jackson"},
		},
		{
			source: javaSourceAvroEndpoint,
			deps:   []string{"camel:dataformat", "camel:avro"},
		},
		{
			source: javaSourceJacksonEndpoint,
			deps:   []string{"camel:dataformat", "camel:jackson"},
		},
		{
			source: javaSourceProtobufEndpoint,
			deps:   []string{"camel:dataformat", "camel:protobuf"},
		},
	}

	inspector := newTestJavaSourceInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestJavaSourceDataFormat-%d", i), func(t *testing.T) {
			assertExtract(t, inspector, test.source, func(meta *Metadata) {
				for _, d := range test.deps {
					assert.Contains(t, meta.Dependencies.List(), d)
				}
			})
		})
	}
}

func TestJavaReplaceURI(t *testing.T) {
	inspector := newTestJavaSourceInspector(t)

	sourceSpec := &v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name:    "test.java",
			Content: "from(\"quartz:trigger?cron=0 0/1 * * * ?\").to(\"log:info\")",
		},
	}
	replaced, err := inspector.ReplaceFromURI(
		sourceSpec,
		"direct:newURI?hello=world",
	)
	assert.Nil(t, err)
	assert.True(t, replaced)
	assert.Equal(t, "from(\"direct:newURI?hello=world\").to(\"log:info\")", sourceSpec.Content)
}

func TestJavaRestOpenapiFirst(t *testing.T) {
	inspector := newTestJavaSourceInspector(t)

	sourceSpec := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "test.java",
			Content: `
public void configure() throws Exception {
    rest().openApi("petstore-v3.json");
}
			`,
		},
	}
	meta := NewMetadata()
	err := inspector.Extract(sourceSpec, &meta)
	require.NoError(t, err)
	assert.Contains(t, meta.Dependencies.List(), "camel:rest-openapi")
}

func TestJavaBeanDependencies(t *testing.T) {
	inspector := newTestJavaSourceInspector(t)

	sourceSpec := &v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name:    "test.java",
			Content: "from(\"timer:foo\").bean(\"myBean\").to(\"log:bar\")",
		},
	}
	assertExtract(t, inspector, sourceSpec.Content, func(meta *Metadata) {
		assert.Contains(t, meta.Dependencies.List(), "camel:timer")
		assert.Contains(t, meta.Dependencies.List(), "camel:bean")
		assert.Contains(t, meta.Dependencies.List(), "camel:log")
	})
}

func TestErrorHandlerDependencies(t *testing.T) {
	inspector := newTestJavaSourceInspector(t)

	sourceSpec := &v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "test.java",
			Content: `
			public void configure() throws Exception {
				errorHandler(deadLetterChannel("seda:error"));
				from("timer:foo").to("log:bar");
			}
			`,
		},
	}
	assertExtract(t, inspector, sourceSpec.Content, func(meta *Metadata) {
		assert.Contains(t, meta.Dependencies.List(), "camel:timer")
		assert.Contains(t, meta.Dependencies.List(), "camel:seda")
		assert.Contains(t, meta.Dependencies.List(), "camel:log")
	})
}

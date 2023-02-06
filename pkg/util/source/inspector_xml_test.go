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

func newTestXMLInspector(t *testing.T) XMLInspector {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	return XMLInspector{
		baseInspector: baseInspector{
			catalog: catalog,
		},
	}
}

const xmlKameletEip = `
<camelContext xmlns="http://camel.apache.org/schema/spring">
  <route>
    <from uri="direct:start"/>
    <kamelet name="foo/bar?baz=test"/>
    <to uri="mock:result"/>
  </route>
</camelContext>
`

const xmlKameletEndpoint = `
<camelContext xmlns="http://camel.apache.org/schema/spring">
  <route>
    <from uri="direct:start"/>
    <to uri="kamelet:foo/bar?baz=test"/>
  </route>
</camelContext>
`

const xmlWireTapEndpoint = `
<camelContext xmlns="http://camel.apache.org/schema/spring">
  <route>
    <from uri="direct:start"/>
    <wireTap uri="kamelet:foo/bar?baz=test"/>
  </route>
</camelContext>
`

func TestXMLKamelet(t *testing.T) {
	tc := []struct {
		source   string
		kamelets []string
	}{
		{
			source:   xmlKameletEip,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   xmlKameletEndpoint,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   xmlWireTapEndpoint,
			kamelets: []string{"foo/bar"},
		},
	}

	inspector := newTestXMLInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestXMLKamelet-%d", i), func(t *testing.T) {
			assertExtract(t, inspector, test.source, func(meta *Metadata) {
				assert.True(t, meta.RequiredCapabilities.IsEmpty())

				for _, k := range test.kamelets {
					assert.Contains(t, meta.Kamelets, k)
				}
			})
		})
	}
}

const xmlJSONEip = `
<camelContext xmlns="http://camel.apache.org/schema/spring">
  <route>
    <from uri="direct:start"/>
    <marshal><json/></marshal>
  </route>
</camelContext>
`

const xmlJSONJacksonEip = `
<camelContext xmlns="http://camel.apache.org/schema/spring">
  <route>
    <from uri="direct:start"/>
    <marshal><json library="Jackson"/></marshal>
  </route>
</camelContext>
`

const xmlAvroEndpoint = `
<camelContext xmlns="http://camel.apache.org/schema/spring">
  <route>
    <from uri="direct:start"/>
    <to uri="dataformat:avro:marshal"/>
  </route>
</camelContext>
`

const xmlJacksonEndpoint = `
<camelContext xmlns="http://camel.apache.org/schema/spring">
  <route>
    <from uri="direct:start"/>
    <to uri="dataformat:jackson:marshal"/>
  </route>
</camelContext>
`

const xmlProtobufEndpoint = `
<camelContext xmlns="http://camel.apache.org/schema/spring">
  <route>
    <from uri="direct:start"/>
    <to uri="dataformat:protobuf:marshal"/>
  </route>
</camelContext>
`

func TestXMLDataFormat(t *testing.T) {
	tc := []struct {
		source string
		deps   []string
	}{
		{
			source: xmlJSONEip,
			deps:   []string{"camel:jackson"},
		},
		{
			source: xmlJSONJacksonEip,
			deps:   []string{"camel:jackson"},
		},
		{
			source: xmlAvroEndpoint,
			deps:   []string{"camel:dataformat", "camel:avro"},
		},
		{
			source: xmlJacksonEndpoint,
			deps:   []string{"camel:dataformat", "camel:jackson"},
		},
		{
			source: xmlProtobufEndpoint,
			deps:   []string{"camel:dataformat", "camel:protobuf"},
		},
	}

	inspector := newTestXMLInspector(t)
	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestXMLDataFormat-%d", i), func(t *testing.T) {
			assertExtract(t, inspector, test.source, func(meta *Metadata) {
				for _, d := range test.deps {
					assert.Contains(t, meta.Dependencies.List(), d)
				}
			})
		})
	}
}

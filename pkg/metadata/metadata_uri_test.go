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
	"github.com/stretchr/testify/require"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
)

func TestJava1(t *testing.T) {
	source := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "test",
			Content: `
			import org.apache.camel.builder.RouteBuilder;

			public class Sample extends RouteBuilder {
  				@Override
  				public void configure() throws Exception {
		  			from("timer:tick")
		    			.setBody(constant("-\n             r\n             o\n             c\nHello! Camel K\n             s\n             !\n"))
							.to("log:info?skipBodyLineSeparator=false");
  				}
			}
		`,
		},
		Language: v1.LanguageJavaSource,
	}

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	metadata, err := Extract(catalog, source)
	require.NoError(t, err)

	assert.Contains(t, metadata.FromURIs, "timer:tick")
	assert.Len(t, metadata.FromURIs, 1)
	assert.Contains(t, metadata.ToURIs, "log:info?skipBodyLineSeparator=false")
	assert.Len(t, metadata.ToURIs, 1)
}

func TestJava2(t *testing.T) {
	source := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "test",
			Content: `
			import org.apache.camel.builder.RouteBuilder;

			public class Sample extends RouteBuilder {
  				@Override
  				public void configure() throws Exception {
		  			from("timer:tick")
		    			.setBody(constant("!\n"))
							.to           (

											"log:info?skipBodyLineSeparator=false"

							               )
							.toD("direct:2")
							.toF("direct:%s", "3");
  				}
			}
		`,
		},
		Language: v1.LanguageJavaSource,
	}

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	metadata, err := Extract(catalog, source)
	require.NoError(t, err)

	assert.Contains(t, metadata.FromURIs, "timer:tick")
	assert.Len(t, metadata.FromURIs, 1)
	assert.Contains(t, metadata.ToURIs, "log:info?skipBodyLineSeparator=false")
	assert.Contains(t, metadata.ToURIs, "direct:2")
	assert.Contains(t, metadata.ToURIs, "direct:%s") // resolution not supported yet
	assert.Len(t, metadata.ToURIs, 3)
}

func TestGroovy1(t *testing.T) {
	source := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "test",
			Content: `

		  	from( "timer:tick")
		    	.setBody().constant("aa")
				.to   ('log:info?skipBodyLineSeparator=false').to(
											'http://url' )

			from("direct:2")
		    	.setBody().constant("aa")
				.to('direct:3')
		`,
		},
		Language: v1.LanguageGroovy,
	}

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	metadata, err := Extract(catalog, source)
	require.NoError(t, err)

	assert.Contains(t, metadata.FromURIs, "timer:tick")
	assert.Contains(t, metadata.FromURIs, "direct:2")
	assert.Len(t, metadata.FromURIs, 2)
	assert.Contains(t, metadata.ToURIs, "log:info?skipBodyLineSeparator=false")
	assert.Contains(t, metadata.ToURIs, "http://url")
	assert.Contains(t, metadata.ToURIs, "direct:3")
	assert.Len(t, metadata.ToURIs, 3)
}

func TestGroovy2(t *testing.T) {
	source := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "test",
			Content: `
			rest().get("/")
				.to   ('log:info?skipBodyLineSeparator=false').to( 'http://url' )
						.toD('seda:1')
						.tony('thisisnot:anuri')
						.toD( "seda:2")
						.toF( "file:%s", "2")
		`,
		},
		Language: v1.LanguageGroovy,
	}

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	metadata, err := Extract(catalog, source)
	require.NoError(t, err)

	assert.Empty(t, metadata.FromURIs)
	assert.Contains(t, metadata.ToURIs, "log:info?skipBodyLineSeparator=false")
	assert.Contains(t, metadata.ToURIs, "http://url")
	assert.Contains(t, metadata.ToURIs, "seda:1")
	assert.Contains(t, metadata.ToURIs, "seda:2")
	assert.Contains(t, metadata.ToURIs, "file:%s") // resolution not supported yet
	assert.Len(t, metadata.ToURIs, 5)
}

func TestXml1(t *testing.T) {
	source := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "test",
			Content: `
			<routes>
			    <route id="hello">
        			<from uri="timer:hello?period=3000"/>
        			<setBody>
            			<constant>Hello World!!!</constant>
        			</setBody>
        			<to uri="log:info"/>
					<to uri="log:info2"/>
					<toD uri="log:info3"/>
    			</route>
			</routes>
		`,
		},
		Language: v1.LanguageXML,
	}

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	metadata, err := Extract(catalog, source)
	require.NoError(t, err)

	assert.Contains(t, metadata.FromURIs, "timer:hello?period=3000")
	assert.Len(t, metadata.FromURIs, 1)
	assert.Contains(t, metadata.ToURIs, "log:info")
	assert.Contains(t, metadata.ToURIs, "log:info2")
	assert.Contains(t, metadata.ToURIs, "log:info3")
	assert.Len(t, metadata.ToURIs, 3)
}

func TestKotlin1(t *testing.T) {
	source := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "test",
			Content: `

		  	from( "timer:tick")
		    	.setBody().constant("aa")
				.to   ("log:info?skipBodyLineSeparator=false").to(
											"http://url" )

			from("direct:2")
		    	.setBody().constant("aa")
				.to("direct:3")
				.toD("direct:4")
				.toF("direct:%s", 5)
		`,
		},
		Language: v1.LanguageKotlin,
	}

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	metadata, err := Extract(catalog, source)
	require.NoError(t, err)

	assert.Contains(t, metadata.FromURIs, "timer:tick")
	assert.Contains(t, metadata.FromURIs, "direct:2")
	assert.Len(t, metadata.FromURIs, 2)
	assert.Contains(t, metadata.ToURIs, "log:info?skipBodyLineSeparator=false")
	assert.Contains(t, metadata.ToURIs, "http://url")
	assert.Contains(t, metadata.ToURIs, "direct:3")
	assert.Contains(t, metadata.ToURIs, "direct:4")
	assert.Contains(t, metadata.ToURIs, "direct:%s") // resolution not supported yet
	assert.Len(t, metadata.ToURIs, 5)
}

func TestJavascript1(t *testing.T) {
	source := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "test",
			Content: `

			rest().get("/")
				.to   ('log:info?skipBodyLineSeparator=false').to( 'http://url' )
				.toD("direct:2")
				.toF("direct:%s", "3")
		`,
		},
		Language: v1.LanguageJavaScript,
	}

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	metadata, err := Extract(catalog, source)
	require.NoError(t, err)

	assert.Empty(t, metadata.FromURIs)
	assert.Contains(t, metadata.ToURIs, "log:info?skipBodyLineSeparator=false")
	assert.Contains(t, metadata.ToURIs, "http://url")
	assert.Contains(t, metadata.ToURIs, "direct:2")
	assert.Contains(t, metadata.ToURIs, "direct:%s") // resolution not supported yet
	assert.Len(t, metadata.ToURIs, 4)
}

const yaml = `
- from:
    uri: "timer:tick"
    steps:
        - to: "log:info1"
        - to:
            uri: "log:info2"
        - split:
            tokenizer: 't'
            steps:
                - to: "log:info3"

`

func TestJYaml(t *testing.T) {
	source := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name:    "test",
			Content: yaml,
		},
		Language: v1.LanguageYaml,
	}

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	metadata, err := Extract(catalog, source)
	require.NoError(t, err)

	assert.NotEmpty(t, metadata.FromURIs)
	assert.Contains(t, metadata.FromURIs, "timer:tick")
	assert.Len(t, metadata.FromURIs, 1)

	assert.NotEmpty(t, metadata.ToURIs)
	assert.Contains(t, metadata.ToURIs, "log:info1")
	assert.Contains(t, metadata.ToURIs, "log:info2")
	assert.Contains(t, metadata.ToURIs, "log:info3")
	assert.Len(t, metadata.ToURIs, 3)
}

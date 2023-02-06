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

package modeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGroovyFile(t *testing.T) {
	it := `
		//     camel-k: pippo=pluto     paperino ciao=1   
		// camel-k : ciao

		from("timer:tick").log("Ciao")
    `
	opts, err := Parse("simple.groovy", it)
	assert.NoError(t, err)
	assert.Len(t, opts, 4)
	assert.Contains(t, opts, Option{Name: "pippo", Value: "pluto"})
	assert.Contains(t, opts, Option{Name: "paperino"})
	assert.Contains(t, opts, Option{Name: "ciao", Value: "1"})
	assert.Contains(t, opts, Option{Name: "ciao"})
}

func TestParseKotlinFile(t *testing.T) {
	it := `
		//     camel-k: pippo=pluto     paperino ciao=1   
		// camel-k : ciao

		from("timer:tick").log("Ciao")
    `
	opts, err := Parse("example.kts", it)
	assert.NoError(t, err)
	assert.Len(t, opts, 4)
	assert.Contains(t, opts, Option{Name: "pippo", Value: "pluto"})
	assert.Contains(t, opts, Option{Name: "paperino"})
	assert.Contains(t, opts, Option{Name: "ciao", Value: "1"})
	assert.Contains(t, opts, Option{Name: "ciao"})
}

func TestParseJavaFile(t *testing.T) {
	it := `
		//     camel-k: pippo=pluto     paperino ciao=1   
		// camel-k : ciao

		import org.apache.camel.builder.RouteBuilder;
		
		public class {{ .Name }} extends RouteBuilder {
		  @Override
		  public void configure() throws Exception {
		
			  // Write your routes here, for example:
			  from("timer:java?period=1000")
				.routeId("java")
				.setBody()
				  .simple("Hello Camel K from ${routeId}")
				.to("log:info");
		
		  }
		}
    `
	opts, err := Parse("Example.java", it)
	assert.NoError(t, err)
	assert.Len(t, opts, 4)
	assert.Contains(t, opts, Option{Name: "pippo", Value: "pluto"})
	assert.Contains(t, opts, Option{Name: "paperino"})
	assert.Contains(t, opts, Option{Name: "ciao", Value: "1"})
	assert.Contains(t, opts, Option{Name: "ciao"})
}

func TestParseJSFile(t *testing.T) {
	it := `
		//     camel-k: pippo=pluto     paperino ciao=1   
		// camel-k : ciao 
		// Write your routes here, for example:
		from('timer:js?period=1000')
			.routeId('js')
			.setBody()
			  .simple('Hello Camel K from ${routeId}')
			.to('log:info')
    `
	opts, err := Parse("example.js", it)
	assert.NoError(t, err)
	assert.Len(t, opts, 4)
	assert.Contains(t, opts, Option{Name: "pippo", Value: "pluto"})
	assert.Contains(t, opts, Option{Name: "paperino"})
	assert.Contains(t, opts, Option{Name: "ciao", Value: "1"})
	assert.Contains(t, opts, Option{Name: "ciao"})
}

func TestParseYAMLFile(t *testing.T) {
	it := `
		#     camel-k: pippo=pluto     paperino ciao=1   
		### camel-k : ciao 

		# Write your routes here, for example:
		- from:
			uri: "timer:yaml"
			parameters:
			  period: "1000"
			steps:
			  - set-body:
				  constant: "Hello Camel K from yaml"
			  - to: "log:info"

    `
	opts, err := Parse("example.yaml", it)
	assert.NoError(t, err)
	assert.Len(t, opts, 4)
	assert.Contains(t, opts, Option{Name: "pippo", Value: "pluto"})
	assert.Contains(t, opts, Option{Name: "paperino"})
	assert.Contains(t, opts, Option{Name: "ciao", Value: "1"})
	assert.Contains(t, opts, Option{Name: "ciao"})
}

func TestParseXMLFile(t *testing.T) {
	it := `
		#     camel-k: pippo=pluto     paperino ciao=1   
		### camel-k : ciao 

		<?xml version="1.0" encoding="UTF-8"?>
		<!--     camel-k: pippo=pluto     paperino ciao=1-->
		<!--camel-k : ciao -->
		<!--     camel-k: language=xml -->

		<routes xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
				xmlns="http://camel.apache.org/schema/spring"
				xsi:schemaLocation="
					http://camel.apache.org/schema/spring
					https://camel.apache.org/schema/spring/camel-spring.xsd">
		
			<!-- Write your routes here, for example: -->
			<route id="xml">
				<from uri="timer:xml?period=1000"/>
				<setBody>
					<simple>Hello Camel K from ${routeId}</simple>
				</setBody>
				<to uri="log:info"/>
			</route>
		
		</routes>


    `
	opts, err := Parse("example.xml", it)
	assert.NoError(t, err)
	assert.Len(t, opts, 5)
	assert.Contains(t, opts, Option{Name: "pippo", Value: "pluto"})
	assert.Contains(t, opts, Option{Name: "paperino"})
	assert.Contains(t, opts, Option{Name: "ciao", Value: "1"})
	assert.Contains(t, opts, Option{Name: "ciao"})
	assert.Contains(t, opts, Option{Name: "language", Value: "xml"})
}

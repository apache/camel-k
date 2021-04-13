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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
)

func TestDependenciesJavaSource(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "Request.java",
			Content: `
			    import org.apache.camel.component.activemq.ActiveMQComponent;

			    from("telegram:bots/cippa").to("log:stash");
			    from("timer:tick").to("amqp:queue");
			    from("ine:xistent").to("amqp:queue");
				from("twitter-search:{{twitterKeywords}}"
                    + "?delay={{twitterDelayMs}}");
			`,
		},
		Language: v1.LanguageJavaSource,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:activemq",
			"camel:amqp",
			"camel:log",
			"camel:telegram",
			"camel:timer",
			"camel:twitter",
		},
		meta.Dependencies.List())
}

func TestDependenciesJavaScript(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "source.js",
			Content: `
			    var component = Java.type("org.apache.camel.component.activemq.ActiveMQComponent");

			    from('telegram:bots/cippa').to("log:stash");
			    from('timer:tick').to("amqp:queue");
			    from("ine:xistent").to("amqp:queue");
			    '"'
		    `,
		},
		Language: v1.LanguageJavaScript,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:activemq",
			"camel:amqp",
			"camel:log",
			"camel:telegram",
			"camel:timer",
		},
		meta.Dependencies.List())
}

func TestDependenciesGroovy(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "source.groovy",
			Content: `
			    import org.apache.camel.component.activemq.ActiveMQComponent;

			    from('telegram:bots/cippa').to("log:stash");
			    from('timer:tick').to("amqp:queue");
			    from("ine:xistent").to("amqp:queue");
				from('twitter-search:{{twitterKeywords}}'
                    + '?delay={{twitterDelayMs}}');
			    '"
		    `,
		},
		Language: v1.LanguageGroovy,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:activemq",
			"camel:amqp",
			"camel:log",
			"camel:telegram",
			"camel:timer",
			"camel:twitter",
		},
		meta.Dependencies.List())
}

func TestDependencies(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "Request.java",
			Content: `
			    from("http:test").to("log:end");
			    from("https:test").to("log:end");
			    from("twitter-timeline:test").to("mock:end");
		    `,
		},
		Language: v1.LanguageJavaSource,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t,
		[]string{
			"camel:http",
			"camel:log",
			"camel:mock",
			"camel:twitter",
		},
		meta.Dependencies.List())
}

func TestDependenciesQuarkus(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "Request.java",
			Content: `
			    import org.apache.camel.component.timer.TimerComponent;

			    from("http:test").to("log:end");
			    from("https:test").to("log:end");
			    from("twitter-timeline:test").to("mock:end");
			    from("direct:start").circuitBreaker().faultToleranceConfiguration().timeoutEnabled(true).timeoutDuration(1500).end()
			    .to("direct:other").onFallback().setBody(constant("Fallback response")).end();
		    `,
		},
		Language: v1.LanguageJavaSource,
	}

	catalog, err := camel.QuarkusCatalog()
	assert.Nil(t, err)
	assert.NotNil(t, catalog)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t,
		[]string{
			"camel:http",
			"camel:log",
			"camel:mock",
			"camel:timer",
			"camel:twitter",
			"camel:direct",
		},
		meta.Dependencies.List())

	assert.True(t, meta.RequiredCapabilities.Has(v1.CapabilityCircuitBreaker))
}

func TestJacksonDependency(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "Request.java",
			Content: `
			    from("http:test").unmarshal().json(JsonLibrary.Jackson).to("log:end");
		    `,
		},
		Language: v1.LanguageJavaSource,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:http",
			"camel:jackson",
			"camel:log",
		},
		meta.Dependencies.List())
}

func TestJacksonImplicitDependency(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "Request.groovy",
			Content: `
			    from("http:test")
					.unmarshal().json()
					.to("log:end")
		    `,
		},
		Language: v1.LanguageGroovy,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:http",
			"camel:jackson",
			"camel:log",
		},
		meta.Dependencies.List())
}

func TestLanguageDependencies(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "Languages.java",
			Content: `
				from("direct:start")
				.transform().ognl("request.body.name == 'Camel K'")
				.transform().simple("${body.toUpperCase()}")
				.transform().mvel("resource:classpath:script.mvel")
				.transform().xquery("/ns:foo/bar", String.class, new Namespaces("ns", "http://foo/bar"))
				.transform().xpath("//foo/bar")
				.transform().jsonpath("$.foo")
				.transform().groovy("request.body += 'modified'")
				.split().xtokenize("/ns:foo/bar", new Namespaces("ns", "http://foo/bar"));
			`,
		},
		Language: v1.LanguageJavaSource,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)
	assert.ElementsMatch(
		t,
		[]string{
			"camel:direct",
			"camel:mvel",
			"camel:ognl",
			"camel:xpath",
			"camel:jsonpath",
			"camel:groovy",
			"camel:saxon",
			"camel:xml-jaxp",
		},
		meta.Dependencies.List())
}

func TestLanguageDependenciesTransformExpression(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "Languages.java",
			Content: `
				from("direct:start")
				.transform(language("ognl", "request.body.name == 'Camel K'"))
				.transform(simple("${body.toUpperCase()}"))
				.transform(xpath("//foo/bar"))
				.transform(jsonpath("$.foo"))
			`,
		},
		Language: v1.LanguageJavaSource,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)
	assert.ElementsMatch(
		t,
		[]string{
			"camel:direct",
			"camel:ognl",
			"camel:xpath",
			"camel:jsonpath",
		},
		meta.Dependencies.List())
}

func TestCircuitBreakerDependency(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "Request.groovy",
			Content: `
			    from("http:test")
					.circuitBreaker()
						.to("log:end")
					.onFallback()
						.to("log:fallback")
		    `,
		},
		Language: v1.LanguageGroovy,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t,
		[]string{
			"camel:http",
			"camel:log",
		},
		meta.Dependencies.List())

	assert.True(t, meta.RequiredCapabilities.Has(v1.CapabilityCircuitBreaker))
}

func TestRestDependency(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "Request.groovy",
			Content: `
                rest()
                    .get("/api")
                    .to("direct:get")
			    from("http:test")
                    .to("log:info")
		    `,
		},
		Language: v1.LanguageGroovy,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:http",
			"camel:direct",
			"camel:log",
			"mvn:org.apache.camel.quarkus:camel-quarkus-rest",
			"mvn:org.apache.camel.quarkus:camel-quarkus-platform-http",
		},
		meta.Dependencies.List())
}

func TestRestWithPathDependency(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "Request.groovy",
			Content: `
                rest("/test")
                    .get("/api")
                    .to("direct:get")
			    from("http:test")
                    .to("log:info")
		    `,
		},
		Language: v1.LanguageGroovy,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t, []string{
			"camel:http",
			"camel:direct",
			"camel:log",
			"mvn:org.apache.camel.quarkus:camel-quarkus-rest",
			"mvn:org.apache.camel.quarkus:camel-quarkus-platform-http",
		},
		meta.Dependencies.List())
}

func TestRestConfigurationDependency(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "Request.groovy",
			Content: `
                restConfiguration()
                    .component("netty-http")
			    from("http:test")
                    .to("log:info")
		    `,
		},
		Language: v1.LanguageGroovy,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t, []string{
			"camel:http",
			"camel:log",
			"mvn:org.apache.camel.quarkus:camel-quarkus-rest",
			"mvn:org.apache.camel.quarkus:camel-quarkus-platform-http",
		},
		meta.Dependencies.List())
}

func TestRestClosureDependencyGroovy(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "Request.groovy",
			Content: `
                rest {
                }
			    from("http:test")
                    .to("log:info")
		    `,
		},
		Language: v1.LanguageGroovy,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:http",
			"camel:log",
			"mvn:org.apache.camel.quarkus:camel-quarkus-rest",
			"mvn:org.apache.camel.quarkus:camel-quarkus-platform-http",
		},
		meta.Dependencies.List())
}

func TestRestClosureDependencyKotlin(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "Request.groovy",
			Content: `
                rest {
                }
			    from("http:test")
                    .to("log:info")
		    `,
		},
		Language: v1.LanguageKotlin,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:http",
			"camel:log",
			"mvn:org.apache.camel.quarkus:camel-quarkus-rest",
			"mvn:org.apache.camel.quarkus:camel-quarkus-platform-http",
		},
		meta.Dependencies.List())
}

func TestXMLCircuitBreakerDependency(t *testing.T) {
	code := v1.SourceSpec{

		DataSpec: v1.DataSpec{
			Name: "routes.xml",
			Content: `
			<from uri="direct:ciao" />
			<circuitBreaker>
				<to uri="log:info" />
				<onFallback>
					<to uri="kafka:topic" />
				</onFallback>
			</circuitBreaker>
		`,
		},
		Language: v1.LanguageXML,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:direct",
			"camel:kafka",
			"camel:log",
		},
		meta.Dependencies.List())

	assert.True(t, meta.RequiredCapabilities.Has(v1.CapabilityCircuitBreaker))
}

func TestXMLRestDependency(t *testing.T) {
	code := v1.SourceSpec{

		DataSpec: v1.DataSpec{
			Name: "routes.xml",
			Content: `
			<rest path="/say">
		      <get uri="/hello">
		        <to uri="direct:hello"/>
		      </get>
		      <get uri="/bye" consumes="application/json">
		        <to uri="direct:bye"/>
		      </get>
		      <post uri="/bye">
		        <to uri="mock:update"/>
		      </post>
		    </rest>
		`,
		},
		Language: v1.LanguageXML,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:direct",
			"camel:mock",
		},
		meta.Dependencies.List())

	assert.True(t, meta.RequiredCapabilities.Has("rest"))
}

func TestXMLLanguageDependencies(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "routes.xml",
			Content: `
               <from uri="direct:start" />
               <transform>
                 <language language="ognl">request.body.name == 'Camel K'</language>
               </transform>
               <transform>
                 <simple>${body.toUpperCase()}</simple>
               </transform>
               <transform>
                 <mvel>resource:classpath:script.mvel</mvel>
               </transform>
               <transform>
                 <jsonpath>$.foo</jsonpath>
               </transform>
               <transform>
                 <groovy>request.body += 'modified'</groovy>
               </transform>
               <transform>
                 <tokenize>request.body += 'modified'</tokenize>
               </transform>
               <transform>
                 <xtokenize>/ns:foo/bar</xtokenize>
               </transform>
               <transform>
                 <xpath>//foo/bar</xpath>
               </transform>
               <transform>
                 <xquery>//ns:foo/bar</xquery>
               </transform>
               <split>
                 <tokenize token=","/>
               </split>
          `,
		},
		Language: v1.LanguageXML,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)
	assert.ElementsMatch(
		t,
		[]string{
			"camel:core",
			"camel:direct",
			"camel:mvel",
			"camel:ognl",
			"camel:xpath",
			"camel:xml-jaxp",
			"camel:jsonpath",
			"camel:saxon",
			"camel:groovy",
		},
		meta.Dependencies.List())
}

const yamlWithRest = `
- rest:
    path: "/"
    steps:
        - to: "log:info"
        - to: "direct:hello"
`
const yamlWithCircuitBreaker = `
- from:
    uri: "direct:start"
    steps:
        - circuitBreaker:
            todo: "not implemented"
`

const yamlWithLanguages = `
- from:
    uri: "direct:start"
    steps:
        - set-body:
            constant: "Hello Camel K"
        - transform:
            language:
                language: "ognl"
                expression: "request.body.name == 'Camel K'"
        - transform:
            simple: "${body.toUpperCase()}"
        - transform:
            xquery: "/ns:foo/bar"
        - transform:
            xpath: "//foo/bar"
        - transform:
            jsonpath: "$.foo"
        - transform:
            groovy: "request.body += 'modified'"
        - split:
            xtokenize: "/ns:foo/bar"
`

func TestYAMLRestDependency(t *testing.T) {
	code := v1.SourceSpec{

		DataSpec: v1.DataSpec{
			Name:    "routes.yaml",
			Content: yamlWithRest,
		},
		Language: v1.LanguageYaml,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:direct",
			"camel:log",
		},
		meta.Dependencies.List())

	assert.True(t, meta.RequiredCapabilities.Has("rest"))
}

func TestYAMLCircuitBreakerDependency(t *testing.T) {
	code := v1.SourceSpec{

		DataSpec: v1.DataSpec{
			Name:    "routes.yaml",
			Content: yamlWithCircuitBreaker,
		},
		Language: v1.LanguageYaml,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:direct",
		},
		meta.Dependencies.List())
	assert.True(t, meta.RequiredCapabilities.Has(v1.CapabilityCircuitBreaker))
}

func TestYAMLLanguageDependencies(t *testing.T) {
	code := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name:    "routes.yaml",
			Content: yamlWithLanguages,
		},
		Language: v1.LanguageYaml,
	}

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(
		t,
		[]string{
			"camel:core",
			"camel:direct",
			"camel:ognl",
			"camel:xpath",
			"camel:jsonpath",
			"camel:groovy",
			"camel:saxon",
			"camel:xml-jaxp",
		},
		meta.Dependencies.List())
}

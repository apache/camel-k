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

	"github.com/apache/camel-k/pkg/util/test"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestDependenciesJavaSource(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Request.java",
			Content: `
			    from("telegram:bots/cippa").to("log:stash");
			    from("timer:tick").to("amqp:queue");
			    from("ine:xistent").to("amqp:queue");
				from("twitter-search:{{twitterKeywords}}"
                    + "?delay={{twitterDelayMs}}");
			`,
		},
		Language: v1alpha1.LanguageJavaSource,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)
	// assert all dependencies are found and sorted (removing duplicates)
	assert.Equal(t, []string{"camel:amqp", "camel:core", "camel:telegram", "camel:twitter"}, meta.Dependencies)
}

func TestDependenciesJavaScript(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "source.js",
			Content: `
			    from('telegram:bots/cippa').to("log:stash");
			    from('timer:tick').to("amqp:queue");
			    from("ine:xistent").to("amqp:queue");
			    '"'
		    `,
		},
		Language: v1alpha1.LanguageJavaScript,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	// assert all dependencies are found and sorted (removing duplicates)
	assert.Equal(t, []string{"camel:amqp", "camel:core", "camel:telegram"}, meta.Dependencies)
}

func TestDependenciesGroovy(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "source.groovy",
			Content: `
			    from('telegram:bots/cippa').to("log:stash");
			    from('timer:tick').to("amqp:queue");
			    from("ine:xistent").to("amqp:queue");
				from('twitter-search:{{twitterKeywords}}'
                    + '?delay={{twitterDelayMs}}');
			    '"
		    `,
		},
		Language: v1alpha1.LanguageGroovy,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	// assert all dependencies are found and sorted (removing duplicates)
	assert.Equal(t, []string{"camel:amqp", "camel:core", "camel:telegram", "camel:twitter"}, meta.Dependencies)
}

func TestDependencies(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Request.java",
			Content: `
			    from("http4:test").to("log:end");
			    from("https4:test").to("log:end");
			    from("twitter-timeline:test").to("mock:end");
		    `,
		},
		Language: v1alpha1.LanguageJavaSource,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	// assert all dependencies are found and sorted (removing duplicates)
	assert.Equal(t, []string{"camel:core", "camel:http4", "camel:twitter"}, meta.Dependencies)
}

func TestJacksonDependency(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Request.java",
			Content: `
			    from("http4:test").unmarshal().json(JsonLibrary.Jackson).to("log:end");
		    `,
		},
		Language: v1alpha1.LanguageJavaSource,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	// assert all dependencies are found and sorted (removing duplicates)
	assert.Equal(t, []string{"camel:core", "camel:http4", "camel:jackson"}, meta.Dependencies)
}

func TestHystrixDependency(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Request.groovy",
			Content: `
			    from("http4:test")
					.hystrix()
						.to("log:end")
					.onFallback()
						.to("log:fallback")
		    `,
		},
		Language: v1alpha1.LanguageGroovy,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	// assert all dependencies are found and sorted (removing duplicates)
	assert.Equal(t, []string{"camel:core", "camel:http4", "camel:hystrix"}, meta.Dependencies)
}

func TestXMLHystrixDependency(t *testing.T) {
	code := v1alpha1.SourceSpec{

		DataSpec: v1alpha1.DataSpec{
			Name: "routes.xml",
			Content: `
			<from uri="direct:ciao" />
			<hystrix>
				<to uri="log:info" />
				<onFallback>
					<to uri="kafka:topic" />
				</onFallback>
			</hystrix>
		`,
		},
		Language: v1alpha1.LanguageXML,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	// assert all dependencies are found and sorted (removing duplicates)
	assert.Equal(t, []string{"camel:core", "camel:hystrix", "camel:kafka"}, meta.Dependencies)
}
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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestHttpJavaSource(t *testing.T) {
	code := v1alpha1.SourceSpec{
		Name:     "Request.java",
		Language: v1alpha1.LanguageJavaSource,
		Content: `
			from("telegram:bots/cippa").to("log:stash");
			from("undertow:uri").to("log:stash");
			from("ine:xistent").to("log:stash");
		`,
	}
	meta := Extract(code)
	assert.True(t, meta.RequiresHTTPService)
	assert.False(t, meta.PassiveEndpoints)
}

func TestHttpOnlyJavaSource(t *testing.T) {
	code := v1alpha1.SourceSpec{
		Name:     "Request.java",
		Language: v1alpha1.LanguageJavaSource,
		Content: `
			from("direct:bots/cippa").to("log:stash");
			from("undertow:uri").to("log:stash");
			from("seda:path").to("log:stash");
		`,
	}
	meta := Extract(code)
	assert.True(t, meta.RequiresHTTPService)
	assert.True(t, meta.PassiveEndpoints)
}

func TestHttpOnlyJavaSourceRest(t *testing.T) {
	code := v1alpha1.SourceSpec{
		Name:     "Request.java",
		Language: v1alpha1.LanguageJavaSource,
		Content: `
			from("direct:bots/cippa").to("log:stash");
			rest().get("").to("log:stash");
		`,
	}
	meta := Extract(code)
	assert.True(t, meta.RequiresHTTPService)
	assert.True(t, meta.PassiveEndpoints)
}

func TestHttpOnlyJavaSourceRest2(t *testing.T) {
	code := v1alpha1.SourceSpec{
		Name:     "Request.java",
		Language: v1alpha1.LanguageJavaSource,
		Content: `
			from("vm:bots/cippa").to("log:stash");
			rest( ).get("").to("log:stash");
		`,
	}
	meta := Extract(code)
	assert.True(t, meta.RequiresHTTPService)
	assert.True(t, meta.PassiveEndpoints)
}

func TestNoHttpGroovySource(t *testing.T) {
	code := v1alpha1.SourceSpec{
		Name:     "Request.groovy",
		Language: v1alpha1.LanguageGroovy,
		Content: `
			from('direct:bots/cippa').to("log:stash");
			from('teelgram:uri').to("log:stash");
			from('seda:path').to("log:stash");
		`,
	}
	meta := Extract(code)
	assert.False(t, meta.RequiresHTTPService)
	assert.False(t, meta.PassiveEndpoints)
}

func TestHttpOnlyGroovySource(t *testing.T) {
	code := v1alpha1.SourceSpec{
		Name:     "Request.groovy",
		Language: v1alpha1.LanguageGroovy,
		Content: `
			from('direct:bots/cippa').to("log:stash");
			from('undertow:uri').to("log:stash");
			from('seda:path').to("log:stash");
		`,
	}
	meta := Extract(code)
	assert.True(t, meta.RequiresHTTPService)
	assert.True(t, meta.PassiveEndpoints)
}

func TestHttpXMLSource(t *testing.T) {
	code := v1alpha1.SourceSpec{
		Name:     "routes.xml",
		Language: v1alpha1.LanguageXML,
		Content: `
			<from uri="telegram:ciao" />
			<rest path="/">
			</rest>
		`,
	}
	meta := Extract(code)
	assert.True(t, meta.RequiresHTTPService)
	assert.False(t, meta.PassiveEndpoints)
}

func TestHttpOnlyXMLSource(t *testing.T) {
	code := v1alpha1.SourceSpec{
		Name:     "routes.xml",
		Language: v1alpha1.LanguageXML,
		Content: `
			<from uri="direct:ciao" />
			<rest path="/">
			</rest>
		`,
	}
	meta := Extract(code)
	assert.True(t, meta.RequiresHTTPService)
	assert.True(t, meta.PassiveEndpoints)
}

func TestMultilangHTTPOnlySource(t *testing.T) {
	codes := []v1alpha1.SourceSpec{
		{
			Name:     "routes.xml",
			Language: v1alpha1.LanguageXML,
			Content: `
				<from uri="direct:ciao" />
				<rest path="/">
				</rest>
			`,
		},
		{
			Name:     "routes2.groovy",
			Language: v1alpha1.LanguageGroovy,
			Content: `
				from('seda:in').to('seda:out')
			`,
		},
	}
	meta := ExtractAll(codes)
	assert.True(t, meta.RequiresHTTPService)
	assert.True(t, meta.PassiveEndpoints)
}

func TestMultilangHTTPSource(t *testing.T) {
	codes := []v1alpha1.SourceSpec{
		{
			Name:     "routes.xml",
			Language: v1alpha1.LanguageXML,
			Content: `
				<from uri="direct:ciao" />
				<rest path="/">
				</rest>
			`,
		},
		{
			Name:     "routes2.groovy",
			Language: v1alpha1.LanguageGroovy,
			Content: `
				from('seda:in').to('seda:out')
				from('timer:tick').to('log:info')
			`,
		},
	}
	meta := ExtractAll(codes)
	assert.True(t, meta.RequiresHTTPService)
	assert.False(t, meta.PassiveEndpoints)
}

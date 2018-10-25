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

func TestDependenciesJavaSource(t *testing.T) {
	code := v1alpha1.SourceSpec{
		Name:     "Request.java",
		Language: v1alpha1.LanguageJavaSource,
		Content: `
			from("telegram:bots/cippa").to("log:stash");
			from("timer:tick").to("amqp:queue");
			from("ine:xistent").to("amqp:queue");
		`,
	}
	meta := Extract(code)
	// assert all dependencies are found and sorted (removing duplicates)
	assert.Equal(t, []string{"camel:amqp", "camel:core", "camel:telegram"}, meta.Dependencies)
}

func TestDependenciesJavaClass(t *testing.T) {
	code := v1alpha1.SourceSpec{
		Name:     "Request.class",
		Language: v1alpha1.LanguageJavaClass,
		Content: `
			from("telegram:bots/cippa").to("log:stash");
			from("timer:tick").to("amqp:queue");
			from("ine:xistent").to("amqp:queue");
		`,
	}
	meta := Extract(code)
	assert.Empty(t, meta.Dependencies)
}

func TestDependenciesJavaScript(t *testing.T) {
	code := v1alpha1.SourceSpec{
		Name:     "source.js",
		Language: v1alpha1.LanguageJavaScript,
		Content: `
			from('telegram:bots/cippa').to("log:stash");
			from('timer:tick').to("amqp:queue");
			from("ine:xistent").to("amqp:queue");
			'"'
		`,
	}
	meta := Extract(code)
	// assert all dependencies are found and sorted (removing duplicates)
	assert.Equal(t, []string{"camel:amqp", "camel:core", "camel:telegram"}, meta.Dependencies)
}

func TestDependenciesGroovy(t *testing.T) {
	code := v1alpha1.SourceSpec{
		Name:     "source.groovy",
		Language: v1alpha1.LanguageGroovy,
		Content: `
			from('telegram:bots/cippa').to("log:stash");
			from('timer:tick').to("amqp:queue");
			from("ine:xistent").to("amqp:queue");
			'"'
		`,
	}
	meta := Extract(code)
	// assert all dependencies are found and sorted (removing duplicates)
	assert.Equal(t, []string{"camel:amqp", "camel:core", "camel:telegram"}, meta.Dependencies)
}

func TestDependencies(t *testing.T) {
	code := v1alpha1.SourceSpec{
		Name:     "Request.java",
		Language: v1alpha1.LanguageJavaSource,
		Content: `
			from("http4:test").to("log:end");
			from("https4:test").to("log:end");
			from("twitter-timeline:test").to("mock:end");
		`,
	}
	meta := Extract(code)
	// assert all dependencies are found and sorted (removing duplicates)
	assert.Equal(t, []string{"camel:core", "camel:http4", "camel:twitter"}, meta.Dependencies)
}

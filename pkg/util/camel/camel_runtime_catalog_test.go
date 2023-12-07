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

package camel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoaderByArtifact(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)
	assert.Equal(t, "groovy", catalog.loaderByArtifact["camel-quarkus-groovy-dsl"])
	assert.Equal(t, "java", catalog.loaderByArtifact["camel-quarkus-java-joor-dsl"])
	assert.Equal(t, "js", catalog.loaderByArtifact["camel-quarkus-js-dsl"])
	assert.Equal(t, "kts", catalog.loaderByArtifact["camel-quarkus-kotlin-dsl"])
	assert.Equal(t, "xml", catalog.loaderByArtifact["camel-quarkus-xml-io-dsl"])
	assert.Equal(t, "yaml", catalog.loaderByArtifact["camel-quarkus-yaml-dsl"])
}

func TestHasLoaderByArtifact(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)
	assert.True(t, catalog.HasLoaderByArtifact("groovy-dsl"))
	assert.True(t, catalog.HasLoaderByArtifact("java-joor-dsl"))
	assert.True(t, catalog.HasLoaderByArtifact("js-dsl"))
	assert.True(t, catalog.HasLoaderByArtifact("kotlin-dsl"))
	assert.True(t, catalog.HasLoaderByArtifact("xml-io-dsl"))
	assert.True(t, catalog.HasLoaderByArtifact("yaml-dsl"))
	assert.False(t, catalog.HasLoaderByArtifact("python-dsl"))
}

func TestIsResolvable(t *testing.T) {
	catalog, err := DefaultCatalog()
	require.NoError(t, err)

	testCases := []struct {
		desc     string
		uri      string
		expected bool
	}{
		// static dependencies
		{desc: "Basic static dependency", uri: "log:info", expected: true},
		{desc: "Basic static dependency with path and param", uri: "http://my-site/test?param=value", expected: true},
		{desc: "Basic static dependency with path and param placeholder", uri: "http://my-site/test?{{params}}", expected: true},
		{desc: "Basic static dependency with path placeholder and param", uri: "http://my-site/{{path}}?key=val", expected: true},

		// placeholders
		{desc: "Basic", uri: "{{url}}", expected: false},
		{desc: "With query param placeholder", uri: "{{url}}?authMethod={{authMethod}}", expected: false},
		{desc: "With query path and param placeholders 1", uri: "{{url}}/test?authMethod={{authMethod}}", expected: false},
		{desc: "With query path and param placeholders 2", uri: "{{url}}/test?authMethod={{authMethod}}&key=val", expected: false},
		{desc: "With query param", uri: "{{url}}?authMethod=Basic", expected: false},
		{desc: "With query param and path", uri: "{{url}}/test", expected: false},
		{desc: "With masked AND url-encoded query params", uri: "{{url}}?authMethod=%7B%7BauthMethod%7D%7D", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			if got := catalog.IsResolvable(testCase.uri); got != testCase.expected {
				t.Errorf("IsResolvable(%v) = %v, want %v", testCase.uri, got, testCase.expected)

			}
		})
	}
}

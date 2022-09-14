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

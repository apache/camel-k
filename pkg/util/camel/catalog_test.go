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

	"github.com/apache/camel-k/version"
	"github.com/stretchr/testify/assert"
)

func TestCatalog(t *testing.T) {
	assert.NotNil(t, defaultCatalog)
	assert.NotEmpty(t, defaultCatalog.Artifacts)
}

func TestRuntimeContainsEmbeddedArtifacts(t *testing.T) {
	artifact := defaultCatalog.GetArtifactByScheme("knative")
	assert.Equal(t, 1, len(artifact.Schemes))
	assert.Equal(t, "org.apache.camel.k", artifact.GroupID)
	assert.Equal(t, "camel-knative", artifact.ArtifactID)
	assert.Equal(t, version.Version, artifact.Version)

	scheme, found := defaultCatalog.GetScheme("knative")
	assert.True(t, found)
	assert.True(t, scheme.HTTP)
}

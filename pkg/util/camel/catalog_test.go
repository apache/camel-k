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
)

func TestCamelTraitGenerateMavenProjectSucceeds(t *testing.T) {
	mvnProject := generateMavenProject("1.0.1", nil)
	assert.NotNil(t, mvnProject)
	assert.Equal(t, "org.apache.camel.k.integration", mvnProject.GroupID)
	assert.Equal(t, "camel-k-catalog-generator", mvnProject.ArtifactID)
	assert.NotNil(t, mvnProject.Build)
	assert.Equal(t, "generate-resources", mvnProject.Build.DefaultGoal)
	assert.NotNil(t, mvnProject.Build.Plugins)
	assert.Len(t, mvnProject.Build.Plugins, 1)
	assert.Equal(t, "org.apache.camel.k", mvnProject.Build.Plugins[0].GroupID)
	assert.Equal(t, "camel-k-maven-plugin", mvnProject.Build.Plugins[0].ArtifactID)
	assert.NotNil(t, mvnProject.Build.Plugins[0].Executions)
	assert.Len(t, mvnProject.Build.Plugins[0].Executions, 1)
	assert.Equal(t, "generate-catalog", mvnProject.Build.Plugins[0].Executions[0].ID)
	assert.NotNil(t, mvnProject.Build.Plugins[0].Executions[0].Goals)
	assert.Len(t, mvnProject.Build.Plugins[0].Executions[0].Goals, 1)
	assert.Equal(t, "generate-catalog", mvnProject.Build.Plugins[0].Executions[0].Goals[0])
	assert.Nil(t, mvnProject.Build.Plugins[0].Dependencies)
}

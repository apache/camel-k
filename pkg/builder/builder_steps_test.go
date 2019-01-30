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

package builder

import (
	"testing"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"

	"github.com/stretchr/testify/assert"
)

func TestGenerateProject(t *testing.T) {
	ctx := Context{
		Request: Request{
			Platform: v1alpha1.IntegrationPlatformSpec{
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					CamelVersion: "2.22.1",
				},
			},
			Repositories: []string{
				"https://repository.apache.org/content/groups/snapshots-group@id=apache-snapshots@snapshots@noreleases",
				"https://oss.sonatype.org/content/repositories/ops4j-snapshots@id=ops4j-snapshots@snapshots@noreleases",
			},
		},
	}

	err := GenerateProject(&ctx)

	assert.Nil(t, err)

	assert.Equal(t, 1, len(ctx.Project.DependencyManagement.Dependencies))
	assert.Equal(t, "org.apache.camel", ctx.Project.DependencyManagement.Dependencies[0].GroupID)
	assert.Equal(t, "camel-bom", ctx.Project.DependencyManagement.Dependencies[0].ArtifactID)
	assert.Equal(t, "2.22.1", ctx.Project.DependencyManagement.Dependencies[0].Version)
	assert.Equal(t, "pom", ctx.Project.DependencyManagement.Dependencies[0].Type)
	assert.Equal(t, "import", ctx.Project.DependencyManagement.Dependencies[0].Scope)

	assert.Equal(t, 2, len(ctx.Project.Repositories))
	assert.Equal(t, "apache-snapshots", ctx.Project.Repositories[0].ID)
	assert.False(t, ctx.Project.Repositories[0].Releases.Enabled)
	assert.True(t, ctx.Project.Repositories[0].Snapshots.Enabled)
	assert.Equal(t, "ops4j-snapshots", ctx.Project.Repositories[1].ID)
	assert.False(t, ctx.Project.Repositories[1].Releases.Enabled)
	assert.True(t, ctx.Project.Repositories[1].Snapshots.Enabled)
}

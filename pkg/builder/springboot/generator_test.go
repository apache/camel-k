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

package springboot

import (
	"testing"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestMavenRepositories(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	ctx := builder.Context{
		Catalog: catalog,
		Request: builder.Request{
			Catalog:        catalog,
			RuntimeVersion: defaults.RuntimeVersion,
			Platform: v1alpha1.IntegrationPlatformSpec{
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					CamelVersion: catalog.Version,
				},
			},
			Dependencies: []string{
				"runtime:jvm",
			},
			Repositories: []string{
				"https://repository.apache.org/content/repositories/snapshots@id=apache.snapshots@snapshots@noreleases",
				"https://oss.sonatype.org/content/repositories/snapshots/@id=sonatype.snapshots@snapshots",
			},
		},
	}

	err = GenerateProject(&ctx)
	assert.Nil(t, err)
	err = builder.InjectDependencies(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(ctx.Project.Repositories))
	assert.Equal(t, 2, len(ctx.Project.PluginRepositories))

	assert.Contains(t, ctx.Project.Repositories, maven.Repository{
		ID:        "apache.snapshots",
		URL:       "https://repository.apache.org/content/repositories/snapshots",
		Snapshots: maven.RepositoryPolicy{Enabled: true},
		Releases:  maven.RepositoryPolicy{Enabled: false},
	})

	assert.Contains(t, ctx.Project.Repositories, maven.Repository{
		ID:        "sonatype.snapshots",
		URL:       "https://oss.sonatype.org/content/repositories/snapshots/",
		Snapshots: maven.RepositoryPolicy{Enabled: true},
		Releases:  maven.RepositoryPolicy{Enabled: true},
	})

	assert.Contains(t, ctx.Project.PluginRepositories, maven.Repository{
		ID:        "apache.snapshots",
		URL:       "https://repository.apache.org/content/repositories/snapshots",
		Snapshots: maven.RepositoryPolicy{Enabled: true},
		Releases:  maven.RepositoryPolicy{Enabled: false},
	})

	assert.Contains(t, ctx.Project.PluginRepositories, maven.Repository{
		ID:        "sonatype.snapshots",
		URL:       "https://oss.sonatype.org/content/repositories/snapshots/",
		Snapshots: maven.RepositoryPolicy{Enabled: true},
		Releases:  maven.RepositoryPolicy{Enabled: true},
	})
}

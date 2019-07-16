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
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

func TestNewProject(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	ctx := Context{
		Catalog: catalog,
		Build: v1alpha1.BuildSpec{
			RuntimeVersion: defaults.RuntimeVersion,
			Platform: v1alpha1.IntegrationPlatformSpec{
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					CamelVersion: catalog.Version,
				},
			},
			Dependencies: []string{
				"camel-k:runtime-main",
				"bom:my.company/my-artifact-1/1.0.0",
				"bom:my.company/my-artifact-2/2.0.0",
			},
		},
	}

	err = generateProject(&ctx)
	assert.Nil(t, err)
	assert.ElementsMatch(
		t,
		ctx.Maven.Project.DependencyManagement.Dependencies,
		[]maven.Dependency{
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "camel-bom",
				Version:    ctx.Catalog.Version,
				Type:       "pom",
				Scope:      "import",
			},
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-runtime-bom",
				Version:    defaults.RuntimeVersion,
				Type:       "pom",
				Scope:      "import",
			},
			{
				GroupID:    "my.company",
				ArtifactID: "my-artifact-1",
				Version:    "1.0.0",
				Type:       "pom",
				Scope:      "import",
			},
			{
				GroupID:    "my.company",
				ArtifactID: "my-artifact-2",
				Version:    "2.0.0",
				Type:       "pom",
				Scope:      "import",
			},
		},
	)
}

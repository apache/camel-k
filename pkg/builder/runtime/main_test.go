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

package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/maven"
)

func TestNewProject(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	ctx := builder.Context{
		Catalog: catalog,
		Build: v1.BuilderTask{
			CamelVersion:   catalog.Version,
			RuntimeVersion: catalog.RuntimeVersion,
			Dependencies: []string{
				"camel-k:runtime-main",
				"bom:my.company/my-artifact-1/1.0.0",
				"bom:my.company/my-artifact-2/2.0.0",
			},
		},
	}

	err = Steps.GenerateProject.Execute(&ctx)
	assert.Nil(t, err)
	err = builder.Steps.InjectDependencies.Execute(&ctx)
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
				Version:    catalog.RuntimeVersion,
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

func TestGenerateJvmProject(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	ctx := builder.Context{
		Catalog: catalog,
		Build: v1.BuilderTask{
			CamelVersion:   catalog.Version,
			RuntimeVersion: catalog.RuntimeVersion,
			Dependencies: []string{
				"camel-k:runtime-main",
			},
		},
	}

	err = Steps.GenerateProject.Execute(&ctx)
	assert.Nil(t, err)
	err = builder.Steps.InjectDependencies.Execute(&ctx)
	assert.Nil(t, err)

	assert.ElementsMatch(
		t,
		ctx.Maven.Project.DependencyManagement.Dependencies,
		[]maven.Dependency{
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "camel-bom",
				Version:    catalog.Version,
				Type:       "pom",
				Scope:      "import",
			},
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-runtime-bom",
				Version:    catalog.RuntimeVersion,
				Type:       "pom",
				Scope:      "import",
			},
		},
	)

	assert.ElementsMatch(
		t,
		ctx.Maven.Project.Dependencies,
		[]maven.Dependency{
			{GroupID: "org.apache.camel.k", ArtifactID: "camel-k-runtime-main"},
			{GroupID: "org.apache.camel", ArtifactID: "camel-core-engine"},
			{GroupID: "org.apache.camel", ArtifactID: "camel-main"},
		},
	)
}

func TestGenerateGroovyProject(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	ctx := builder.Context{
		Catalog: catalog,
		Build: v1.BuilderTask{
			CamelVersion:   catalog.Version,
			RuntimeVersion: catalog.RuntimeVersion,
			Dependencies: []string{
				"camel-k:runtime-main",
				"camel-k:loader-groovy",
			},
		},
	}

	err = Steps.GenerateProject.Execute(&ctx)
	assert.Nil(t, err)
	err = builder.Steps.InjectDependencies.Execute(&ctx)
	assert.Nil(t, err)

	assert.ElementsMatch(
		t,
		ctx.Maven.Project.DependencyManagement.Dependencies,
		[]maven.Dependency{
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "camel-bom",
				Version:    catalog.Version,
				Type:       "pom",
				Scope:      "import",
			},
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-runtime-bom",
				Version:    catalog.RuntimeVersion,
				Type:       "pom",
				Scope:      "import",
			},
		},
	)

	assert.ElementsMatch(
		t,
		ctx.Maven.Project.Dependencies,
		[]maven.Dependency{
			{GroupID: "org.apache.camel.k", ArtifactID: "camel-k-runtime-main"},
			{GroupID: "org.apache.camel.k", ArtifactID: "camel-k-loader-groovy"},
			{GroupID: "org.apache.camel", ArtifactID: "camel-core-engine"},
			{GroupID: "org.apache.camel", ArtifactID: "camel-main"},
			{GroupID: "org.apache.camel", ArtifactID: "camel-groovy"},
			{GroupID: "org.apache.camel", ArtifactID: "camel-endpointdsl"},
		},
	)
}

func TestSanitizeDependencies(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	ctx := builder.Context{
		Catalog: catalog,
		Build: v1.BuilderTask{
			CamelVersion:   catalog.Version,
			RuntimeVersion: catalog.RuntimeVersion,
			Dependencies: []string{
				"camel:undertow",
				"mvn:org.apache.camel/camel-core/2.18.0",
				"mvn:org.apache.camel.k/camel-k-runtime-main/1.0.0",
				"mvn:com.mycompany/my-dep/1.2.3",
			},
		},
	}

	err = Steps.GenerateProject.Execute(&ctx)
	assert.Nil(t, err)
	err = builder.Steps.InjectDependencies.Execute(&ctx)
	assert.Nil(t, err)
	err = builder.Steps.SanitizeDependencies.Execute(&ctx)
	assert.Nil(t, err)

	assert.Contains(t, ctx.Maven.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel.k",
		ArtifactID: "camel-k-runtime-main",
	})
	assert.Contains(t, ctx.Maven.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel",
		ArtifactID: "camel-core",
	})
	assert.Contains(t, ctx.Maven.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel",
		ArtifactID: "camel-undertow",
	})
	assert.Contains(t, ctx.Maven.Project.Dependencies, maven.Dependency{
		GroupID:    "com.mycompany",
		ArtifactID: "my-dep",
		Version:    "1.2.3",
	})
}

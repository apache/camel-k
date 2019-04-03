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
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/maven"
)

// ArtifactIDs --
func ArtifactIDs(artifacts []v1alpha1.Artifact) []string {
	result := make([]string, 0, len(artifacts))

	for _, a := range artifacts {
		result = append(result, a.ID)
	}

	return result
}

// NewProject --
func NewProject(ctx *Context) (maven.Project, error) {
	//
	// Catalog
	//
	if ctx.Catalog == nil {
		c, err := camel.Catalog(ctx.Request.C, ctx.Client, ctx.Namespace, ctx.Request.Platform.Build.CamelVersion)
		if err != nil {
			return maven.Project{}, err
		}

		ctx.Catalog = c
	}

	p := maven.Project{
		XMLName:              xml.Name{Local: "project"},
		XMLNs:                "http://maven.apache.org/POM/4.0.0",
		XMLNsXsi:             "http://www.w3.org/2001/XMLSchema-instance",
		XsiSchemaLocation:    "http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd",
		ModelVersion:         "4.0.0",
		GroupID:              "org.apache.camel.k.integration",
		ArtifactID:           "camel-k-integration",
		Version:              defaults.Version,
		Properties:           ctx.Request.Platform.Build.Properties,
		DependencyManagement: maven.DependencyManagement{Dependencies: make([]maven.Dependency, 0)},
		Dependencies:         make([]maven.Dependency, 0),
	}

	//
	// DependencyManagement
	//

	p.DependencyManagement.Dependencies = append(p.DependencyManagement.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel",
		ArtifactID: "camel-bom",
		Version:    ctx.Catalog.Version,
		Type:       "pom",
		Scope:      "import",
	})

	for _, d := range ctx.Request.Dependencies {
		if strings.HasPrefix(d, "bom:") {
			mid := strings.TrimPrefix(d, "bom:")
			gav := strings.Replace(mid, "/", ":", -1)

			d, err := maven.ParseGAV(gav)
			if err != nil {
				return maven.Project{}, err
			}

			p.DependencyManagement.Dependencies = append(p.DependencyManagement.Dependencies, maven.Dependency{
				GroupID:    d.GroupID,
				ArtifactID: d.ArtifactID,
				Version:    d.Version,
				Type:       "pom",
				Scope:      "import",
			})
		}
	}

	//p.DependencyManagement.Dependencies = dm

	//
	// Repositories
	//

	p.Repositories = make([]maven.Repository, 0, len(ctx.Request.Repositories))
	p.PluginRepositories = make([]maven.Repository, 0, len(ctx.Request.Repositories))

	for i, r := range ctx.Request.Repositories {
		repo := maven.NewRepository(r)
		if repo.ID == "" {
			repo.ID = fmt.Sprintf("repo-%03d", i)
		}

		p.Repositories = append(p.Repositories, repo)
		p.PluginRepositories = append(p.PluginRepositories, repo)
	}

	return p, nil
}

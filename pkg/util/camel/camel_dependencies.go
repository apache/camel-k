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
	"fmt"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/jitpack"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/rs/xid"
)

// ManageIntegrationDependencies --
func ManageIntegrationDependencies(
	project *maven.Project,
	dependencies []string,
	catalog *RuntimeCatalog) error {

	// Add dependencies from build
	for _, d := range dependencies {
		switch {
		case strings.HasPrefix(d, "bom:"):
			gav := strings.TrimPrefix(d, "bom:")

			d, err := maven.ParseGAV(gav)
			if err != nil {
				return err
			}

			project.DependencyManagement.Dependencies = append(project.DependencyManagement.Dependencies, maven.Dependency{
				GroupID:    d.GroupID,
				ArtifactID: d.ArtifactID,
				Version:    d.Version,
				Type:       "pom",
				Scope:      "import",
			})
		case strings.HasPrefix(d, "camel:"):
			if catalog != nil && catalog.Runtime.Provider == v1.RuntimeProviderQuarkus {
				artifactID := strings.TrimPrefix(d, "camel:")

				if !strings.HasPrefix(artifactID, "camel-") {
					artifactID = "camel-quarkus-" + artifactID
				}

				project.AddDependencyGAV("org.apache.camel.quarkus", artifactID, "")
			} else {
				artifactID := strings.TrimPrefix(d, "camel:")

				if !strings.HasPrefix(artifactID, "camel-") {
					artifactID = "camel-" + artifactID
				}

				project.AddDependencyGAV("org.apache.camel", artifactID, "")
			}
		case strings.HasPrefix(d, "camel-k:"):
			artifactID := strings.TrimPrefix(d, "camel-k:")

			if !strings.HasPrefix(artifactID, "camel-k-") {
				artifactID = "camel-k-" + artifactID
			}

			project.AddDependencyGAV("org.apache.camel.k", artifactID, "")
		case strings.HasPrefix(d, "camel-quarkus:"):
			artifactID := strings.TrimPrefix(d, "camel-quarkus:")

			if !strings.HasPrefix(artifactID, "camel-quarkus-") {
				artifactID = "camel-quarkus-" + artifactID
			}

			project.AddDependencyGAV("org.apache.camel.quarkus", artifactID, "")
		case strings.HasPrefix(d, "mvn:"):
			gav := strings.TrimPrefix(d, "mvn:")

			project.AddEncodedDependencyGAV(gav)
			// TODO hack for tools.jar dependency issue in jolokia-jvm
			// this block should be removed once the jolokia-jvm pom issue
			// is resolved
			// https://github.com/rhuss/jolokia/issues/473
			if strings.Contains(gav, "org.jolokia:jolokia-jvm") {
				me := maven.Exclusion{
					GroupID:    "com.sun",
					ArtifactID: "tools",
				}
				project.AddEncodedDependencyExclusion(gav, me)
			}
		default:
			if dep := jitpack.ToDependency(d); dep != nil {
				project.AddDependency(*dep)

				addRepo := true
				for _, repo := range project.Repositories {
					if repo.URL == jitpack.RepoURL {
						addRepo = false
						break
					}
				}
				if addRepo {
					project.Repositories = append(project.Repositories, v1.Repository{
						ID:  "jitpack.io-" + xid.New().String(),
						URL: jitpack.RepoURL,
						Releases: v1.RepositoryPolicy{
							Enabled:        true,
							ChecksumPolicy: "fail",
						},
						Snapshots: v1.RepositoryPolicy{
							Enabled:        true,
							ChecksumPolicy: "fail",
						},
					})
				}
			} else {
				return fmt.Errorf("unknown dependency type: %s", d)
			}
		}
	}

	// Add dependencies from catalog
	deps := make([]maven.Dependency, len(project.Dependencies))
	copy(deps, project.Dependencies)

	for _, d := range deps {
		if a, ok := catalog.Artifacts[d.ArtifactID]; ok {
			for _, dep := range a.Dependencies {
				md := maven.Dependency{
					GroupID:    dep.GroupID,
					ArtifactID: dep.ArtifactID,
				}

				project.AddDependency(md)

				for _, e := range dep.Exclusions {
					me := maven.Exclusion{
						GroupID:    e.GroupID,
						ArtifactID: e.ArtifactID,
					}

					project.AddDependencyExclusion(md, me)
				}
			}
		}
	}

	// Post process dependencies
	deps = make([]maven.Dependency, len(project.Dependencies))
	copy(deps, project.Dependencies)

	for _, d := range deps {
		if a, ok := catalog.Artifacts[d.ArtifactID]; ok {
			md := maven.Dependency{
				GroupID:    a.GroupID,
				ArtifactID: a.ArtifactID,
			}

			for _, e := range a.Exclusions {
				me := maven.Exclusion{
					GroupID:    e.GroupID,
					ArtifactID: e.ArtifactID,
				}

				project.AddDependencyExclusion(md, me)
			}
		}
	}

	return nil
}

// SanitizeIntegrationDependencies --
func SanitizeIntegrationDependencies(dependencies []maven.Dependency) error {
	for i := 0; i < len(dependencies); i++ {
		dep := dependencies[i]

		// It may be externalized into runtime provider specific steps
		switch dep.GroupID {
		case "org.apache.camel":
			fallthrough
		case "org.apache.camel.k":
			fallthrough
		case "org.apache.camel.quarkus":
			//
			// Remove the version so we force using the one configured by the bom
			//
			dependencies[i].Version = ""
		}
	}

	return nil
}

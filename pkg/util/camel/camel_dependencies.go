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
	"io"
	"path/filepath"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/jitpack"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/rs/xid"
)

// NormalizeDependency converts different forms of camel dependencies
// -- `camel-xxx`, `camel-quarkus-xxx`, and `camel-quarkus:xxx` --
// into the unified form `camel:xxx`.
func NormalizeDependency(dependency string) string {
	newDep := dependency
	switch {
	case strings.HasPrefix(newDep, "camel-quarkus-"):
		newDep = "camel:" + strings.TrimPrefix(dependency, "camel-quarkus-")
	case strings.HasPrefix(newDep, "camel-quarkus:"):
		newDep = "camel:" + strings.TrimPrefix(dependency, "camel-quarkus:")
	case strings.HasPrefix(newDep, "camel-k-"):
		newDep = "camel-k:" + strings.TrimPrefix(dependency, "camel-k-")
	case strings.HasPrefix(newDep, "camel-k:"):
		// do nothing
	case strings.HasPrefix(newDep, "camel-"):
		newDep = "camel:" + strings.TrimPrefix(dependency, "camel-")
	}
	return newDep
}

type Output interface {
	OutOrStdout() io.Writer
	ErrOrStderr() io.Writer
}

// ValidateDependencies validates dependencies against Camel catalog.
// It only shows warning and does not throw error in case the Catalog is just not complete
// and we don't want to let it stop the process.
func ValidateDependencies(catalog *RuntimeCatalog, dependencies []string, out Output) {
	for _, d := range dependencies {
		ValidateDependency(catalog, d, out)
	}
}

// ValidateDependency validates a dependency against Camel catalog.
// It only shows warning and does not throw error in case the Catalog is just not complete
// and we don't want to let it stop the process.
func ValidateDependency(catalog *RuntimeCatalog, dependency string, out Output) {
	switch {
	case strings.HasPrefix(dependency, "camel:"):
		artifact := strings.TrimPrefix(dependency, "camel:")
		if ok := catalog.IsValidArtifact(artifact); !ok {
			fmt.Fprintf(out.ErrOrStderr(), "Warning: dependency %s not found in Camel catalog\n", dependency)
		}
	case strings.HasPrefix(dependency, "mvn:org.apache.camel:"):
		component := strings.Split(dependency, ":")[2]
		fmt.Fprintf(out.ErrOrStderr(), "Warning: do not use %s. Use %s instead\n",
			dependency, NormalizeDependency(component))
	case strings.HasPrefix(dependency, "mvn:org.apache.camel.quarkus:"):
		component := strings.Split(dependency, ":")[2]
		fmt.Fprintf(out.ErrOrStderr(), "Warning: do not use %s. Use %s instead\n",
			dependency, NormalizeDependency(component))
	}

}

// ValidateDependenciesE validates dependencies against Camel catalog and throws error
// if it doesn't exist in the catalog.
func ValidateDependenciesE(catalog *RuntimeCatalog, dependencies []string) error {
	for _, dependency := range dependencies {
		if !strings.HasPrefix(dependency, "camel:") {
			continue
		}

		artifact := strings.TrimPrefix(dependency, "camel:")
		if ok := catalog.IsValidArtifact(artifact); !ok {
			return fmt.Errorf("dependency %s not found in Camel catalog", dependency)
		}
	}

	return nil
}

// ManageIntegrationDependencies sets up all the required dependencies for the given Maven project.
func ManageIntegrationDependencies(project *maven.Project, dependencies []string, catalog *RuntimeCatalog) error {
	// Add dependencies from build
	if err := addDependencies(project, dependencies, catalog); err != nil {
		return err
	}
	// Add dependencies from catalog
	addDependenciesFromCatalog(project, catalog)
	// Post process dependencies
	postProcessDependencies(project, catalog)

	return nil
}

func addDependencies(project *maven.Project, dependencies []string, catalog *RuntimeCatalog) error {
	for _, d := range dependencies {
		switch {
		case strings.HasPrefix(d, "bom:"):
			if err := addBOM(project, d); err != nil {
				return err
			}
		case strings.HasPrefix(d, "camel:"):
			addCamelComponent(project, catalog, d)
		case strings.HasPrefix(d, "camel-k:"):
			addCamelKComponent(project, d)
		case strings.HasPrefix(d, "camel-quarkus:"):
			addCamelQuarkusComponent(project, d)
		case strings.HasPrefix(d, "mvn:"):
			addMavenDependency(project, d)
		case strings.HasPrefix(d, "registry-mvn:"):
			if err := addRegistryMavenDependency(project, d); err != nil {
				return err
			}
		default:
			if err := addJitPack(project, d); err != nil {
				return err
			}
		}
	}

	return nil
}

func addBOM(project *maven.Project, dependency string) error {
	gav := strings.TrimPrefix(dependency, "bom:")
	d, err := maven.ParseGAV(gav)
	if err != nil {
		return err
	}
	project.DependencyManagement.Dependencies = append(project.DependencyManagement.Dependencies,
		maven.Dependency{
			GroupID:    d.GroupID,
			ArtifactID: d.ArtifactID,
			Version:    d.Version,
			Type:       "pom",
			Scope:      "import",
		})

	return nil
}

func addCamelComponent(project *maven.Project, catalog *RuntimeCatalog, dependency string) {
	artifactID := strings.TrimPrefix(dependency, "camel:")
	if catalog != nil && catalog.Runtime.Provider == v1.RuntimeProviderQuarkus {
		if !strings.HasPrefix(artifactID, "camel-") {
			artifactID = "camel-quarkus-" + artifactID
		}
		project.AddDependencyGAV("org.apache.camel.quarkus", artifactID, "")
	} else {
		if !strings.HasPrefix(artifactID, "camel-") {
			artifactID = "camel-" + artifactID
		}
		project.AddDependencyGAV("org.apache.camel", artifactID, "")
	}
}

func addCamelKComponent(project *maven.Project, dependency string) {
	artifactID := strings.TrimPrefix(dependency, "camel-k:")
	if !strings.HasPrefix(artifactID, "camel-k-") {
		artifactID = "camel-k-" + artifactID
	}
	project.AddDependencyGAV("org.apache.camel.k", artifactID, "")
}

func addCamelQuarkusComponent(project *maven.Project, dependency string) {
	artifactID := strings.TrimPrefix(dependency, "camel-quarkus:")
	if !strings.HasPrefix(artifactID, "camel-quarkus-") {
		artifactID = "camel-quarkus-" + artifactID
	}
	project.AddDependencyGAV("org.apache.camel.quarkus", artifactID, "")
}

func addMavenDependency(project *maven.Project, dependency string) {
	gav := strings.TrimPrefix(dependency, "mvn:")
	project.AddEncodedDependencyGAV(gav)
}

func addRegistryMavenDependency(project *maven.Project, dependency string) error {
	mapping := strings.Split(dependency, "@")
	outputFileRelativePath := mapping[1]
	gavString := strings.TrimPrefix(mapping[0], "registry-mvn:")
	gav, err := maven.ParseGAV(gavString)
	if err != nil {
		return err
	}
	plugin := getOrCreateBuildPlugin(project, "com.googlecode.maven-download-plugin", "download-maven-plugin", "1.6.8")
	outputDirectory := "../context"
	isClasspath := len(mapping) == 3 && mapping[2] == "classpath"
	if isClasspath {
		outputDirectory = "src/main/resources"
	}

	properties := v1.PluginProperties{}
	properties.Add("outputDirectory", filepath.Join(outputDirectory, filepath.Dir(outputFileRelativePath)))
	properties.Add("outputFileName", filepath.Base(outputFileRelativePath))
	properties.Add("groupId", gav.GroupID)
	properties.Add("artifactId", gav.ArtifactID)
	properties.Add("version", gav.Version)
	properties.Add("type", gav.Type)

	exec := maven.Execution{
		ID:    fmt.Sprint(len(plugin.Executions)),
		Phase: "validate",
		Goals: []string{
			"artifact",
		},
		Configuration: properties,
	}
	plugin.Executions = append(plugin.Executions, exec)

	return nil
}

func getOrCreateBuildPlugin(project *maven.Project, groupID string, artifactID string, version string) *maven.Plugin {
	for i, plugin := range project.Build.Plugins {
		if plugin.GroupID == groupID && plugin.ArtifactID == artifactID && plugin.Version == version {
			return &project.Build.Plugins[i]
		}
	}
	plugin := maven.Plugin{
		GroupID:    groupID,
		ArtifactID: artifactID,
		Version:    version,
		Executions: []maven.Execution{},
	}
	project.Build.Plugins = append(project.Build.Plugins, plugin)

	return &project.Build.Plugins[len(project.Build.Plugins)-1]
}

func addJitPack(project *maven.Project, dependency string) error {
	dep := jitpack.ToDependency(dependency)
	if dep == nil {
		return fmt.Errorf("unknown dependency type: %s", dependency)
	}

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

	return nil
}

func addDependenciesFromCatalog(project *maven.Project, catalog *RuntimeCatalog) {
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
}

func postProcessDependencies(project *maven.Project, catalog *RuntimeCatalog) {
	deps := make([]maven.Dependency, len(project.Dependencies))
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
}

// SanitizeIntegrationDependencies --.
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

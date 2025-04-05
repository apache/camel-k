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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/jib"

	"github.com/apache/camel-k/v2/pkg/util/io"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/digest"
	"github.com/apache/camel-k/v2/pkg/util/maven"
)

const projectModePerm = 0600

func init() {
	registerSteps(Quarkus)

	Quarkus.CommonSteps = []Step{
		Quarkus.LoadCamelQuarkusCatalog,
		Quarkus.GenerateQuarkusProject,
		Quarkus.BuildQuarkusMavenContext,
		Quarkus.BuildQuarkusMavenProject,
	}
}

type quarkusSteps struct {
	LoadCamelQuarkusCatalog    Step
	GenerateQuarkusProject     Step
	BuildQuarkusMavenContext   Step
	BuildQuarkusMavenProject   Step
	ComputeQuarkusDependencies Step
	PrepareProjectWithSources  Step

	CommonSteps []Step
}

//nolint:mnd
var Quarkus = quarkusSteps{
	LoadCamelQuarkusCatalog:    NewStep(InitPhase, loadCamelQuarkusCatalog),
	GenerateQuarkusProject:     NewStep(ProjectGenerationPhase, generateQuarkusProject),
	BuildQuarkusMavenContext:   NewStep(ProjectGenerationPhase+1, buildMavenContextSettings),
	PrepareProjectWithSources:  NewStep(ProjectBuildPhase-1, prepareProjectWithSources),
	BuildQuarkusMavenProject:   NewStep(ProjectBuildPhase+2, buildMavenProject),
	ComputeQuarkusDependencies: NewStep(ProjectBuildPhase+1, computeQuarkusDependencies),
}

func prepareProjectWithSources(ctx *builderContext) error {
	sourcesPath := filepath.Join(ctx.Path, "maven", "src", "main", "resources", "routes")
	if err := os.MkdirAll(sourcesPath, os.ModePerm); err != nil {
		return fmt.Errorf("failure while creating resource folder: %w", err)
	}

	sourceList := ""
	for _, source := range ctx.Build.Sources {
		if sourceList != "" {
			sourceList += ","
		}
		sourceList += "classpath:routes/" + source.Name
		if err := os.WriteFile(
			filepath.Join(sourcesPath, source.Name),
			[]byte(source.Content),
			projectModePerm,
		); err != nil {
			return fmt.Errorf("failure while writing %s: %w", source.Name, err)
		}
	}

	if sourceList != "" {
		routesIncludedPattern := "camel.main.routes-include-pattern = " + sourceList
		if err := os.WriteFile(
			filepath.Join(filepath.Dir(sourcesPath), "application.properties"),
			[]byte(routesIncludedPattern),
			projectModePerm,
		); err != nil {
			return fmt.Errorf("failure while writing the configuration application.properties: %w", err)
		}
	}
	return nil
}

func loadCamelQuarkusCatalog(ctx *builderContext) error {
	runtime := ctx.Build.Runtime.DeepCopy()
	if runtime.Provider == v1.RuntimeProviderPlainQuarkus {
		// We need this workaround to load the last existing catalog
		// TODO: this part will be subject to future refactoring
		runtime.Version = defaults.DefaultRuntimeVersion
	}

	catalog, err := camel.LoadCatalog(ctx.C, ctx.Client, ctx.Namespace, *runtime)
	if err != nil {
		return err
	}

	if catalog == nil {
		return fmt.Errorf("unable to find catalog matching version requirement: runtime=%s, provider=%s",
			runtime.Version,
			runtime.Provider)
	}

	ctx.Catalog = catalog

	return nil
}

func generateQuarkusProject(ctx *builderContext) error {
	p := generateQuarkusProjectCommon(
		ctx.Build.Runtime.Provider,
		ctx.Build.Runtime.Version,
		ctx.Build.Runtime.Metadata["quarkus.version"],
	)
	// Add Maven build extensions
	p.Build.Extensions = &ctx.Build.Maven.Extension
	// Add Maven repositories
	p.Repositories = append(p.Repositories, ctx.Build.Maven.Repositories...)
	p.PluginRepositories = append(p.PluginRepositories, ctx.Build.Maven.Repositories...)
	ctx.Maven.Project = p

	return nil
}

func generateQuarkusProjectCommon(runtimeProvider v1.RuntimeProvider, runtimeVersion string,
	quarkusPlatformVersion string) maven.Project {
	if runtimeProvider == v1.RuntimeProviderPlainQuarkus {
		// We need this workaround to load the last existing catalog
		// TODO: this part will be subject to future refactoring
		quarkusPlatformVersion = runtimeVersion
	}
	p := maven.NewProjectWithGAV("org.apache.camel.k.integration", "camel-k-integration", defaults.Version)
	p.DependencyManagement = &maven.DependencyManagement{Dependencies: make([]maven.Dependency, 0)}
	p.Dependencies = make([]maven.Dependency, 0)
	p.Build = &maven.Build{Plugins: make([]maven.Plugin, 0)}

	// set fast-jar packaging by default, since it gives some startup time improvements
	p.Properties.Add("quarkus.package.jar.type", "fast-jar")
	// Reproducible builds: https://maven.apache.org/guides/mini/guide-reproducible-builds.html
	p.Properties.Add("project.build.outputTimestamp", time.Now().Format(time.RFC3339))
	// DependencyManagement
	if runtimeProvider == v1.RuntimeProviderPlainQuarkus {
		p.DependencyManagement.Dependencies = append(p.DependencyManagement.Dependencies,
			maven.Dependency{
				GroupID:    "io.quarkus.platform",
				ArtifactID: "quarkus-camel-bom",
				Version:    runtimeVersion,
				Type:       "pom",
				Scope:      "import",
			},
			maven.Dependency{
				GroupID:    "io.quarkus.platform",
				ArtifactID: "quarkus-bom",
				Version:    runtimeVersion,
				Type:       "pom",
				Scope:      "import",
			},
		)
	} else {
		// Camel K Runtime (Quarkus based) default
		p.DependencyManagement.Dependencies = append(p.DependencyManagement.Dependencies,
			maven.Dependency{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-runtime-bom",
				Version:    runtimeVersion,
				Type:       "pom",
				Scope:      "import",
			},
		)
	}

	// Plugins
	p.Build.Plugins = append(p.Build.Plugins,
		maven.Plugin{
			GroupID:    "io.quarkus",
			ArtifactID: "quarkus-maven-plugin",
			Version:    quarkusPlatformVersion,
			Executions: []maven.Execution{
				{
					ID: "build-integration",
					Goals: []string{
						"build",
					},
				},
			},
		},
	)

	// Jib publish profile
	p.AddProfile(jib.JibMavenProfile(jib.JibMavenPluginVersionDefault, jib.JibLayerFilterExtensionMavenVersionDefault))

	return p
}

func buildMavenProject(ctx *builderContext) error {
	mc := newMavenContext(ctx)

	return BuildQuarkusRunnerCommon(ctx.C, *mc, ctx.Maven.Project, ctx.Build.Maven.Properties)
}

func BuildQuarkusRunnerCommon(ctx context.Context, mc maven.Context, project maven.Project, applicationProperties map[string]string) error {
	resourcesPath := filepath.Join(mc.Path, "src", "main", "resources")
	if err := os.MkdirAll(resourcesPath, os.ModePerm); err != nil {
		return fmt.Errorf("failure while creating resource folder: %w", err)
	}
	if err := computeApplicationProperties(filepath.Join(resourcesPath, "application.properties"), applicationProperties); err != nil {
		return err
	}
	if err := project.Command(mc).DoPom(ctx); err != nil {
		return fmt.Errorf("failure while generating pom file: %w", err)
	}
	mc.AddArgument("package")
	mc.AddArgument("-Dmaven.test.skip=true")
	if err := project.Command(mc).Do(ctx); err != nil {
		return fmt.Errorf("failure while building project: %w", err)
	}

	return nil
}

func computeApplicationProperties(appPropertiesPath string, applicationProperties map[string]string) error {
	f, err := os.OpenFile(appPropertiesPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, io.FilePerm644)
	if err != nil {
		return fmt.Errorf("failure while opening/creating application.properties: %w", err)
	}
	fstat, err := f.Stat()
	if err != nil {
		return err
	}
	if applicationProperties == nil {
		// Default build time properties
		applicationProperties = make(map[string]string)
	}
	// disable quarkus banner
	applicationProperties["quarkus.banner.enabled"] = boolean.FalseString
	// camel-quarkus does route discovery at startup, but we don't want
	// this to happen as routes are loaded at runtime and looking for
	// routes at build time may try to load camel-k-runtime routes builder
	// proxies which in some case may fail.
	applicationProperties["quarkus.camel.routes-discovery.enabled"] = boolean.FalseString
	// required for to resolve data type transformers at runtime with service discovery
	// the different Camel runtimes use different resource paths for the service lookup
	applicationProperties["quarkus.camel.service.discovery.include-patterns"] = "META-INF/services/org/apache/camel/datatype/converter/*,META-INF/services/org/apache/camel/datatype/transformer/*,META-INF/services/org/apache/camel/transformer/*"
	// Workaround to prevent JS runtime errors, see https://github.com/apache/camel-quarkus/issues/5678
	applicationProperties["quarkus.class-loading.parent-first-artifacts"] = "org.graalvm.regex:regex"
	defer f.Close()
	// Add a new line if the file is already containing some value
	if fstat.Size() > 0 {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	// Fill with properties coming from user configuration
	for k, v := range applicationProperties {
		if _, err := f.WriteString(fmt.Sprintf("%s=%s\n", k, v)); err != nil {
			return err
		}
	}
	return nil
}

func computeQuarkusDependencies(ctx *builderContext) error {
	// Quarkus fast-jar format is split into various sub-directories in quarkus-app
	quarkusAppDir := filepath.Join(ctx.Path, "maven", "target", "quarkus-app")
	// Process artifacts list and add it to existing artifacts
	artifacts, err := processQuarkusTransitiveDependencies(quarkusAppDir)
	if err != nil {
		return err
	}
	ctx.Artifacts = append(ctx.Artifacts, artifacts...)

	return nil
}

func processQuarkusTransitiveDependencies(dir string) ([]v1.Artifact, error) {
	var artifacts []v1.Artifact

	// Discover application dependencies from the Quarkus fast-jar directory tree
	err := filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fileRelPath := strings.Replace(filePath, dir, "", 1)

		if !info.IsDir() {
			sha1, err := digest.ComputeSHA1(filePath)
			if err != nil {
				return err
			}

			artifacts = append(artifacts, v1.Artifact{
				ID:       filepath.Base(fileRelPath),
				Location: filePath,
				Target:   filepath.Join(DependenciesDir, fileRelPath),
				Checksum: "sha1:" + sha1,
			})
		}

		return nil
	})

	return artifacts, err
}

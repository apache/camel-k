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
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/maven"
)

func init() {
	registerSteps(Quarkus)

	Quarkus.CommonSteps = []Step{
		Quarkus.LoadCamelQuarkusCatalog,
		Quarkus.GenerateQuarkusProject,
		Quarkus.BuildQuarkusRunner,
	}
}

type quarkusSteps struct {
	LoadCamelQuarkusCatalog    Step
	GenerateQuarkusProject     Step
	BuildQuarkusRunner         Step
	ComputeQuarkusDependencies Step

	CommonSteps []Step
}

var Quarkus = quarkusSteps{
	LoadCamelQuarkusCatalog:    NewStep(InitPhase, loadCamelQuarkusCatalog),
	GenerateQuarkusProject:     NewStep(ProjectGenerationPhase, generateQuarkusProject),
	BuildQuarkusRunner:         NewStep(ProjectBuildPhase, buildQuarkusRunner),
	ComputeQuarkusDependencies: NewStep(ProjectBuildPhase+1, computeQuarkusDependencies),
}

func loadCamelQuarkusCatalog(ctx *builderContext) error {
	catalog, err := camel.LoadCatalog(ctx.C, ctx.Client, ctx.Namespace, ctx.Build.Runtime)
	if err != nil {
		return err
	}

	if catalog == nil {
		return fmt.Errorf("unable to find catalog matching version requirement: runtime=%s, provider=%s",
			ctx.Build.Runtime.Version,
			ctx.Build.Runtime.Provider)
	}

	ctx.Catalog = catalog

	return nil
}

func generateQuarkusProject(ctx *builderContext) error {
	p := GenerateQuarkusProjectCommon(
		ctx.Build.Runtime.Metadata["camel-quarkus.version"],
		ctx.Build.Runtime.Version,
		ctx.Build.Runtime.Metadata["quarkus.version"])

	// Add all the properties from the build configuration
	p.Properties.AddAll(ctx.Build.Maven.Properties)

	// Add Maven build extensions
	p.Build.Extensions = ctx.Build.Maven.Extension

	// Add Maven repositories
	p.Repositories = append(p.Repositories, ctx.Build.Maven.Repositories...)

	ctx.Maven.Project = p

	return nil
}

func GenerateQuarkusProjectCommon(camelQuarkusVersion string, runtimeVersion string, quarkusVersion string) maven.Project {
	p := maven.NewProjectWithGAV("org.apache.camel.k.integration", "camel-k-integration", defaults.Version)
	p.DependencyManagement = &maven.DependencyManagement{Dependencies: make([]maven.Dependency, 0)}
	p.Dependencies = make([]maven.Dependency, 0)
	p.Build = &maven.Build{Plugins: make([]maven.Plugin, 0)}

	// camel-quarkus does route discovery at startup, but we don't want
	// this to happen as routes are loaded at runtime and looking for
	// routes at build time may try to load camel-k-runtime routes builder
	// proxies which in some case may fail.
	p.Properties["quarkus.camel.routes-discovery.enabled"] = "false"

	// disable quarkus banner
	p.Properties["quarkus.banner.enabled"] = "false"

	// set fast-jar packaging by default, since it gives some startup time improvements
	p.Properties["quarkus.package.type"] = "fast-jar"

	// DependencyManagement
	p.DependencyManagement.Dependencies = append(p.DependencyManagement.Dependencies,
		maven.Dependency{
			GroupID:    "org.apache.camel.k",
			ArtifactID: "camel-k-runtime-bom",
			Version:    runtimeVersion,
			Type:       "pom",
			Scope:      "import",
		},
	)

	// Plugins
	p.Build.Plugins = append(p.Build.Plugins,
		maven.Plugin{
			GroupID:    "io.quarkus",
			ArtifactID: "quarkus-maven-plugin",
			Version:    quarkusVersion,
			Executions: []maven.Execution{
				{
					Goals: []string{
						"build",
					},
				},
			},
		},
	)

	return p
}

func buildQuarkusRunner(ctx *builderContext) error {
	mc := maven.NewContext(path.Join(ctx.Path, "maven"))
	mc.GlobalSettings = ctx.Maven.GlobalSettings
	mc.UserSettings = ctx.Maven.UserSettings
	mc.SettingsSecurity = ctx.Maven.SettingsSecurity
	mc.LocalRepository = ctx.Build.Maven.LocalRepository
	mc.AdditionalArguments = ctx.Build.Maven.CLIOptions

	if ctx.Maven.TrustStoreName != "" {
		mc.ExtraMavenOpts = append(mc.ExtraMavenOpts,
			"-Djavax.net.ssl.trustStore="+path.Join(ctx.Path, ctx.Maven.TrustStoreName),
			"-Djavax.net.ssl.trustStorePassword="+ctx.Maven.TrustStorePass,
		)
	}

	err := BuildQuarkusRunnerCommon(ctx.C, mc, ctx.Maven.Project)
	if err != nil {
		return err
	}

	return nil
}

func BuildQuarkusRunnerCommon(ctx context.Context, mc maven.Context, project maven.Project) error {
	resourcesPath := path.Join(mc.Path, "src", "main", "resources")
	if err := os.MkdirAll(resourcesPath, os.ModePerm); err != nil {
		return errors.Wrap(err, "failure while creating resource folder")
	}

	// Generate an empty application.properties so that there will be something in
	// target/classes as if such directory does not exist, the quarkus maven plugin
	// may fail the build.
	// In the future there should be a way to provide build information from secrets,
	// configmap, etc.
	if _, err := os.Create(path.Join(resourcesPath, "application.properties")); err != nil {
		return errors.Wrap(err, "failure while creating application.properties")
	}

	mc.AddArgument("package")

	// Run the Maven goal
	if err := project.Command(mc).Do(ctx); err != nil {
		return errors.Wrap(err, "failure while building project")
	}

	return nil
}

func computeQuarkusDependencies(ctx *builderContext) error {
	mc := maven.NewContext(path.Join(ctx.Path, "maven"))
	mc.GlobalSettings = ctx.Maven.GlobalSettings
	mc.UserSettings = ctx.Maven.UserSettings
	mc.SettingsSecurity = ctx.Maven.SettingsSecurity
	mc.LocalRepository = ctx.Build.Maven.LocalRepository
	mc.AdditionalArguments = ctx.Build.Maven.CLIOptions

	// Process artifacts list and add it to existing artifacts
	artifacts, err := ProcessQuarkusTransitiveDependencies(mc)
	if err != nil {
		return err
	}
	ctx.Artifacts = append(ctx.Artifacts, artifacts...)

	return nil
}

func ProcessQuarkusTransitiveDependencies(mc maven.Context) ([]v1.Artifact, error) {
	var artifacts []v1.Artifact

	// Quarkus fast-jar format is split into various sub-directories in quarkus-app
	quarkusAppDir := path.Join(mc.Path, "target", "quarkus-app")

	// Discover application dependencies from the Quarkus fast-jar directory tree
	err := filepath.Walk(quarkusAppDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fileRelPath := strings.Replace(filePath, quarkusAppDir, "", 1)

		if !info.IsDir() {
			sha1, err := digest.ComputeSHA1(filePath)
			if err != nil {
				return err
			}

			artifacts = append(artifacts, v1.Artifact{
				ID:       filepath.Base(fileRelPath),
				Location: filePath,
				Target:   path.Join(DependenciesDir, fileRelPath),
				Checksum: "sha1:" + sha1,
			})
		}

		return nil
	})

	return artifacts, err
}

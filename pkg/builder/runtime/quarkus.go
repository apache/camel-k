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
	"fmt"
	"io/ioutil"
	"os"
	"path"

	yaml2 "gopkg.in/yaml.v2"

	"github.com/pkg/errors"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/maven"
)

// QuarkusSteps --
var QuarkusSteps = []builder.Step{
	Steps.LoadCamelQuarkusCatalog,
	Steps.GenerateQuarkusProject,
	Steps.ComputeQuarkusDependencies,
}

func loadCamelQuarkusCatalog(ctx *builder.Context) error {
	catalog, err := camel.LoadCatalog(ctx.C, ctx.Client, ctx.Build.Meta.Namespace, ctx.Build.CamelVersion, ctx.Build.RuntimeVersion, ctx.Build.RuntimeProvider.Quarkus)
	if err != nil {
		return err
	}

	if catalog == nil {
		return fmt.Errorf("unable to find catalog matching version requirement: camel=%s, runtime=%s, camel-quarkus=%s, quarkus=%s",
			ctx.Build.CamelVersion, ctx.Build.RuntimeVersion, ctx.Build.RuntimeProvider.Quarkus.CamelQuarkusVersion, ctx.Build.RuntimeProvider.Quarkus.QuarkusVersion)
	}

	ctx.Catalog = catalog

	return nil
}

func generateQuarkusProject(ctx *builder.Context) error {
	p := maven.NewProjectWithGAV("org.apache.camel.k.integration", "camel-k-integration", defaults.Version)
	p.DependencyManagement = &maven.DependencyManagement{Dependencies: make([]maven.Dependency, 0)}
	p.Dependencies = make([]maven.Dependency, 0)
	p.Build = &maven.Build{Plugins: make([]maven.Plugin, 0)}

	p.Properties = make(map[string]string)
	for k, v := range ctx.Build.Properties {
		p.Properties[k] = v
	}

	// camel-quarkus doe routes discovery at startup but we don't want
	// this to happen as routes are loaded at runtime and looking for
	// routes at build time may try to load camel-k-runtime routes builder
	// proxies which in some case may fail
	p.Properties["quarkus.camel.main.routes-discovery.enabled"] = "false"

	// DependencyManagement
	p.DependencyManagement.Dependencies = append(p.DependencyManagement.Dependencies,
		maven.Dependency{
			GroupID:    "org.apache.camel.quarkus",
			ArtifactID: "camel-quarkus-bom",
			Version:    ctx.Build.RuntimeProvider.Quarkus.CamelQuarkusVersion,
			Type:       "pom",
			Scope:      "import",
		},
		maven.Dependency{
			GroupID:    "org.apache.camel.k",
			ArtifactID: "camel-k-runtime-bom",
			Version:    ctx.Build.RuntimeVersion,
			Type:       "pom",
			Scope:      "import",
		},
	)

	// Plugins
	p.Build.Plugins = append(p.Build.Plugins,
		maven.Plugin{
			GroupID:    "io.quarkus",
			ArtifactID: "quarkus-maven-plugin",
			Version:    ctx.Build.RuntimeProvider.Quarkus.QuarkusVersion,
			Executions: []maven.Execution{
				{
					Goals: []string{
						"build",
					},
				},
			},
		},
	)

	ctx.Maven.Project = p

	return nil
}

func computeQuarkusDependencies(ctx *builder.Context) error {
	mc := maven.NewContext(path.Join(ctx.Path, "maven"), ctx.Maven.Project)
	mc.SettingsContent = ctx.Maven.SettingsData
	mc.LocalRepository = ctx.Build.Maven.LocalRepository
	mc.Timeout = ctx.Build.Maven.GetTimeout().Duration

	// Build the project
	mc.AddArgument("package")
	if err := maven.Run(mc); err != nil {
		return errors.Wrap(err, "failure while building project")
	}

	// Retrieve the runtime dependencies
	mc.AdditionalArguments = nil
	mc.AddArgumentf("org.apache.camel.k:camel-k-maven-plugin:%s:generate-dependency-list", ctx.Catalog.RuntimeVersion)
	if err := maven.Run(mc); err != nil {
		return errors.Wrap(err, "failure while determining classpath")
	}

	dependencies := path.Join(mc.Path, "target", "dependencies.yaml")
	content, err := ioutil.ReadFile(dependencies)
	if err != nil {
		return err
	}

	cp := make(map[string][]v1.Artifact)
	err = yaml2.Unmarshal(content, &cp)
	if err != nil {
		return err
	}

	for _, e := range cp["dependencies"] {
		gav, err := maven.ParseGAV(e.ID)
		if err != nil {
			return nil
		}

		fileName := gav.GroupID + "." + gav.ArtifactID + "-" + gav.Version + "." + gav.Type
		location := path.Join(mc.Path, "target", "lib", fileName)
		_, err = os.Stat(location)
		// We check that the dependency actually exists in the lib directory
		if os.IsNotExist(err) {
			continue
		}

		ctx.Artifacts = append(ctx.Artifacts, v1.Artifact{
			ID:       gav.GroupID + ":" + gav.ArtifactID + ":" + gav.Type + ":" + gav.Version,
			Location: location,
			Target:   path.Join("lib", fileName),
		})
	}

	runner := "camel-k-integration-" + defaults.Version + "-runner.jar"
	ctx.Artifacts = append(ctx.Artifacts, v1.Artifact{
		ID:       runner,
		Location: path.Join(mc.Path, "target", runner),
		Target:   runner,
	})

	return nil
}

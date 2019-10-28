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
	"path"

	"github.com/pkg/errors"

	yaml2 "gopkg.in/yaml.v2"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/maven"
)

// MainSteps --
var MainSteps = []builder.Step{
	Steps.LoadCamelCatalog,
	Steps.GenerateProject,
	Steps.ComputeDependencies,
}

func loadCamelCatalog(ctx *builder.Context) error {
	catalog, err := camel.LoadCatalog(ctx.C, ctx.Client, ctx.Build.Meta.Namespace, ctx.Build.CamelVersion, ctx.Build.RuntimeVersion, nil)
	if err != nil {
		return err
	}

	if catalog == nil {
		return fmt.Errorf("unable to find catalog matching version requirement: camel=%s, runtime=%s",
			ctx.Build.CamelVersion, ctx.Build.RuntimeVersion)
	}

	ctx.Catalog = catalog

	return nil
}

func generateProject(ctx *builder.Context) error {
	p := maven.NewProjectWithGAV("org.apache.camel.k.integration", "camel-k-integration", defaults.Version)
	p.Properties = ctx.Build.Platform.Build.Properties
	p.DependencyManagement = &maven.DependencyManagement{Dependencies: make([]maven.Dependency, 0)}
	p.Dependencies = make([]maven.Dependency, 0)

	// DependencyManagement
	p.DependencyManagement.Dependencies = append(p.DependencyManagement.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel",
		ArtifactID: "camel-bom",
		Version:    ctx.Build.CamelVersion,
		Type:       "pom",
		Scope:      "import",
	})
	p.DependencyManagement.Dependencies = append(p.DependencyManagement.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel.k",
		ArtifactID: "camel-k-runtime-bom",
		Version:    ctx.Build.RuntimeVersion,
		Type:       "pom",
		Scope:      "import",
	})

	ctx.Maven.Project = p

	return nil
}

func computeDependencies(ctx *builder.Context) error {
	mc := maven.NewContext(path.Join(ctx.Path, "maven"), ctx.Maven.Project)
	mc.SettingsContent = ctx.Maven.SettingsData
	mc.LocalRepository = ctx.Build.Platform.Build.Maven.LocalRepository
	mc.Timeout = ctx.Build.Platform.Build.Maven.Timeout.Duration
	mc.AddArgumentf("org.apache.camel.k:camel-k-maven-plugin:%s:generate-dependency-list", ctx.Catalog.RuntimeVersion)

	if err := maven.Run(mc); err != nil {
		return errors.Wrap(err, "failure while determining classpath")
	}

	dependencies := path.Join(mc.Path, "target", "dependencies.yaml")
	content, err := ioutil.ReadFile(dependencies)
	if err != nil {
		return err
	}

	cp := make(map[string][]v1alpha1.Artifact)
	err = yaml2.Unmarshal(content, &cp)
	if err != nil {
		return err
	}

	for _, e := range cp["dependencies"] {
		_, fileName := path.Split(e.Location)

		gav, err := maven.ParseGAV(e.ID)
		if err != nil {
			return nil
		}

		ctx.Artifacts = append(ctx.Artifacts, v1alpha1.Artifact{
			ID:       e.ID,
			Location: e.Location,
			Target:   path.Join("dependencies", gav.GroupID+"."+fileName),
		})
	}

	return nil
}

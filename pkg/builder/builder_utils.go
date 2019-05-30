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
	"fmt"
	"strings"

	"github.com/apache/camel-k/pkg/util/kubernetes"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/maven"
)

// StepIDsFor --
func StepIDsFor(steps ...Step) []string {
	IDs := make([]string, 0)
	for _, step := range steps {
		IDs = append(IDs, step.ID())
	}
	return IDs
}

func artifactIDs(artifacts []v1alpha1.Artifact) []string {
	result := make([]string, 0, len(artifacts))

	for _, a := range artifacts {
		result = append(result, a.ID)
	}

	return result
}

// NewMavenProject --
func NewMavenProject(ctx *Context) (maven.Project, error) {
	//
	// Catalog
	//
	if ctx.Catalog == nil {
		c, err := camel.Catalog(ctx.C, ctx.Client, ctx.Namespace, ctx.Build.Platform.Build.CamelVersion)
		if err != nil {
			return maven.Project{}, err
		}

		ctx.Catalog = c
	}

	p := maven.NewProject("org.apache.camel.k.integration", "camel-k-integration", defaults.Version)
	p.Properties = ctx.Build.Platform.Build.Properties
	p.DependencyManagement = maven.DependencyManagement{Dependencies: make([]maven.Dependency, 0)}
	p.Dependencies = make([]maven.Dependency, 0)

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

	for _, d := range ctx.Build.Dependencies {
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
	p.Repositories = make([]maven.Repository, 0, len(ctx.Build.Repositories))
	p.PluginRepositories = make([]maven.Repository, 0, len(ctx.Build.Repositories))

	for i, r := range ctx.Build.Repositories {
		repo := maven.NewRepository(r)
		if repo.ID == "" {
			repo.ID = fmt.Sprintf("repo-%03d", i)
		}

		p.Repositories = append(p.Repositories, repo)
		p.PluginRepositories = append(p.PluginRepositories, repo)
	}

	return p, nil
}

// NewMavenSettings --
func NewMavenSettings(ctx *Context) (maven.Settings, error) {
	settings := maven.NewSettings()
	settings.Proxies = make([]maven.Proxy, 0, len(ctx.Build.Platform.Build.Proxies))

	for i, p := range ctx.Build.Platform.Build.Proxies {
		proxy := maven.Proxy{
			Active:        true,
			ID:            p.ID,
			Protocol:      p.Protocol,
			Host:          p.Host,
			Port:          p.Port,
			NonProxyHosts: p.NonProxyHosts,
			Username:      p.Username,
			Password:      p.Password,
		}

		if p.Active != nil {
			proxy.Active = *p.Active
		}

		if proxy.ID == "" {
			proxy.ID = fmt.Sprintf("proxy-%03d", i)
		}

		if proxy.Protocol == "" {
			proxy.Protocol = "http"
		}

		if p.UsernameFrom != nil {
			val, err := kubernetes.ResolveValueSource(ctx.C, ctx.Client, ctx.Namespace, p.UsernameFrom)
			if err != nil {
				return maven.Settings{}, err
			}

			proxy.Username = val
		}

		if p.PasswordFrom != nil {
			val, err := kubernetes.ResolveValueSource(ctx.C, ctx.Client, ctx.Namespace, p.PasswordFrom)
			if err != nil {
				return maven.Settings{}, err
			}

			proxy.Password = val
		}

		settings.Proxies = append(settings.Proxies, proxy)
	}

	return settings, nil
}

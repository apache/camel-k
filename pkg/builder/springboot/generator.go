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

package springboot

import (
	"fmt"
	"strings"

	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/maven"
)

// GenerateProject --
func GenerateProject(ctx *builder.Context) error {
	p, err := builder.NewProject(ctx)
	if err != nil {
		return err
	}

	ctx.Project = p

	//
	// set-up dependencies
	//

	ctx.Project.AddDependency(maven.Dependency{
		GroupID:    "org.apache.camel.k",
		ArtifactID: "camel-k-runtime-spring-boot",
		Version:    ctx.Request.RuntimeVersion,
		Exclusions: &[]maven.Exclusion{
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "*",
			},
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "*",
			},
			{
				GroupID:    "org.springframework.boot",
				ArtifactID: "*",
			},
		},
	})

	//
	// others
	//

	for _, d := range ctx.Request.Dependencies {
		switch {
		case strings.HasPrefix(d, "camel:"):
			if d == "camel:core" {
				continue
			}

			artifactID := strings.TrimPrefix(d, "camel:")

			if !strings.HasPrefix(artifactID, "camel-") {
				artifactID = "camel-" + artifactID
			}

			ctx.Project.AddDependency(maven.Dependency{
				GroupID:    "org.apache.camel",
				ArtifactID: artifactID + "-starter",
				Version:    ctx.Catalog.Version,
				Exclusions: &[]maven.Exclusion{
					{
						GroupID:    "com.sun.xml.bind",
						ArtifactID: "*",
					},
					{
						GroupID:    "org.apache.camel",
						ArtifactID: "camel-core",
					},
					{
						GroupID:    "org.apache.camel",
						ArtifactID: "camel-core-starter",
					},
					{
						GroupID:    "org.apache.camel",
						ArtifactID: "camel-spring-boot-starter",
					},
					{
						GroupID:    "org.springframework.boot",
						ArtifactID: "spring-boot-starter",
					},
				},
			})
		case strings.HasPrefix(d, "camel-k:"):
			artifactID := strings.TrimPrefix(d, "camel-k:")

			if !strings.HasPrefix(artifactID, "camel-") {
				artifactID = "camel-" + artifactID
			}

			ctx.Project.AddDependencyGAV("org.apache.camel.k", artifactID, ctx.Request.RuntimeVersion)
		case strings.HasPrefix(d, "mvn:"):
			mid := strings.TrimPrefix(d, "mvn:")
			gav := strings.Replace(mid, "/", ":", -1)

			ctx.Project.AddEncodedDependencyGAV(gav)
		case strings.HasPrefix(d, "runtime:"):
			if d == "runtime:jvm" {
				// common
				continue
			}
			if d == "runtime:spring-boot" {
				// common
				continue
			}

			artifactID := strings.Replace(d, "runtime:", "camel-k-runtime-", 1)
			dependency := maven.NewDependency("org.apache.camel.k", artifactID, ctx.Request.RuntimeVersion)

			ctx.Project.AddDependency(dependency)
		default:
			return fmt.Errorf("unknown dependency type: %s", d)
		}
	}

	return nil
}

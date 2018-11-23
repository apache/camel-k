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
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/version"
)

// GenerateProject --
func GenerateProject(ctx *builder.Context) error {
	ctx.Project = maven.Project{
		XMLName:           xml.Name{Local: "project"},
		XMLNs:             "http://maven.apache.org/POM/4.0.0",
		XMLNsXsi:          "http://www.w3.org/2001/XMLSchema-instance",
		XsiSchemaLocation: "http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd",
		ModelVersion:      "4.0.0",
		GroupID:           "org.apache.camel.k.integration",
		ArtifactID:        "camel-k-integration",
		Version:           version.Version,
		DependencyManagement: maven.DependencyManagement{
			Dependencies: maven.Dependencies{
				Dependencies: []maven.Dependency{
					{
						//TODO: camel version should be retrieved from an external request or provided as static version
						GroupID:    "org.apache.camel",
						ArtifactID: "camel-bom",
						Version:    "2.22.2",
						Type:       "pom",
						Scope:      "import",
					},
				},
			},
		},
		Dependencies: maven.Dependencies{
			Dependencies: make([]maven.Dependency, 0),
		},
	}

	//
	// set-up dependencies
	//

	deps := &ctx.Project.Dependencies

	//
	// common
	//

	deps.Add(maven.Dependency{
		GroupID:    "org.apache.camel.k",
		ArtifactID: "camel-k-runtime-spring-boot",
		Version:    version.Version,
		Exclusions: &maven.Exclusions{
			Exclusions: []maven.Exclusion{
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
		},
	})

	//
	// others
	//

	for _, d := range ctx.Request.Dependencies {
		if strings.HasPrefix(d, "camel:") {
			if d == "camel:core" {
				continue
			}

			artifactID := strings.TrimPrefix(d, "camel:")

			if !strings.HasPrefix(artifactID, "camel-") {
				artifactID = "camel-" + artifactID
			}

			deps.Add(maven.Dependency{
				GroupID:    "org.apache.camel",
				ArtifactID: artifactID + "-starter",
				Version:    "2.22.2",
				Exclusions: &maven.Exclusions{
					Exclusions: []maven.Exclusion{
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
				},
			})
		} else if strings.HasPrefix(d, "mvn:") {
			mid := strings.TrimPrefix(d, "mvn:")
			gav := strings.Replace(mid, "/", ":", -1)

			deps.AddEncodedGAV(gav)
		} else if strings.HasPrefix(d, "runtime:") {
			if d == "runtime:jvm" {
				// common
				continue
			}
			if d == "runtime:spring-boot" {
				// common
				continue
			}

			artifactID := strings.Replace(d, "runtime:", "camel-k-runtime-", 1)

			deps.AddGAV("org.apache.camel.k", artifactID, version.Version)
		} else {
			return fmt.Errorf("unknown dependency type: %s", d)
		}
	}

	return nil
}

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

package assemble

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/apache/camel-k/pkg/build"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/sirupsen/logrus"

	"github.com/apache/camel-k/version"
)

type mavenAssembler struct {
	buffer chan assembleOperation
}

type assembleOperation struct {
	request build.Request
	output  chan build.AssembledOutput
}

// NewMavenAssembler create a new builder
func NewMavenAssembler(ctx context.Context) build.Assembler {
	assembler := mavenAssembler{
		buffer: make(chan assembleOperation, 100),
	}
	go assembler.assembleCycle(ctx)
	return &assembler
}

func (b *mavenAssembler) Assemble(request build.Request) <-chan build.AssembledOutput {
	res := make(chan build.AssembledOutput, 1)
	op := assembleOperation{
		request: request,
		output:  res,
	}
	b.buffer <- op
	return res
}

func (b *mavenAssembler) assembleCycle(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			b.buffer = nil
			return
		case op := <-b.buffer:
			now := time.Now()
			logrus.Info("Starting new Maven build")
			res := b.execute(&op.request)
			elapsed := time.Now().Sub(now)

			if res.Error != nil {
				logrus.Error("Error during Maven build (total time ", elapsed.Seconds(), " seconds): ", res.Error)
			} else {
				logrus.Info("Maven build completed in ", elapsed.Seconds(), " seconds")
			}

			op.output <- res
		}
	}
}

func (b *mavenAssembler) execute(request *build.Request) build.AssembledOutput {
	project, err := generateProject(request)
	if err != nil {
		return build.AssembledOutput{
			Error: err,
		}
	}

	res, err := maven.Process(project)
	if err != nil {
		return build.AssembledOutput{
			Error: err,
		}
	}

	output := build.AssembledOutput{
		Classpath: make([]build.ClasspathEntry, 0, len(res.Classpath)),
	}
	for _, e := range res.Classpath {
		output.Classpath = append(output.Classpath, build.ClasspathEntry{
			ID:       e.ID,
			Location: e.Location,
		})
	}

	return output
}

func generateProject(source *build.Request) (maven.Project, error) {
	project := maven.Project{
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

	deps := &project.Dependencies
	deps.AddGAV("org.apache.camel.k", "camel-k-runtime-jvm", version.Version)

	for _, d := range source.Dependencies {
		if strings.HasPrefix(d, "camel:") {
			artifactID := strings.TrimPrefix(d, "camel:")

			if !strings.HasPrefix(artifactID, "camel-") {
				artifactID = "camel-" + artifactID
			}

			deps.AddGAV("org.apache.camel", artifactID, "")
		} else if strings.HasPrefix(d, "mvn:") {
			mid := strings.TrimPrefix(d, "mvn:")
			gav := strings.Replace(mid, "/", ":", -1)

			deps.AddEncodedGAV(gav)
		} else if strings.HasPrefix(d, "runtime:") {
			artifactID := strings.Replace(d, "runtime:", "camel-k-runtime-", 1)

			deps.AddGAV("org.apache.camel.k", artifactID, version.Version)
		} else {
			return maven.Project{}, fmt.Errorf("unknown dependency type: %s", d)
		}
	}

	return project, nil
}

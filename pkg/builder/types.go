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
	"math"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/maven"
)

const (
	// InitPhase --
	InitPhase int32 = 0
	// ProjectGenerationPhase --
	ProjectGenerationPhase int32 = 10
	// ProjectBuildPhase --
	ProjectBuildPhase int32 = 20
	// ApplicationPackagePhase --
	ApplicationPackagePhase int32 = 30
	// ApplicationPublishPhase --
	ApplicationPublishPhase int32 = 40
	// NotifyPhase --
	NotifyPhase int32 = math.MaxInt32
)

func New(c client.Client) *Builder {
	return &Builder{
		log:    log.WithName("builder"),
		client: c,
	}
}

type Builder struct {
	log    log.Logger
	client client.Client
}

type Build struct {
	builder Builder
	build   *v1.Build
}

type Task interface {
	Do(ctx context.Context) v1.BuildStatus
}

type Step interface {
	ID() string
	Phase() int32
	execute(*builderContext) error
}

type resource struct {
	Target  string
	Content []byte
}

type builderContext struct {
	client.Client
	C                 context.Context
	Catalog           *camel.RuntimeCatalog
	Build             v1.BuilderTask
	BaseImage         string
	Error             error
	Namespace         string
	Path              string
	Artifacts         []v1.Artifact
	SelectedArtifacts []v1.Artifact
	Resources         []resource
	Maven             struct {
		Project        maven.Project
		SettingsData   []byte
		TrustStorePath string
	}
}

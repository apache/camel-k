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
	"math"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/camel"
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

// Builder --
type Builder interface {
	Run(ctx context.Context, ns string, build v1.BuilderTask) v1.BuildStatus
}

// Step --
type Step interface {
	ID() string
	Phase() int32
	Execute(*Context) error
}

type stepWrapper struct {
	StepID string
	phase  int32
	task   StepTask
}

func (s *stepWrapper) String() string {
	return fmt.Sprintf("%s@%d", s.StepID, s.phase)
}

func (s *stepWrapper) ID() string {
	return s.StepID
}

func (s *stepWrapper) Phase() int32 {
	return s.phase
}

func (s *stepWrapper) Execute(ctx *Context) error {
	return s.task(ctx)
}

// StepTask ---
type StepTask func(*Context) error

// NewStep --
func NewStep(phase int32, task StepTask) Step {
	s := stepWrapper{
		phase: phase,
		task:  task,
	}

	return &s
}

// Resource --
type Resource struct {
	Target  string
	Content []byte
}

// Context --
type Context struct {
	client.Client
	C                 context.Context
	Catalog           *camel.RuntimeCatalog
	Build             v1.BuilderTask
	BaseImage         string
	Image             string
	Digest            string
	Error             error
	Namespace         string
	Path              string
	Artifacts         []v1.Artifact
	SelectedArtifacts []v1.Artifact
	Resources         []Resource

	Maven struct {
		Project      maven.Project
		SettingsData []byte
	}
}

// HasRequiredImage --
func (c *Context) HasRequiredImage() bool {
	return c.Build.Image != ""
}

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
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/util/maven"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
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
	Submit(request Request) Result
	Purge(request Request)
}

// Step --
type Step interface {
	ID() string
	Phase() int32
	Execute(*Context) error
}

type stepWrapper struct {
	id    string
	phase int32
	task  StepTask
}

func (s *stepWrapper) String() string {
	return fmt.Sprintf("%s@%d", s.id, s.phase)
}

func (s *stepWrapper) ID() string {
	return s.id
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
func NewStep(ID string, phase int32, task StepTask) Step {
	s := stepWrapper{
		id:    ID,
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

// Request --
type Request struct {
	Meta         v1.ObjectMeta
	Platform     v1alpha1.IntegrationPlatformSpec
	Dependencies []string
	Repositories []string
	Steps        []Step
	BuildDir     string
	Image        string
	Resources    []Resource
}

// Task --
type Task struct {
	StartedAt   time.Time
	CompletedAt time.Time
}

// Elapsed --
func (t Task) Elapsed() time.Duration {
	return t.CompletedAt.Sub(t.StartedAt)
}

// Result represents the result of a build
type Result struct {
	Request     Request
	Image       string
	PublicImage string
	Error       error
	Status      Status
	Artifacts   []v1alpha1.Artifact
	Task        Task
}

// Context --
type Context struct {
	C                 context.Context
	Request           Request
	Image             string
	PublicImage       string
	Error             error
	Namespace         string
	Project           maven.Project
	Path              string
	Artifacts         []v1alpha1.Artifact
	SelectedArtifacts []v1alpha1.Artifact
	Archive           string
	ContextFiler      func(integrationContext *v1alpha1.IntegrationContext) bool
}

// HasRequiredImage --
func (c *Context) HasRequiredImage() bool {
	return c.Request.Image != ""
}

// GetImage --
func (c *Context) GetImage() string {
	if c.Request.Image != "" {
		return c.Request.Image
	}

	return c.Image
}

// PublishedImage --
type PublishedImage struct {
	Image        string
	Artifacts    []v1alpha1.Artifact
	Dependencies []string
}

// Status --
type Status int

const (
	// StatusSubmitted --
	StatusSubmitted Status = iota

	// StatusStarted --
	StatusStarted

	// StatusCompleted --
	StatusCompleted

	// StatusError --
	StatusError
)

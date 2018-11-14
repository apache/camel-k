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
	"time"

	"github.com/apache/camel-k/pkg/util/maven"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

// Builder --
type Builder interface {
	Submit(request Request) Result
	Purge(request Request)
}

// Step --
type Step interface {
	ID() string
	Phase() int
	Execute(*Context) error
}

type stepWrapper struct {
	id    string
	phase int
	task  func(*Context) error
}

func (s *stepWrapper) ID() string {
	return s.id
}

func (s *stepWrapper) Phase() int {
	return s.phase
}

func (s *stepWrapper) Execute(ctx *Context) error {
	return s.task(ctx)
}

// NewStep --
func NewStep(ID string, phase int, task func(*Context) error) Step {
	s := stepWrapper{
		id:    ID,
		phase: phase,
		task:  task,
	}

	return &s
}

// Identifier --
type Identifier struct {
	Name      string
	Qualifier string
}

func (r *Identifier) String() string {
	return r.Name + ":" + r.Qualifier
}

// Request --
type Request struct {
	Identifier   Identifier
	Platform     v1alpha1.IntegrationPlatformSpec
	Code         v1alpha1.SourceSpec
	Dependencies []string
	Steps        []Step
}

// Result represents the result of a build
type Result struct {
	Request            Request
	Status             Status
	Image              string
	Error              error
	Classpath          []string
	ProcessStartedAt   time.Time
	ProcessCompletedAt time.Time
}

// Context --
type Context struct {
	C         context.Context
	Namespace string
	Values    map[string]interface{}
	Request   Request
	Project   maven.Project
	Path      string
	Libraries []v1alpha1.Artifact
	Image     string
	Error     error
	StepsDone []string
	Archive   string
}

// PublishedImage --
type PublishedImage struct {
	Image     string
	Classpath []string
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

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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

// Build convert the Build CR in a struct that can be executable as an operator routine.
func (b *Builder) Build(build *v1.Build) *Build {
	return &Build{
		builder: *b,
		build:   build,
	}
}

// Task convert the task in a routine task which can be executed inside operator.
func (b *Build) Task(task v1.Task) Task {
	switch {
	case task.Builder != nil:
		return &builderTask{
			c:     b.builder.client,
			log:   b.builder.log,
			build: b.build,
			task:  task.Builder,
		}
	// Custom tasks are not supported in routines
	case task.Custom != nil:
		return &unsupportedTask{
			build: b.build,
			name:  task.Custom.Name,
		}
	case task.Package != nil:
		return &builderTask{
			c:     b.builder.client,
			log:   b.builder.log,
			build: b.build,
			task:  task.Package,
		}
	case task.Spectrum != nil:
		return &spectrumTask{
			c:     b.builder.client,
			build: b.build,
			task:  task.Spectrum,
		}
	case task.S2i != nil:
		return &s2iTask{
			c:     b.builder.client,
			build: b.build,
			task:  task.S2i,
		}
	case task.Jib != nil:
		return &jibTask{
			c:     b.builder.client,
			build: b.build,
			task:  task.Jib,
		}
	}

	return &emptyTask{
		build: b.build,
	}
}

type emptyTask struct {
	build *v1.Build
}

func (t *emptyTask) Do(_ context.Context) v1.BuildStatus {
	return v1.BuildStatus{
		Phase: v1.BuildPhaseError,
		Error: fmt.Sprintf("Cannot execute empty task in build [%s/%s]", t.build.Namespace, t.build.Name),
	}
}

type missingTask struct {
	build *v1.Build
	name  string
}

func (t *missingTask) Do(_ context.Context) v1.BuildStatus {
	return v1.BuildStatus{
		Phase: v1.BuildPhaseError,
		Error: fmt.Sprintf("No task with name [%s] in build [%s/%s]", t.name, t.build.Namespace, t.build.Name),
	}
}

type unsupportedTask struct {
	build *v1.Build
	name  string
}

func (t *unsupportedTask) Do(_ context.Context) v1.BuildStatus {
	return v1.BuildStatus{
		Phase: v1.BuildPhaseError,
		Error: fmt.Sprintf("Cannot execute task with name [%s] in build [%s/%s]", t.name, t.build.Namespace, t.build.Name),
	}
}

var _ Task = &missingTask{}

// TaskByName return the task identified by the name parameter.
func (b *Build) TaskByName(name string) Task {
	for _, task := range b.build.Spec.Tasks {
		switch {
		case task.Builder != nil && task.Builder.Name == name:
			return &builderTask{
				c:     b.builder.client,
				log:   b.builder.log,
				build: b.build,
				task:  task.Builder,
			}
		case task.Custom != nil && task.Custom.Name == name:
			return &unsupportedTask{
				build: b.build,
				name:  task.Custom.Name,
			}
		case task.Package != nil && task.Package.Name == name:
			return &builderTask{
				c:     b.builder.client,
				log:   b.builder.log,
				build: b.build,
				task:  task.Package,
			}
		case task.Spectrum != nil && task.Spectrum.Name == name:
			return &spectrumTask{
				c:     b.builder.client,
				build: b.build,
				task:  task.Spectrum,
			}
		case task.S2i != nil && task.S2i.Name == name:
			return &s2iTask{
				c:     b.builder.client,
				build: b.build,
				task:  task.S2i,
			}
		case task.Jib != nil && task.Jib.Name == name:
			return &jibTask{
				c:     b.builder.client,
				build: b.build,
				task:  task.Jib,
			}
		}
	}
	return &missingTask{
		build: b.build,
		name:  name,
	}
}

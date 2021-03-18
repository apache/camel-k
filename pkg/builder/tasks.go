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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func (b *Builder) Build(build *v1.Build) *Build {
	return &Build{
		builder: *b,
		build:   build,
	}
}

func (b *Build) Task(task v1.Task) Task {
	if task.Builder != nil {
		return &builderTask{
			c:     b.builder.client,
			log:   b.builder.log,
			build: b.build,
			task:  task.Builder,
		}
	} else if task.Buildah != nil {
		return &unsupportedTask{
			build: b.build,
			name:  task.Buildah.Name,
		}
	} else if task.Kaniko != nil {
		return &unsupportedTask{
			build: b.build,
			name:  task.Kaniko.Name,
		}
	} else if task.Spectrum != nil {
		return &spectrumTask{
			c:     b.builder.client,
			build: b.build,
			task:  task.Spectrum,
		}
	} else if task.S2i != nil {
		return &s2iTask{
			c:     b.builder.client,
			build: b.build,
			task:  task.S2i,
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

func (b *Build) TaskByName(name string) Task {
	for _, task := range b.build.Spec.Tasks {
		if task.Builder != nil && task.Builder.Name == name {
			return &builderTask{
				c:     b.builder.client,
				log:   b.builder.log,
				build: b.build,
				task:  task.Builder,
			}
		} else if task.Buildah != nil && task.Buildah.Name == name {
			return &unsupportedTask{
				build: b.build,
				name:  task.Buildah.Name,
			}
		} else if task.Kaniko != nil && task.Kaniko.Name == name {
			return &unsupportedTask{
				build: b.build,
				name:  task.Kaniko.Name,
			}
		} else if task.Spectrum != nil && task.Spectrum.Name == name {
			return &spectrumTask{
				c:     b.builder.client,
				build: b.build,
				task:  task.Spectrum,
			}
		} else if task.S2i != nil && task.S2i.Name == name {
			return &s2iTask{
				c:     b.builder.client,
				build: b.build,
				task:  task.S2i,
			}
		}
	}
	return &missingTask{
		build: b.build,
		name:  name,
	}
}

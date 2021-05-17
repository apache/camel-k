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
	"os"
	"path"
	"sort"
	"strconv"
	"time"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/log"
)

type builderTask struct {
	c     client.Client
	log   log.Logger
	build *v1.Build
	task  *v1.BuilderTask
}

var _ Task = &builderTask{}

func (t *builderTask) Do(ctx context.Context) v1.BuildStatus {
	result := v1.BuildStatus{}

	buildDir := t.task.BuildDir
	if buildDir == "" {
		// Use the working directory.
		// This is useful when the task is executed in-container,
		// so that its WorkingDir can be used to share state and
		// coordinate with other tasks.
		pwd, err := os.Getwd()
		if err != nil {
			return result.Failed(err)
		}
		buildDir = pwd
	}

	c := builderContext{
		Client:    t.c,
		C:         ctx,
		Path:      buildDir,
		Namespace: t.build.Namespace,
		Build:     *t.task,
		BaseImage: t.task.BaseImage,
	}

	// Add sources
	for _, data := range t.task.Sources {
		c.Resources = append(c.Resources, resource{
			Content: []byte(data.Content),
			Target:  path.Join("sources", data.Name),
		})
	}

	// Add resources
	for _, data := range t.task.Resources {
		t := path.Join("resources", data.Name)

		if data.MountPath != "" {
			t = path.Join(data.MountPath, data.Name)
		}

		c.Resources = append(c.Resources, resource{
			Content: []byte(data.Content),
			Target:  t,
		})
	}

	if result.Phase == v1.BuildPhaseFailed {
		return result
	}

	steps := make([]Step, 0)
	for _, step := range t.task.Steps {
		s, ok := stepsByID[step]
		if !ok {
			log.Info("Skipping unknown build step", "step", step)
			continue
		}
		steps = append(steps, s)
	}
	// Sort steps by phase
	sort.SliceStable(steps, func(i, j int) bool {
		return steps[i].Phase() < steps[j].Phase()
	})

	t.log.Infof("steps: %v", steps)
	for _, step := range steps {
		if c.Error != nil || result.Phase == v1.BuildPhaseInterrupted {
			break
		}

		select {
		case <-ctx.Done():
			result.Phase = v1.BuildPhaseInterrupted
		default:
			l := t.log.WithValues(
				"step", step.ID(),
				"phase", strconv.FormatInt(int64(step.Phase()), 10),
				"task", t.task.Name,
			)

			l.Infof("executing step")

			start := time.Now()
			c.Error = step.execute(&c)

			if c.Error == nil {
				l.Infof("step done in %f seconds", time.Since(start).Seconds())
			} else {
				l.Infof("step failed with error: %s", c.Error)
			}
		}
	}

	if result.Phase != v1.BuildPhaseInterrupted {
		result.BaseImage = c.BaseImage

		if c.Error != nil {
			result.Error = c.Error.Error()
			result.Phase = v1.BuildPhaseFailed
		}

		result.Artifacts = make([]v1.Artifact, 0, len(c.Artifacts))
		result.Artifacts = append(result.Artifacts, c.Artifacts...)

		t.log.Infof("dependencies: %s", t.task.Dependencies)
		t.log.Infof("artifacts: %s", artifactIDs(c.Artifacts))
		t.log.Infof("artifacts selected: %s", artifactIDs(c.SelectedArtifacts))
		t.log.Infof("base image: %s", t.task.BaseImage)
		t.log.Infof("resolved base image: %s", c.BaseImage)
	} else {
		t.log.Infof("build task %s interrupted", t.task.Name)
	}

	return result
}

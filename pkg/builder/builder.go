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
	"errors"
	"io/ioutil"
	"os"
	"sort"
	"time"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/cancellable"
	"github.com/apache/camel-k/pkg/util/log"
)

type defaultBuilder struct {
	log    log.Logger
	ctx    cancellable.Context
	client client.Client
}

// New --
func New(c client.Client) Builder {
	m := defaultBuilder{
		log:    log.WithName("builder"),
		ctx:    cancellable.NewContext(),
		client: c,
	}

	return &m
}

// Build --
func (b *defaultBuilder) Build(request Request) Result {
	result := Result{
		Builder: b,
		Request: request,
	}

	// create tmp path
	buildDir := request.BuildDir
	if buildDir == "" {
		buildDir = os.TempDir()
	}
	builderPath, err := ioutil.TempDir(buildDir, "builder-")
	if err != nil {
		log.Error(err, "Unexpected error while creating a temporary dir")

		result.Status = v1alpha1.BuildPhaseFailed
		result.Error = err
	}

	defer os.RemoveAll(builderPath)

	c := Context{
		Client:    b.client,
		Catalog:   request.Catalog,
		Path:      builderPath,
		Namespace: request.Meta.Namespace,
		Request:   request,
		Image:     request.Platform.Build.BaseImage,
	}

	if request.Image != "" {
		c.Image = request.Image
	}

	// base image is mandatory
	if c.Image == "" {
		result.Status = v1alpha1.BuildPhaseFailed
		result.Image = ""
		result.PublicImage = ""
		result.Error = errors.New("no base image defined")
		result.Task.CompletedAt = time.Now()
	}

	c.BaseImage = c.Image

	if result.Status == v1alpha1.BuildPhaseFailed {
		return result
	}

	// Sort steps by phase
	sort.SliceStable(request.Steps, func(i, j int) bool {
		return request.Steps[i].Phase() < request.Steps[j].Phase()
	})

	b.log.Infof("steps: %v", request.Steps)
	for _, step := range request.Steps {
		if c.Error != nil || result.Status == v1alpha1.BuildPhaseInterrupted {
			break
		}

		select {
		case <-request.C.Done():
			result.Status = v1alpha1.BuildPhaseInterrupted
		default:
			l := b.log.WithValues(
				"step", step.ID(),
				"phase", step.Phase(),
				"context", request.Meta.Name,
			)

			l.Infof("executing step")

			start := time.Now()
			c.Error = step.Execute(&c)

			if c.Error == nil {
				l.Infof("step done in %f seconds", time.Since(start).Seconds())
			} else {
				l.Infof("step failed with error: %s", c.Error)
			}
		}
	}

	result.Task.CompletedAt = time.Now()

	if result.Status != v1alpha1.BuildPhaseInterrupted {
		result.Status = v1alpha1.BuildPhaseSucceeded
		result.BaseImage = c.BaseImage
		result.Image = c.Image
		result.PublicImage = c.PublicImage
		result.Error = c.Error

		if result.Error != nil {
			result.Status = v1alpha1.BuildPhaseFailed
		}

		result.Artifacts = make([]v1alpha1.Artifact, 0, len(c.Artifacts))
		result.Artifacts = append(result.Artifacts, c.Artifacts...)

		b.log.Infof("build request %s executed in %f seconds", request.Meta.Name, result.Task.Elapsed().Seconds())
		b.log.Infof("dependencies: %s", request.Dependencies)
		b.log.Infof("artifacts: %s", ArtifactIDs(c.Artifacts))
		b.log.Infof("artifacts selected: %s", ArtifactIDs(c.SelectedArtifacts))
		b.log.Infof("requested image: %s", request.Image)
		b.log.Infof("base image: %s", c.BaseImage)
		b.log.Infof("resolved image: %s", c.Image)
		b.log.Infof("resolved public image: %s", c.PublicImage)
	} else {
		b.log.Infof("build request %s interrupted after %f seconds", request.Meta.Name, result.Task.Elapsed().Seconds())
	}

	return result
}

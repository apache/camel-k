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

package build

import (
	"context"
	"path"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/builder/kaniko"
	"github.com/apache/camel-k/pkg/builder/s2i"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/cancellable"
	logger "github.com/apache/camel-k/pkg/util/log"
)

func SubmitBuildRequest(ctx context.Context, c client.Client, build *v1alpha1.Build, log logger.Logger, callback func(v1alpha1.BuildPhase)) error {
	b := builder.NewLocalBuilder(c, build.Namespace)

	catalog, err := camel.Catalog(ctx, c, build.Namespace, build.Spec.CamelVersion)
	if err != nil {
		return err
	}

	stepsDictionary := make(map[string]builder.Step)
	for _, step := range kaniko.DefaultSteps {
		stepsDictionary[step.ID()] = step
	}
	for _, step := range s2i.DefaultSteps {
		stepsDictionary[step.ID()] = step
	}

	steps := make([]builder.Step, 0)
	for _, step := range build.Spec.Steps {
		s, ok := stepsDictionary[step]
		if !ok {
			log.Info("Skipping unknown build step", "step", step)
			continue
		}
		steps = append(steps, s)
	}

	request := builder.Request{
		C:              cancellable.NewContext(),
		Catalog:        catalog,
		RuntimeVersion: build.Spec.RuntimeVersion,
		Meta:           build.Spec.Meta,
		Dependencies:   build.Spec.Dependencies,
		Repositories:   build.Spec.Repositories,
		Steps:          steps,
		BuildDir:       build.Spec.BuildDir,
		Platform:       build.Spec.Platform,
		Image:          build.Spec.Image,
	}

	// Add sources
	for _, data := range build.Spec.Sources {
		request.Resources = append(request.Resources, builder.Resource{
			Content: []byte(data.Content),
			Target:  path.Join("sources", data.Name),
		})
	}

	// Add resources
	for _, data := range build.Spec.Resources {
		t := path.Join("resources", data.Name)

		if data.MountPath != "" {
			t = path.Join(data.MountPath, data.Name)
		}

		request.Resources = append(request.Resources, builder.Resource{
			Content: []byte(data.Content),
			Target:  t,
		})
	}

	b.Submit(request, func(result *builder.Result) {
		buildHandler(build, result, c, request.C, log, callback)
	})

	return nil
}

func buildHandler(build *v1alpha1.Build, result *builder.Result, c client.Client, ctx cancellable.Context, log logger.Logger, callback func(v1alpha1.BuildPhase)) {
	// Refresh build
	err := c.Get(ctx, types.NamespacedName{Namespace: build.Namespace, Name: build.Name}, build)
	if err != nil {
		log.Error(err, "Build refresh failed")
	}

	switch result.Status {

	case v1alpha1.BuildPhaseRunning:
		log.Info("Build started")

		b := build.DeepCopy()
		b.Status.Phase = v1alpha1.BuildPhaseRunning
		err = updateBuildStatus(b, c, ctx, log)

	case v1alpha1.BuildPhaseInterrupted:
		log.Info("Build interrupted")

		b := build.DeepCopy()
		b.Status.Phase = v1alpha1.BuildPhaseInterrupted
		err = updateBuildStatus(b, c, ctx, log)

	case v1alpha1.BuildPhaseFailed:
		log.Error(result.Error, "Build failed")

		b := build.DeepCopy()
		b.Status.Phase = v1alpha1.BuildPhaseFailed
		b.Status.Error = result.Error.Error()
		err = updateBuildStatus(b, c, ctx, log)

	case v1alpha1.BuildPhaseSucceeded:
		log.Info("Build completed")

		b := build.DeepCopy()
		b.Status.Phase = v1alpha1.BuildPhaseSucceeded
		b.Status.Image = result.Image
		b.Status.BaseImage = result.BaseImage
		b.Status.PublicImage = result.PublicImage
		b.Status.Artifacts = result.Artifacts
		err = updateBuildStatus(b, c, ctx, log)
	}

	if callback != nil {
		if err != nil {
			callback(v1alpha1.BuildPhaseFailed)
		} else {
			callback(result.Status)
		}
	}
}

func updateBuildStatus(b *v1alpha1.Build, c client.Client, ctx cancellable.Context, log logger.Logger) error {
	err := c.Status().Update(ctx, b)
	if err != nil {
		if k8serrors.IsConflict(err) {
			// Refresh build
			err := c.Get(ctx, types.NamespacedName{Namespace: b.Namespace, Name: b.Name}, b)
			if err != nil {
				log.Error(err, "Build refresh failed")
				return err
			}
			return updateBuildStatus(b, c, ctx, log)
		}
		log.Error(err, "Build update failed")
		return err
	}
	return nil
}

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

package main

import (
	"flag"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"math/rand"
	"os"
	"runtime"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/builder/kaniko"
	"github.com/apache/camel-k/pkg/builder/s2i"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/cancellable"
	"github.com/apache/camel-k/pkg/util/defaults"
)

var log = logf.Log.WithName("builder")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Camel K Version: %v", defaults.Version))
}

var completed = make(chan bool)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	flag.Parse()

	logf.SetLogger(logf.ZapLogger(false))

	printVersion()

	c, err := client.NewClient()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := cancellable.NewContext()

	build := &v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: os.Args[1],
			Name:      os.Args[2],
		},
	}

	err = c.Get(ctx, types.NamespacedName{Namespace: build.GetNamespace(), Name: build.GetName()}, build)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	b := builder.NewLocalBuilder(c, build.Namespace)

	catalog, err := camel.Catalog(ctx, c, build.Namespace, build.Spec.CamelVersion)

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
		C:              ctx,
		Catalog:        catalog,
		RuntimeVersion: build.Spec.RuntimeVersion,
		Meta:           build.Spec.Meta,
		Dependencies:   build.Spec.Dependencies,
		Repositories:   build.Spec.Repositories,
		Steps:          steps,
		BuildDir:       "",
		Platform:       build.Spec.Platform,
		Image:          build.Spec.Image,
		//Resources:      build.Spec.Resources,
	}

	b.Submit(request, func(result *builder.Result) {
		buildHandler(build, result, c, ctx)
	})

	for {
		select {
		case success := <-completed:
			if !success {
				os.Exit(1)
			} else {
				os.Exit(0)
			}
		default:
			log.V(1).Info("Waiting for the build to complete...")
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func buildHandler(build *v1alpha1.Build, result *builder.Result, c client.Client, ctx cancellable.Context) {
	// Refresh build
	err := c.Get(ctx, types.NamespacedName{Namespace: build.GetNamespace(), Name: build.GetName()}, build)
	if err != nil {
		log.Error(err, "Build refresh failed")
		completed <- false
	}

	switch result.Status {
	//case v1alpha1.BuildScheduled:
	//	log.Info("Build submitted")

	case v1alpha1.BuildStarted:
		log.Info("Build started")

		b := build.DeepCopy()
		b.Status.Phase = v1alpha1.BuildStarted
		updateBuildStatus(b, c, ctx)

	case v1alpha1.BuildInterrupted:
		log.Info("Build interrupted")

		b := build.DeepCopy()
		b.Status.Phase = v1alpha1.BuildInterrupted
		updateBuildStatus(b, c, ctx)

		completed <- false

	case v1alpha1.BuildError:
		log.Error(result.Error, "Build error")

		b := build.DeepCopy()
		b.Status.Phase = v1alpha1.BuildError
		b.Status.Error = result.Error.Error()
		updateBuildStatus(b, c, ctx)

		completed <- false

	case v1alpha1.BuildCompleted:
		log.Info("Build completed")

		b := build.DeepCopy()
		b.Status.Phase = v1alpha1.BuildCompleted
		b.Status.Image = result.Image
		b.Status.BaseImage = result.BaseImage
		b.Status.PublicImage = result.PublicImage
		b.Status.Artifacts = result.Artifacts
		updateBuildStatus(b, c, ctx)

		completed <- true
	}
}

func updateBuildStatus(b *v1alpha1.Build, c client.Client, ctx cancellable.Context) {
	err := c.Status().Update(ctx, b)
	if err != nil {
		log.Error(err, "Build update failed")
		completed <- false
	}
}

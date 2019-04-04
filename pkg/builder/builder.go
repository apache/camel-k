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
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/cancellable"
	"github.com/apache/camel-k/pkg/util/log"
)

// ********************************
//
// Default builder
//
// ********************************

type buildTask struct {
	handlers []func(*Result)
	request  Request
}

type localBuilder struct {
	log       log.Logger
	ctx       cancellable.Context
	client    client.Client
	tasks     chan buildTask
	interrupt chan bool
	request   sync.Map
	running   int32
	namespace string
}

// NewLocalBuilder --
func NewLocalBuilder(c client.Client, namespace string) Builder {
	m := localBuilder{
		log:       log.WithName("local builder"),
		ctx:       cancellable.NewContext(),
		client:    c,
		tasks:     make(chan buildTask),
		interrupt: make(chan bool, 1),
		running:   0,
		namespace: namespace,
	}

	return &m
}

func (b *localBuilder) IsBuilding(object metav1.ObjectMeta) bool {
	_, ok := b.request.Load(object.Name)

	return ok
}

// Submit --
func (b *localBuilder) Submit(request Request, handlers ...func(*Result)) {
	if atomic.CompareAndSwapInt32(&b.running, 0, 1) {
		go b.loop()
	}

	result, present := b.request.Load(request.Meta.Name)
	if !present || result == nil {
		r := Result{
			Builder: b,
			Request: request,
			Status:  v1alpha1.BuildPhasePending,
		}

		b.log.Infof("submitting request: %+v", request)

		for _, handler := range handlers {
			handler(&r)
		}

		b.request.Store(request.Meta.Name, r)
		b.tasks <- buildTask{handlers: handlers, request: request}
	}
}

func (b *localBuilder) Close() {
	b.ctx.Cancel()
}

// ********************************
//
// Helpers
//
// ********************************

func (b *localBuilder) loop() {
	for atomic.LoadInt32(&b.running) == 1 {
		select {
		case <-b.ctx.Done():
			b.interrupt <- true

			close(b.interrupt)
			close(b.tasks)

			atomic.StoreInt32(&b.running, 0)
		case t, ok := <-b.tasks:
			if ok {
				b.log.Infof("executing request: %+v", t.request)
				b.process(t.request, t.handlers...)
			}
		}
	}
}

func (b *localBuilder) process(request Request, handlers ...func(*Result)) {
	result, present := b.request.Load(request.Meta.Name)
	if !present || result == nil {

		r := result.(Result)
		r.Status = v1alpha1.BuildPhaseFailed
		r.Error = fmt.Errorf("no info found for: %s/%s", request.Meta.Namespace, request.Meta.Name)

		b.log.Error(r.Error, "error processing request")

		for _, handler := range handlers {
			handler(&r)
		}

		return
	}

	// update the status
	r := result.(Result)
	r.Status = v1alpha1.BuildPhaseRunning
	r.Task.StartedAt = time.Now()

	for _, handler := range handlers {
		handler(&r)
	}

	// create tmp path
	buildDir := request.BuildDir
	if buildDir == "" {
		buildDir = os.TempDir()
	}
	builderPath, err := ioutil.TempDir(buildDir, "builder-")
	if err != nil {
		log.Error(err, "Unexpected error while creating a temporary dir")

		r.Status = v1alpha1.BuildPhaseFailed
		r.Error = err
	}

	defer os.RemoveAll(builderPath)
	defer b.request.Delete(request.Meta.Name)

	c := Context{
		Client:    b.client,
		Catalog:   request.Catalog,
		Path:      builderPath,
		Namespace: b.namespace,
		Request:   request,
		Image:     request.Platform.Build.BaseImage,
	}

	if request.Image != "" {
		c.Image = request.Image
	}

	// base image is mandatory
	if c.Image == "" {
		r.Status = v1alpha1.BuildPhaseFailed
		r.Image = ""
		r.PublicImage = ""
		r.Error = errors.New("no base image defined")
		r.Task.CompletedAt = time.Now()

		// update the cache
		b.request.Store(request.Meta.Name, r)
	}

	c.BaseImage = c.Image

	// update the cache
	b.request.Store(request.Meta.Name, r)

	if r.Status == v1alpha1.BuildPhaseFailed {
		for _, handler := range handlers {
			handler(&r)
		}

		return
	}

	// Sort steps by phase
	sort.SliceStable(request.Steps, func(i, j int) bool {
		return request.Steps[i].Phase() < request.Steps[j].Phase()
	})

	b.log.Infof("steps: %v", request.Steps)
	for _, step := range request.Steps {
		if c.Error != nil || r.Status == v1alpha1.BuildPhaseInterrupted {
			break
		}

		select {
		case <-b.interrupt:
			c.Error = errors.New("builder canceled")
		case <-request.C.Done():
			r.Status = v1alpha1.BuildPhaseInterrupted
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

	r.Task.CompletedAt = time.Now()

	if r.Status != v1alpha1.BuildPhaseInterrupted {
		r.Status = v1alpha1.BuildPhaseSucceeded
		r.BaseImage = c.BaseImage
		r.Image = c.Image
		r.PublicImage = c.PublicImage
		r.Error = c.Error

		if r.Error != nil {
			r.Status = v1alpha1.BuildPhaseFailed
		}

		r.Artifacts = make([]v1alpha1.Artifact, 0, len(c.Artifacts))
		r.Artifacts = append(r.Artifacts, c.Artifacts...)

		b.log.Infof("builder request %s executed in %f seconds", request.Meta.Name, r.Task.Elapsed().Seconds())
		b.log.Infof("dependencies: %s", request.Dependencies)
		b.log.Infof("artifacts: %s", ArtifactIDs(c.Artifacts))
		b.log.Infof("artifacts selected: %s", ArtifactIDs(c.SelectedArtifacts))
		b.log.Infof("requested image: %s", request.Image)
		b.log.Infof("base image: %s", c.BaseImage)
		b.log.Infof("resolved image: %s", c.Image)
		b.log.Infof("resolved public image: %s", c.PublicImage)
	} else {
		b.log.Infof("builder request %s interrupted after %f seconds", request.Meta.Name, r.Task.Elapsed().Seconds())
	}

	// update the cache
	b.request.Store(request.Meta.Name, r)

	for _, handler := range handlers {
		handler(&r)
	}
}

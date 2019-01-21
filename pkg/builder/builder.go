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
	"errors"
	"io/ioutil"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/sirupsen/logrus"
)

// ********************************
//
// Default builder
//
// ********************************

type buildTask struct {
	handler func(Result)
	request Request
}

type defaultBuilder struct {
	log       *logrus.Entry
	ctx       context.Context
	client    client.Client
	tasks     chan buildTask
	interrupt chan bool
	request   sync.Map
	running   int32
	namespace string
}

// New --
func New(ctx context.Context, c client.Client, namespace string) Builder {
	m := defaultBuilder{
		log:       logrus.WithField("logger", "builder"),
		ctx:       ctx,
		client:    c,
		tasks:     make(chan buildTask),
		interrupt: make(chan bool, 1),
		running:   0,
		namespace: namespace,
	}

	return &m
}

func (b *defaultBuilder) IsBuilding(object v1.ObjectMeta) bool {
	_, ok := b.request.Load(object.Name)

	return ok
}

// Submit --
func (b *defaultBuilder) Submit(request Request, handler func(Result)) {
	if atomic.CompareAndSwapInt32(&b.running, 0, 1) {
		go b.loop()
	}

	result, present := b.request.Load(request.Meta.Name)
	if !present || result == nil {
		result = Result{
			Builder: b,
			Request: request,
			Status:  StatusSubmitted,
		}

		b.log.Infof("submitting request: %+v", request)

		b.request.Store(request.Meta.Name, result)
		b.tasks <- buildTask{handler: handler, request: request}
	}
}

// ********************************
//
// Helpers
//
// ********************************

func (b *defaultBuilder) loop() {
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
				b.process(t.request, t.handler)
			}
		}
	}
}

func (b *defaultBuilder) process(request Request, handler func(Result)) {
	result, present := b.request.Load(request.Meta.Name)
	if !present || result == nil {
		b.log.Panicf("no info found for: %+v", request.Meta.Name)
	}

	// update the status
	r := result.(Result)
	r.Status = StatusStarted
	r.Task.StartedAt = time.Now()

	if handler != nil {
		handler(r)
	}

	// create tmp path
	buildDir := request.BuildDir
	if buildDir == "" {
		buildDir = os.TempDir()
	}
	builderPath, err := ioutil.TempDir(buildDir, "builder-")
	if err != nil {
		logrus.Warning("Unexpected error while creating a temporary dir ", err)
		r.Status = StatusError
		r.Error = err
	}

	defer os.RemoveAll(builderPath)
	defer b.request.Delete(request.Meta.Name)

	c := Context{
		C:         b.ctx,
		Client:    b.client,
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
		r.Status = StatusError
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

	if r.Status == StatusError {
		if handler != nil {
			handler(r)
		}

		return
	}

	// Sort steps by phase
	sort.SliceStable(request.Steps, func(i, j int) bool {
		return request.Steps[i].Phase() < request.Steps[j].Phase()
	})

	b.log.Infof("steps: %v", request.Steps)
	for _, step := range request.Steps {
		if c.Error != nil {
			break
		}

		select {
		case <-b.interrupt:
			c.Error = errors.New("build canceled")
		default:
			l := b.log.WithFields(logrus.Fields{
				"step":    step.ID(),
				"phase":   step.Phase(),
				"context": request.Meta.Name,
			})

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

	r.Status = StatusCompleted
	r.BaseImage = c.BaseImage
	r.Image = c.Image
	r.PublicImage = c.PublicImage
	r.Error = c.Error
	r.Task.CompletedAt = time.Now()

	if r.Error != nil {
		r.Status = StatusError
	}

	r.Artifacts = make([]v1alpha1.Artifact, 0, len(c.Artifacts))
	r.Artifacts = append(r.Artifacts, c.Artifacts...)

	// update the cache
	b.request.Store(request.Meta.Name, r)

	b.log.Infof("build request %s executed in %f seconds", request.Meta.Name, r.Task.Elapsed().Seconds())
	b.log.Infof("dependencies: %s", request.Dependencies)
	b.log.Infof("artifacts: %s", ArtifactIDs(c.Artifacts))
	b.log.Infof("artifacts selected: %s", ArtifactIDs(c.SelectedArtifacts))
	b.log.Infof("requested image: %s", request.Image)
	b.log.Infof("base image: %s", c.BaseImage)
	b.log.Infof("resolved image: %s", c.Image)
	b.log.Infof("resolved public image: %s", c.PublicImage)

	if handler != nil {
		handler(r)
	}
}

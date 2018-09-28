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
	"errors"
	"sync"
)

// Manager represent the main facade to the image build system
type Manager struct {
	ctx       context.Context
	builds    sync.Map
	assembler Assembler
	packager  Packager
	publisher Publisher
}

// NewManager creates an instance of the build manager using the given assembler, packager and publisher
func NewManager(ctx context.Context, assembler Assembler, packager Packager, publisher Publisher) *Manager {
	return &Manager{
		ctx:       ctx,
		assembler: assembler,
		packager:  packager,
		publisher: publisher,
	}
}

// Get retrieve the build result associated to the given build identifier
func (m *Manager) Get(identifier Identifier) Result {
	info, present := m.builds.Load(identifier)
	if !present || info == nil {
		return noBuildInfo()
	}

	return info.(Result)
}

// Start starts a new build
func (m *Manager) Start(request Request) {
	m.builds.Store(request.Identifier, initialBuildInfo(request))

	assembleChannel := m.assembler.Assemble(request)
	go func() {
		var assembled AssembledOutput
		select {
		case <-m.ctx.Done():
			m.builds.Store(request.Identifier, canceledBuildInfo(request))
			return
		case assembled = <-assembleChannel:
			if assembled.Error != nil {
				m.builds.Store(request.Identifier, failedAssembleBuildInfo(request, assembled))
				return
			}
		}

		packageChannel := m.packager.Package(request, assembled)
		var packaged PackagedOutput
		select {
		case <-m.ctx.Done():
			m.builds.Store(request.Identifier, canceledBuildInfo(request))
			return
		case packaged = <-packageChannel:
			if packaged.Error != nil {
				m.builds.Store(request.Identifier, failedPackageBuildInfo(request, packaged))
				return
			}
		}
		defer m.packager.Cleanup(packaged)

		publishChannel := m.publisher.Publish(request, assembled, packaged)
		var published PublishedOutput
		select {
		case <-m.ctx.Done():
			m.builds.Store(request.Identifier, canceledBuildInfo(request))
			return
		case published = <-publishChannel:
			if published.Error != nil {
				m.builds.Store(request.Identifier, failedPublishBuildInfo(request, published))
				return
			}
		}

		m.builds.Store(request.Identifier, completeResult(request, assembled, published))
	}()
}

func noBuildInfo() Result {
	return Result{
		Status: StatusNotRequested,
	}
}

func initialBuildInfo(request Request) Result {
	return Result{
		Request: request,
		Status:  StatusStarted,
	}
}

func canceledBuildInfo(request Request) Result {
	return Result{
		Request: request,
		Error:   errors.New("build canceled"),
		Status:  StatusError,
	}
}

func failedAssembleBuildInfo(request Request, output AssembledOutput) Result {
	return Result{
		Request: request,
		Error:   output.Error,
		Status:  StatusError,
	}
}

func failedPackageBuildInfo(request Request, output PackagedOutput) Result {
	return Result{
		Request: request,
		Error:   output.Error,
		Status:  StatusError,
	}
}

func failedPublishBuildInfo(request Request, output PublishedOutput) Result {
	return Result{
		Request: request,
		Error:   output.Error,
		Status:  StatusError,
	}
}

func completeResult(request Request, a AssembledOutput, p PublishedOutput) Result {
	return Result{
		Request:   request,
		Status:    StatusCompleted,
		Classpath: a.Classpath,
		Image:     p.Image,
	}
}

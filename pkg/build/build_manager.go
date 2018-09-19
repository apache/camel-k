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
	"sync"
)

// Manager represent the main facade to the image build system
type Manager struct {
	builds    sync.Map
	ctx       context.Context
	namespace string
	builder   Builder
}

// NewManager creates an instance of the build manager using the given builder
func NewManager(ctx context.Context, namespace string, builder Builder) *Manager {
	return &Manager{
		ctx:       ctx,
		namespace: namespace,
		builder:   builder,
	}
}

// Get retrieve the build result associated to the given build identifier
func (m *Manager) Get(identifier Identifier) Result {
	info, present := m.builds.Load(identifier)
	if !present || info == nil {
		return noBuildInfo()
	}

	return *info.(*Result)
}

// Start starts a new build
func (m *Manager) Start(source Request) {
	initialBuildInfo := initialBuildInfo(&source)
	m.builds.Store(source.Identifier, &initialBuildInfo)

	resChannel := m.builder.Build(source)
	go func() {
		res := <-resChannel
		m.builds.Store(res.Request.Identifier, &res)
	}()
}

func noBuildInfo() Result {
	return Result{
		Status: StatusNotRequested,
	}
}

func initialBuildInfo(source *Request) Result {
	return Result{
		Request: source,
		Status:  StatusStarted,
	}
}

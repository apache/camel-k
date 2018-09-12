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

	"github.com/apache/camel-k/pkg/build/api"
	"github.com/apache/camel-k/pkg/build/local"
)

// main facade to the image build system
type Manager struct {
	builds  sync.Map
	builder api.Builder
}

func NewManager(ctx context.Context, namespace string) *Manager {
	return &Manager{
		builder: local.NewLocalBuilder(ctx, namespace),
	}
}

func (m *Manager) Get(identifier api.BuildIdentifier) api.BuildResult {
	if info, present := m.builds.Load(identifier); !present || info == nil {
		return noBuildInfo()
	} else {
		return *info.(*api.BuildResult)
	}
}

func (m *Manager) Start(source api.BuildSource) {
	initialBuildInfo := initialBuildInfo(&source)
	m.builds.Store(source.Identifier, &initialBuildInfo)

	resChannel := m.builder.Build(source)
	go func() {
		res := <-resChannel
		m.builds.Store(res.Source.Identifier, &res)
	}()
}

func noBuildInfo() api.BuildResult {
	return api.BuildResult{
		Status: api.BuildStatusNotRequested,
	}
}

func initialBuildInfo(source *api.BuildSource) api.BuildResult {
	return api.BuildResult{
		Source: source,
		Status: api.BuildStatusStarted,
	}
}

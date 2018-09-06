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
	"sync"
	"github.com/apache/camel-k/pkg/build/local"
	"context"
	"github.com/apache/camel-k/pkg/build/api"
)

// main facade to the image build system
type BuildManager struct {
	builds	map[string]*api.BuildResult
	mutex	sync.Mutex
	builder	api.Builder
}

func NewBuildManager(ctx context.Context, namespace string) *BuildManager {
	return &BuildManager{
		builds: make(map[string]*api.BuildResult),
		builder: local.NewLocalBuilder(ctx, namespace),
	}
}

func (m *BuildManager) Get(identifier string) api.BuildResult {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if info, present := m.builds[identifier]; !present || info == nil {
		return noBuildInfo()
	} else {
		return *info
	}
}

func (m *BuildManager) Start(source api.BuildSource) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	initialBuildInfo := initialBuildInfo()
	m.builds[source.Identifier] = &initialBuildInfo

	resChannel := m.builder.Build(source)
	go func() {
		res := <-resChannel
		m.mutex.Lock()
		defer m.mutex.Unlock()

		m.builds[res.Source.Identifier] = &res
	}()
}

func noBuildInfo() api.BuildResult {
	return api.BuildResult{
		Status: api.BuildStatusNotRequested,
	}
}

func initialBuildInfo() api.BuildResult {
	return api.BuildResult{
		Status: api.BuildStatusStarted,
	}
}
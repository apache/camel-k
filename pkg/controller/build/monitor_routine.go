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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// NewMonitorRoutineAction creates a new monitor action for scheduled routine
func NewMonitorRoutineAction(r *sync.Map) Action {
	return &monitorRoutineAction{
		routines: r,
	}
}

type monitorRoutineAction struct {
	baseAction
	routines *sync.Map
}

// Name returns a common name of the action
func (action *monitorRoutineAction) Name() string {
	return "monitor-routine"
}

// CanHandle tells whether this action can handle the build
func (action *monitorRoutineAction) CanHandle(build *v1.Build) bool {
	return build.Status.Phase == v1.BuildPhasePending || build.Status.Phase == v1.BuildPhaseRunning
}

// Handle handles the builds
func (action *monitorRoutineAction) Handle(ctx context.Context, build *v1.Build) (*v1.Build, error) {
	// Check the build routine
	if _, ok := action.routines.Load(build.Name); !ok && build.Status.Phase != v1.BuildPhaseFailed {
		// and recover the build if it's missing. This can happen when the operator
		// stops abruptly and restarts or the build status update fails.
		build.Status.Phase = v1.BuildPhaseFailed
		return build, nil
	}

	return nil, nil
}

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

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
)

// NewScheduleRoutineAction creates a new schedule routine action
func NewScheduleRoutineAction(reader client.Reader, b builder.Builder, r *sync.Map) Action {
	return &scheduleRoutineAction{
		reader:   reader,
		builder:  b,
		routines: r,
	}
}

type scheduleRoutineAction struct {
	baseAction
	lock     sync.Mutex
	reader   client.Reader
	builder  builder.Builder
	routines *sync.Map
}

// Name returns a common name of the action
func (action *scheduleRoutineAction) Name() string {
	return "schedule-routine"
}

// CanHandle tells whether this action can handle the build
func (action *scheduleRoutineAction) CanHandle(build *v1alpha1.Build) bool {
	return build.Status.Phase == v1alpha1.BuildPhaseScheduling &&
		build.Spec.Platform.Build.BuildStrategy == v1alpha1.IntegrationPlatformBuildStrategyRoutine
}

// Handle handles the builds
func (action *scheduleRoutineAction) Handle(ctx context.Context, build *v1alpha1.Build) (*v1alpha1.Build, error) {
	// Enter critical section
	action.lock.Lock()
	defer action.lock.Unlock()

	builds := &v1alpha1.BuildList{}
	// We use the non-caching client as informers cache is not invalidated nor updated
	// atomically by write operations
	err := action.reader.List(ctx, builds, client.InNamespace(build.Namespace))
	if err != nil {
		return nil, err
	}

	// Emulate a serialized working queue to only allow one build to run at a given time.
	// This is currently necessary for the incremental build to work as expected.
	for _, b := range builds.Items {
		if b.Status.Phase == v1alpha1.BuildPhasePending || b.Status.Phase == v1alpha1.BuildPhaseRunning {
			// Let's requeue the build in case one is already running
			return nil, nil
		}
	}

	// Transition the build to pending state
	// This must be done in the critical section rather than delegated to the controller
	target := build.DeepCopy()
	target.Status.Phase = v1alpha1.BuildPhasePending
	action.L.Info("Build state transition", "phase", target.Status.Phase)
	err = action.client.Status().Update(ctx, target)
	if err != nil {
		return nil, err
	}

	// Start the build
	progress := action.builder.Build(build.Spec)
	// And follow the build progress asynchronously to avoid blocking the reconcile loop
	go func() {
		for status := range progress {
			target := build.DeepCopy()
			target.Status = status
			// Copy the failure field from the build to persist recovery state
			target.Status.Failure = build.Status.Failure
			// Patch the build status with the current progress
			err := action.client.Status().Patch(ctx, target, client.MergeFrom(build))
			if err != nil {
				action.L.Errorf(err, "Error while updating build status: %s", build.Name)
			}
			build.Status = target.Status
		}
	}()

	action.routines.Store(build.Name, true)

	return nil, nil
}

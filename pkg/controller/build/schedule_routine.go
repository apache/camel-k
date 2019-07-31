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

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

// NewScheduleRoutineAction creates a new schedule routine action
func NewScheduleRoutineAction() Action {
	return &scheduleRoutineAction{}
}

type scheduleRoutineAction struct {
	baseAction
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
func (action *scheduleRoutineAction) Handle(ctx context.Context, build *v1alpha1.Build) error {
	builds := &v1alpha1.BuildList{}
	options := &k8sclient.ListOptions{Namespace: build.Namespace}
	err := action.client.List(ctx, options, builds)
	if err != nil {
		return err
	}

	// Emulate a serialized working queue to only allow one build to run at a given time.
	// This is currently necessary for the incremental build to work as expected.
	hasScheduledPod := false
	for _, b := range builds.Items {
		if b.Status.Phase == v1alpha1.BuildPhasePending || b.Status.Phase == v1alpha1.BuildPhaseRunning {
			hasScheduledPod = true
		}
	}

	if hasScheduledPod {
		// Let's requeue the build in case one is already running
		return nil
	}

	err = SubmitBuildRequest(ctx, action.client, build, action.L, nil)
	if err != nil {
		return nil
	}

	target := build.DeepCopy()
	target.Status.Phase = v1alpha1.BuildPhasePending
	action.L.Info("Build state transition", "phase", target.Status.Phase)

	return action.client.Status().Update(ctx, target)
}

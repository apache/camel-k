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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/event"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func newScheduleAction(reader ctrl.Reader, buildMonitor Monitor) Action {
	return &scheduleAction{
		reader:       reader,
		buildMonitor: buildMonitor,
	}
}

type scheduleAction struct {
	baseAction
	lock         sync.Mutex
	reader       ctrl.Reader
	buildMonitor Monitor
}

// Name returns a common name of the action.
func (action *scheduleAction) Name() string {
	return "schedule"
}

// CanHandle tells whether this action can handle the build.
func (action *scheduleAction) CanHandle(build *v1.Build) bool {
	return build.Status.Phase == v1.BuildPhaseScheduling
}

// Handle handles the builds.
func (action *scheduleAction) Handle(ctx context.Context, build *v1.Build) (*v1.Build, error) {
	// Enter critical section
	action.lock.Lock()
	defer action.lock.Unlock()

	allowed, schedulingCondition, err := action.buildMonitor.canSchedule(ctx, action.reader, build)

	if err != nil {
		return nil, err
	} else if !allowed {
		// Build not allowed at this state (probably max running builds limit exceeded) - let's requeue the build
		// Update the condition without reseting the Build Status.
		// This must be done in the critical section, rather than delegated to the controller.
		return nil, action.toUpdatedCondition(ctx, build, schedulingCondition)
	}

	// Reset the Build status, and transition it to pending phase.
	// This must be done in the critical section, rather than delegated to the controller.
	return nil, action.toUpdatedStatus(ctx, build, schedulingCondition, v1.BuildPhasePending)
}

func (action *scheduleAction) toUpdatedCondition(ctx context.Context, build *v1.Build, condition *v1.BuildCondition) error {
	return action.patchBuildStatus(ctx, build, func(b *v1.Build) {
		b.Status = v1.BuildStatus{
			Phase:      b.Status.Phase,
			StartedAt:  b.Status.StartedAt,
			Failure:    b.Status.Failure,
			Conditions: b.Status.Conditions,
		}
		b.Status.SetConditions(*condition)
	})

}

func (action *scheduleAction) toUpdatedStatus(ctx context.Context, build *v1.Build, condition *v1.BuildCondition, phase v1.BuildPhase) error {
	err := action.patchBuildStatus(ctx, build, func(b *v1.Build) {
		now := metav1.Now()
		b.Status = v1.BuildStatus{
			Phase:      phase,
			StartedAt:  &now,
			Failure:    b.Status.Failure,
			Conditions: b.Status.Conditions,
		}
		b.Status.SetConditions(*condition)
	})

	if err != nil {
		return err
	}

	monitorRunningBuild(build)

	buildCreator := kubernetes.GetCamelCreator(build)
	// Report the duration the Build has been waiting in the build queue
	observeBuildQueueDuration(build, buildCreator)

	return nil
}

func (action *scheduleAction) patchBuildStatus(ctx context.Context, build *v1.Build, mutate func(b *v1.Build)) error {
	target := build.DeepCopy()
	mutate(target)
	if err := action.client.Status().Patch(ctx, target, ctrl.MergeFrom(build)); err != nil {
		return err
	}

	if target.Status.Phase != build.Status.Phase {
		action.L.Info("State transition", "phase-from", build.Status.Phase, "phase-to", target.Status.Phase)
	}
	event.NotifyBuildUpdated(ctx, action.client, action.recorder, build, target)

	*build = *target
	return nil
}

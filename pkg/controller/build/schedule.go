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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/event"
)

func newScheduleAction(reader ctrl.Reader) Action {
	return &scheduleAction{
		reader: reader,
	}
}

type scheduleAction struct {
	baseAction
	lock   sync.Mutex
	reader ctrl.Reader
}

// Name returns a common name of the action
func (action *scheduleAction) Name() string {
	return "schedule"
}

// CanHandle tells whether this action can handle the build
func (action *scheduleAction) CanHandle(build *v1.Build) bool {
	return build.Status.Phase == v1.BuildPhaseScheduling
}

// Handle handles the builds
func (action *scheduleAction) Handle(ctx context.Context, build *v1.Build) (*v1.Build, error) {
	// Enter critical section
	action.lock.Lock()
	defer action.lock.Unlock()

	layout := build.Labels[v1.IntegrationKitLayoutLabel]

	// Native builds can be run in parallel, as incremental images is not applicable.
	if layout == v1.IntegrationKitLayoutNative {
		// Reset the Build status, and transition it to pending phase.
		// This must be done in the critical section, rather than delegated to the controller.
		return nil, action.toPendingPhase(ctx, build)
	}

	// We assume incremental images is only applicable across images whose layout is identical
	withCompatibleLayout, err := labels.NewRequirement(v1.IntegrationKitLayoutLabel, selection.Equals, []string{layout})
	if err != nil {
		return nil, err
	}

	builds := &v1.BuildList{}
	// We use the non-caching client as informers cache is not invalidated nor updated
	// atomically by write operations
	err = action.reader.List(ctx, builds,
		ctrl.InNamespace(build.Namespace),
		ctrl.MatchingLabelsSelector{
			Selector: labels.NewSelector().Add(*withCompatibleLayout),
		})
	if err != nil {
		return nil, err
	}

	// Emulate a serialized working queue to only allow one build to run at a given time.
	// This is currently necessary for the incremental build to work as expected.
	// We may want to explicitly manage build priority as opposed to relying on
	// the reconciliation loop to handle the queuing.
	for _, b := range builds.Items {
		if b.Status.Phase == v1.BuildPhasePending || b.Status.Phase == v1.BuildPhaseRunning {
			// Let's requeue the build in case one is already running
			return nil, nil
		}
	}

	// Reset the Build status, and transition it to pending phase.
	// This must be done in the critical section, rather than delegated to the controller.
	return nil, action.toPendingPhase(ctx, build)
}

func (action *scheduleAction) toPendingPhase(ctx context.Context, build *v1.Build) error {
	err := action.patchBuildStatus(ctx, build, func(b *v1.Build) {
		now := metav1.Now()
		b.Status = v1.BuildStatus{
			Phase:      v1.BuildPhasePending,
			StartedAt:  &now,
			Failure:    b.Status.Failure,
			Conditions: b.Status.Conditions,
		}
	})
	if err != nil {
		return err
	}

	// Report the duration the Build has been waiting in the build queue
	observeBuildQueueDuration(build)

	return nil
}

func (action *scheduleAction) patchBuildStatus(ctx context.Context, build *v1.Build, mutate func(b *v1.Build)) error {
	target := build.DeepCopy()
	mutate(target)
	if err := action.client.Status().Patch(ctx, target, ctrl.MergeFrom(build)); err != nil {
		return err
	}

	if target.Status.Phase != build.Status.Phase {
		action.L.Info("state transition", "phase-from", build.Status.Phase, "phase-to", target.Status.Phase)
	}
	event.NotifyBuildUpdated(ctx, action.client, action.recorder, build, target)

	*build = *target
	return nil
}

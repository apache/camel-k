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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/event"
	"github.com/apache/camel-k/pkg/util/patch"
)

// NewScheduleRoutineAction creates a new schedule routine action
func NewScheduleRoutineAction(reader client.Reader, b *builder.Builder, r *sync.Map) Action {
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
	builder  *builder.Builder
	routines *sync.Map
}

// Name returns a common name of the action
func (action *scheduleRoutineAction) Name() string {
	return "schedule-routine"
}

// CanHandle tells whether this action can handle the build
func (action *scheduleRoutineAction) CanHandle(build *v1.Build) bool {
	return build.Status.Phase == v1.BuildPhaseScheduling
}

// Handle handles the builds
func (action *scheduleRoutineAction) Handle(ctx context.Context, build *v1.Build) (*v1.Build, error) {
	// Enter critical section
	action.lock.Lock()
	defer action.lock.Unlock()

	builds := &v1.BuildList{}
	// We use the non-caching client as informers cache is not invalidated nor updated
	// atomically by write operations
	err := action.reader.List(ctx, builds, client.InNamespace(build.Namespace))
	if err != nil {
		return nil, err
	}

	// Emulate a serialized working queue to only allow one build to run at a given time.
	// This is currently necessary for the incremental build to work as expected.
	for _, b := range builds.Items {
		if b.Status.Phase == v1.BuildPhasePending || b.Status.Phase == v1.BuildPhaseRunning {
			// Let's requeue the build in case one is already running
			return nil, nil
		}
	}

	// Transition the build to pending state
	// This must be done in the critical section rather than delegated to the controller
	err = action.updateBuildStatus(ctx, build, v1.BuildStatus{Phase: v1.BuildPhasePending})
	if err != nil {
		return nil, err
	}

	// Report the duration the Build has been waiting in the build queue
	queueDuration.Observe(time.Now().Sub(getBuildQueuingTime(build)).Seconds())

	// Start the build asynchronously to avoid blocking the reconcile loop
	action.routines.Store(build.Name, true)

	go action.runBuild(build)

	return nil, nil
}

func (action *scheduleRoutineAction) runBuild(build *v1.Build) {
	defer action.routines.Delete(build.Name)

	ctx := context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, build.Spec.Timeout.Duration)
	defer cancel()

	now := metav1.Now()
	status := v1.BuildStatus{
		Phase:     v1.BuildPhaseRunning,
		StartedAt: &now,
	}
	if err := action.updateBuildStatus(ctx, build, status); err != nil {
		return
	}

	buildDir := ""

	for i, task := range build.Spec.Tasks {
		select {
		case <-ctxWithTimeout.Done():
			status.Phase = v1.BuildPhaseInterrupted
			break

		default:
			// Coordinate the build and context directories across the sequence of tasks
			if t := task.Builder; t != nil {
				if t.BuildDir == "" {
					tmpDir, err := ioutil.TempDir(os.TempDir(), build.Name+"-")
					if err != nil {
						status.Failed(err)
						break
					}
					t.BuildDir = tmpDir
					// Deferring in the for loop is what we want here
					defer os.RemoveAll(tmpDir)
				}
				buildDir = t.BuildDir
			} else if t := task.Spectrum; t != nil && t.ContextDir == "" {
				if buildDir == "" {
					status.Failed(fmt.Errorf("cannot determine context directory for task %s", t.Name))
					break
				}
				t.ContextDir = path.Join(buildDir, builder.ContextDir)
			} else if t := task.S2i; t != nil && t.ContextDir == "" {
				if buildDir == "" {
					status.Failed(fmt.Errorf("cannot determine context directory for task %s", t.Name))
					break
				}
				t.ContextDir = path.Join(buildDir, builder.ContextDir)
			}

			// Execute the task
			status = action.builder.Build(build).Task(task).Do(ctxWithTimeout)

			lastTask := i == len(build.Spec.Tasks)-1
			taskFailed := status.Phase == v1.BuildPhaseFailed || status.Phase == v1.BuildPhaseError
			if lastTask && !taskFailed {
				status.Phase = v1.BuildPhaseSucceeded
			}

			if lastTask || taskFailed {
				// Spare a redundant update
				break
			}

			// Update the Build status
			err := action.updateBuildStatus(ctx, build, status)
			if err != nil {
				status.Failed(err)
				break
			}
		}
	}

	duration := metav1.Now().Sub(build.Status.StartedAt.Time)
	status.Duration = duration.String()
	// Account for the Build metrics
	observeBuildResult(build, status.Phase, duration)

	_ = action.updateBuildStatus(ctx, build, status)
}

func (action *scheduleRoutineAction) updateBuildStatus(ctx context.Context, build *v1.Build, status v1.BuildStatus) error {
	target := build.DeepCopy()
	target.Status = status
	// Copy the failure field from the build to persist recovery state
	target.Status.Failure = build.Status.Failure
	// Patch the build status with the result
	p, err := patch.PositiveMergePatch(build, target)
	if err != nil {
		action.L.Errorf(err, "Cannot patch build status: %s", build.Name)
		return err
	}
	err = action.client.Status().Patch(ctx, target, client.RawPatch(types.MergePatchType, p))
	if err != nil {
		action.L.Errorf(err, "Cannot update build status: %s", build.Name)
		return err
	}
	if target.Status.Phase != build.Status.Phase {
		action.L.Info("Build state transition", "phase", target.Status.Phase)
	}
	event.NotifyBuildUpdated(ctx, action.client, action.recorder, build, target)
	build.Status = target.Status
	return nil
}

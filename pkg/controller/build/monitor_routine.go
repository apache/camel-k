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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/event"
	"github.com/apache/camel-k/pkg/util/patch"
)

var routines sync.Map

func newMonitorRoutineAction() Action {
	return &monitorRoutineAction{}
}

type monitorRoutineAction struct {
	baseAction
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
	switch build.Status.Phase {

	case v1.BuildPhasePending:
		if _, ok := routines.Load(build.Name); ok {
			// Something went wrong. Let's fail the Build to start over a clean state.
			routines.Delete(build.Name)
			build.Status.Phase = v1.BuildPhaseFailed
			build.Status.Error = "Build routine exists"
			return build, nil
		}
		status := v1.BuildStatus{Phase: v1.BuildPhaseRunning}
		if err := action.updateBuildStatus(ctx, build, status); err != nil {
			return nil, err
		}
		// Start the build asynchronously to avoid blocking the reconciliation loop
		routines.Store(build.Name, true)
		go action.runBuild(build)

	case v1.BuildPhaseRunning:
		if _, ok := routines.Load(build.Name); !ok {
			// Recover the build if the routine missing. This can happen when the operator
			// stops abruptly and restarts or the build status update fails.
			build.Status.Phase = v1.BuildPhaseFailed
			build.Status.Error = "Build routine not running"
			return build, nil
		}
	}

	return nil, nil
}

func (action *monitorRoutineAction) runBuild(build *v1.Build) {
	defer routines.Delete(build.Name)

	ctx := context.Background()
	ctxWithTimeout, cancel := context.WithDeadline(ctx, build.Status.StartedAt.Add(build.Spec.Timeout.Duration))
	defer cancel()

	status := v1.BuildStatus{}
	buildDir := ""
	Builder := builder.New(action.client)

tasks:
	for i, task := range build.Spec.Tasks {
		select {
		case <-ctxWithTimeout.Done():
			if errors.Is(ctxWithTimeout.Err(), context.Canceled) {
				// Context canceled
				status.Phase = v1.BuildPhaseInterrupted
			} else {
				// Context timeout
				status.Phase = v1.BuildPhaseFailed
			}
			status.Error = ctxWithTimeout.Err().Error()

			break tasks

		default:
			// Coordinate the build and context directories across the sequence of tasks
			if t := task.Builder; t != nil {
				if t.BuildDir == "" {
					tmpDir, err := ioutil.TempDir(os.TempDir(), build.Name+"-")
					if err != nil {
						status.Failed(err)

						break tasks
					}
					t.BuildDir = tmpDir
					// Deferring in the for loop is what we want here
					defer os.RemoveAll(tmpDir)
				}
				buildDir = t.BuildDir
			} else if t := task.Spectrum; t != nil && t.ContextDir == "" {
				if buildDir == "" {
					status.Failed(fmt.Errorf("cannot determine context directory for task %s", t.Name))
					break tasks
				}
				t.ContextDir = path.Join(buildDir, builder.ContextDir)
			} else if t := task.S2i; t != nil && t.ContextDir == "" {
				if buildDir == "" {
					status.Failed(fmt.Errorf("cannot determine context directory for task %s", t.Name))
					break tasks
				}
				t.ContextDir = path.Join(buildDir, builder.ContextDir)
			}

			// Execute the task
			status = Builder.Build(build).Task(task).Do(ctxWithTimeout)

			lastTask := i == len(build.Spec.Tasks)-1
			taskFailed := status.Phase == v1.BuildPhaseFailed ||
				status.Phase == v1.BuildPhaseError ||
				status.Phase == v1.BuildPhaseInterrupted
			if lastTask && !taskFailed {
				status.Phase = v1.BuildPhaseSucceeded
			}

			if lastTask || taskFailed {
				// Spare a redundant update
				break tasks
			}

			// Update the Build status
			err := action.updateBuildStatus(ctx, build, status)
			if err != nil {
				status.Failed(err)
				break tasks
			}
		}
	}

	duration := metav1.Now().Sub(build.Status.StartedAt.Time)
	status.Duration = duration.String()
	// Account for the Build metrics
	observeBuildResult(build, status.Phase, duration)

	_ = action.updateBuildStatus(ctx, build, status)
}

func (action *monitorRoutineAction) updateBuildStatus(ctx context.Context, build *v1.Build, status v1.BuildStatus) error {
	target := build.DeepCopy()
	target.Status = status
	// Copy the failure field from the build to persist recovery state
	target.Status.Failure = build.Status.Failure
	// Patch the build status with the result
	p, err := patch.PositiveMergePatch(build, target)
	if err != nil {
		action.L.Errorf(err, "Cannot patch build status: %s", build.Name)
		event.NotifyBuildError(ctx, action.client, action.recorder, build, target, err)
		return err
	}
	if target.Status.Phase == v1.BuildPhaseFailed {
		action.L.Errorf(nil, "Build %s failed: %s", build.Name, target.Status.Error)
	} else if target.Status.Phase == v1.BuildPhaseError {
		action.L.Errorf(nil, "Build %s errored: %s", build.Name, target.Status.Error)
	}
	err = action.client.Status().Patch(ctx, target, ctrl.RawPatch(types.MergePatchType, p))
	if err != nil {
		action.L.Errorf(err, "Cannot update build status: %s", build.Name)
		event.NotifyBuildError(ctx, action.client, action.recorder, build, target, err)
		return err
	}
	if target.Status.Phase != build.Status.Phase {
		action.L.Info("state transition", "phase-from", build.Status.Phase, "phase-to", target.Status.Phase)
	}
	event.NotifyBuildUpdated(ctx, action.client, action.recorder, build, target)
	build.Status = target.Status
	return nil
}

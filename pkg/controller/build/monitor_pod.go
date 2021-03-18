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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// NewMonitorPodAction creates a new monitor action for scheduled pod
func NewMonitorPodAction() Action {
	return &monitorPodAction{}
}

type monitorPodAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *monitorPodAction) Name() string {
	return "monitor-pod"
}

// CanHandle tells whether this action can handle the build
func (action *monitorPodAction) CanHandle(build *v1.Build) bool {
	return build.Status.Phase == v1.BuildPhasePending || build.Status.Phase == v1.BuildPhaseRunning
}

// Handle handles the builds
func (action *monitorPodAction) Handle(ctx context.Context, build *v1.Build) (*v1.Build, error) {
	pod, err := getBuilderPod(ctx, action.client, build)
	if err != nil {
		return nil, err
	}

	switch {
	case pod == nil:
		build.Status.Phase = v1.BuildPhaseScheduling

	// Pod remains in pending phase when init containers execute
	case pod.Status.Phase == corev1.PodPending && action.isPodScheduled(pod),
		pod.Status.Phase == corev1.PodRunning:
		build.Status.Phase = v1.BuildPhaseRunning
		if build.Status.StartedAt == nil || build.Status.StartedAt.Time.IsZero() {
			now := metav1.Now()
			build.Status.StartedAt = &now
		}

	case pod.Status.Phase == corev1.PodSucceeded:
		build.Status.Phase = v1.BuildPhaseSucceeded
		duration := metav1.Now().Sub(build.Status.StartedAt.Time)
		build.Status.Duration = duration.String()

		// Account for the Build metrics
		observeBuildResult(build, build.Status.Phase, duration)

		for _, task := range build.Spec.Tasks {
			if t := task.Buildah; t != nil {
				build.Status.Image = t.Image
				break
			} else if t := task.Kaniko; t != nil {
				build.Status.Image = t.Image
				break
			}
		}
		// Reconcile image digest from build container status if available
		for _, container := range pod.Status.ContainerStatuses {
			if container.Name == "buildah" {
				build.Status.Digest = container.State.Terminated.Message
				break
			}
		}

	case pod.Status.Phase == corev1.PodFailed:
		phase := v1.BuildPhaseFailed
		// Do not override errored build
		if build.Status.Phase == v1.BuildPhaseError {
			phase = v1.BuildPhaseError
		}
		build.Status.Phase = phase
		duration := metav1.Now().Sub(build.Status.StartedAt.Time)
		build.Status.Duration = duration.String()

		// Account for the Build metrics
		observeBuildResult(build, build.Status.Phase, duration)
	}

	return build, nil
}

func (action *monitorPodAction) isPodScheduled(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

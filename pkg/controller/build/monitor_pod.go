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
	"encoding/json"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

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
		// Emulate context cancellation
		build.Status.Phase = v1.BuildPhaseInterrupted
		build.Status.Error = "Pod deleted"

	case pod.Status.Phase == corev1.PodRunning,
		// Pod remains in pending phase when init containers execute
		pod.Status.Phase == corev1.PodPending:
		if action.isPodScheduled(pod) {
			build.Status.Phase = v1.BuildPhaseRunning
		}
		if time.Now().Sub(build.Status.StartedAt.Time) > build.Spec.Timeout.Duration {
			// Send SIGTERM signal to running containers
			err := action.sigterm(pod)
			if err != nil {
				// Requeue
				return nil, err
			}
		}

	case pod.Status.Phase == corev1.PodSucceeded:
		build.Status.Phase = v1.BuildPhaseSucceeded
		finishedAt := action.getTerminatedTime(pod)
		duration := finishedAt.Sub(build.Status.StartedAt.Time)
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
		message := "Pod failed"
		if terminationMessage := action.getTerminationMessage(pod); terminationMessage != "" {
			message = terminationMessage
		}
		if pod.DeletionTimestamp != nil {
			phase = v1.BuildPhaseInterrupted
			message = "Pod deleted"
		}
		// Do not override errored build
		if build.Status.Phase == v1.BuildPhaseError {
			phase = v1.BuildPhaseError
		}
		build.Status.Phase = phase
		build.Status.Error = message
		finishedAt := action.getTerminatedTime(pod)
		duration := finishedAt.Sub(build.Status.StartedAt.Time)
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

func (action *monitorPodAction) sigterm(pod *corev1.Pod) error {
	var containers []corev1.ContainerStatus
	containers = append(containers, pod.Status.InitContainerStatuses...)
	containers = append(containers, pod.Status.ContainerStatuses...)

	for _, container := range containers {
		if container.State.Running == nil {
			continue
		}

		r := action.client.CoreV1().RESTClient().Post().
			Resource("pods").
			Namespace(pod.Namespace).
			Name(pod.Name).
			SubResource("exec").
			Param("container", container.Name)

		r.VersionedParams(&corev1.PodExecOptions{
			Container: container.Name,
			Command:   []string{"kill", "-SIGTERM", "1"},
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

		exec, err := remotecommand.NewSPDYExecutor(action.client.GetConfig(), "POST", r.URL())
		if err != nil {
			return err
		}

		err = exec.Stream(remotecommand.StreamOptions{
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			Tty:    false,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (action *monitorPodAction) getTerminatedTime(pod *corev1.Pod) metav1.Time {
	var finishedAt metav1.Time

	var containers []corev1.ContainerStatus
	containers = append(containers, pod.Status.InitContainerStatuses...)
	containers = append(containers, pod.Status.ContainerStatuses...)

	for _, container := range containers {
		if container.State.Terminated == nil {
			// The container has not run
			continue
		}
		if t := container.State.Terminated.FinishedAt; finishedAt.IsZero() || t.After(finishedAt.Time) {
			finishedAt = t
		}
	}

	return finishedAt
}

func (action *monitorPodAction) getTerminationMessage(pod *corev1.Pod) string {
	var terminationMessages []terminationMessage

	var containers []corev1.ContainerStatus
	containers = append(containers, pod.Status.InitContainerStatuses...)
	containers = append(containers, pod.Status.ContainerStatuses...)

	for _, container := range containers {
		if t := container.State.Terminated; t != nil && t.ExitCode != 0 && t.Message != "" {
			terminationMessages = append(terminationMessages, terminationMessage{
				Container: container.Name,
				Message:   t.Message,
			})
		}
	}

	switch len(terminationMessages) {
	case 0:
		return ""
	case 1:
		return terminationMessages[0].Message
	default:
		message, err := json.Marshal(terminationMessages)
		if err != nil {
			return ""
		}
		return string(message)
	}
}

type terminationMessage struct {
	Container string `json:"container,omitempty"`
	Message   string `json:"message,omitempty"`
}

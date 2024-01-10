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
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes/log"
)

const timeoutAnnotation = "camel.apache.org/timeout"

func newMonitorPodAction(reader ctrl.Reader) Action {
	return &monitorPodAction{
		reader: reader,
	}
}

type monitorPodAction struct {
	baseAction
	reader ctrl.Reader
}

// Name returns a common name of the action.
func (action *monitorPodAction) Name() string {
	return "monitor-pod"
}

// CanHandle tells whether this action can handle the build.
func (action *monitorPodAction) CanHandle(build *v1.Build) bool {
	return build.Status.Phase == v1.BuildPhasePending || build.Status.Phase == v1.BuildPhaseRunning
}

// Handle handles the builds.
func (action *monitorPodAction) Handle(ctx context.Context, build *v1.Build) (*v1.Build, error) {
	pod, err := getBuilderPod(ctx, action.reader, build)
	if err != nil {
		return nil, err
	}

	if pod == nil {
		switch build.Status.Phase {

		case v1.BuildPhasePending:
			if pod, err = newBuildPod(ctx, action.reader, action.client, build); err != nil {
				return nil, err
			}

			// If the Builder Pod is in the Build namespace, we can set the ownership to it. If not (global operator mode)
			// we set the ownership to the Operator Pod instead
			var owner metav1.Object
			owner = build
			if build.Namespace != pod.Namespace {
				operatorPod := platform.GetOperatorPod(ctx, action.reader, pod.Namespace)
				if operatorPod != nil {
					owner = operatorPod
				}
			}
			if err = controllerutil.SetControllerReference(owner, pod, action.client.GetScheme()); err != nil {
				return nil, err
			}

			if err = action.client.Create(ctx, pod); err != nil {
				return nil, fmt.Errorf("cannot create build pod: %w", err)
			}

		case v1.BuildPhaseRunning:
			// Emulate context cancellation
			build.Status.Phase = v1.BuildPhaseInterrupted
			build.Status.Error = "Pod deleted"
			monitorFinishedBuild(build)
			return build, nil
		}
	}

	switch pod.Status.Phase {

	case corev1.PodPending, corev1.PodRunning:
		// Pod remains in pending phase when init containers execute
		if action.isPodScheduled(pod) {
			build.Status.Phase = v1.BuildPhaseRunning
		}
		if time.Since(build.Status.StartedAt.Time) > build.Spec.Timeout.Duration {
			// Patch the Pod with an annotation, to identify termination signal
			// has been sent because the Build has timed out
			if err = action.addTimeoutAnnotation(ctx, pod, metav1.Now()); err != nil {
				return nil, err
			}
			// Send SIGTERM signal to running containers
			if err = action.sigterm(ctx, pod); err != nil {
				// Requeue
				return nil, err
			}

			monitorFinishedBuild(build)
		} else {
			// Monitor running state of the build - this may have been done already by the schedule action but the build monitor is idempotent
			// We do this here to potentially restore the running build state in the monitor in case of an operator restart
			monitorRunningBuild(build)
		}

	case corev1.PodSucceeded:
		build.Status.Phase = v1.BuildPhaseSucceeded
		// Remove the annotation in case the Build succeeded, between
		// the timeout deadline and the termination signal.
		if err = action.removeTimeoutAnnotation(ctx, pod); err != nil {
			return nil, err
		}
		finishedAt := action.getTerminatedTime(pod)
		duration := finishedAt.Sub(build.Status.StartedAt.Time)
		build.Status.Duration = duration.String()
		action.setConditionsFromTerminationMessages(ctx, pod, &build.Status)
		monitorFinishedBuild(build)

		buildCreator := kubernetes.GetCamelCreator(build)
		// Account for the Build metrics
		observeBuildResult(build, build.Status.Phase, buildCreator, duration)

		build.Status.Image = publishTaskImageName(build.Spec.Tasks)
		build.Status.Digest = publishTaskDigest(build.Spec.Tasks, pod.Status.ContainerStatuses)
		if build.Status.Digest == "" {
			// Likely to happen for users provided publishing tasks and not providing the digest image among statuses
			build.Status.Phase = v1.BuildPhaseError
			build.Status.SetCondition(
				"ImageDigestAvailable",
				corev1.ConditionFalse,
				"ImageDigestAvailable",
				fmt.Sprintf(
					"%s publishing task completed but no digest is available in container status. Make sure that the process successfully push the image to the registry and write its digest to /dev/termination-log",
					publishTaskName(build.Spec.Tasks),
				),
			)
		}

	case corev1.PodFailed:
		phase := v1.BuildPhaseFailed
		message := fmt.Sprintf("Builder Pod %s failed (see conditions for more details)", pod.Name)
		if pod.DeletionTimestamp != nil {
			phase = v1.BuildPhaseInterrupted
			message = fmt.Sprintf("Builder Pod %s deleted", pod.Name)
		} else if _, ok := pod.GetAnnotations()[timeoutAnnotation]; ok {
			message = fmt.Sprintf("Builder Pod %s timeout", pod.Name)
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
		action.setConditionsFromTerminationMessages(ctx, pod, &build.Status)
		monitorFinishedBuild(build)

		buildCreator := kubernetes.GetCamelCreator(build)
		// Account for the Build metrics
		observeBuildResult(build, build.Status.Phase, buildCreator, duration)
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

func (action *monitorPodAction) sigterm(ctx context.Context, pod *corev1.Pod) error {
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
			Command:   []string{"/bin/bash", "-c", "kill -SIGTERM 1"},
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

		exec, err := remotecommand.NewSPDYExecutor(action.client.GetConfig(), "POST", r.URL())
		if err != nil {
			return err
		}

		err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
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

func (action *monitorPodAction) addTimeoutAnnotation(ctx context.Context, pod *corev1.Pod, time metav1.Time) error {
	if _, ok := pod.GetAnnotations()[timeoutAnnotation]; ok {
		return nil
	}
	return action.patchPod(ctx, pod, func(p *corev1.Pod) {
		if p.GetAnnotations() != nil {
			p.GetAnnotations()[timeoutAnnotation] = time.String()
		} else {
			p.SetAnnotations(map[string]string{
				timeoutAnnotation: time.String(),
			})
		}
	})
}

func (action *monitorPodAction) removeTimeoutAnnotation(ctx context.Context, pod *corev1.Pod) error {
	if _, ok := pod.GetAnnotations()[timeoutAnnotation]; !ok {
		return nil
	}
	return action.patchPod(ctx, pod, func(p *corev1.Pod) {
		delete(p.GetAnnotations(), timeoutAnnotation)
	})
}

func (action *monitorPodAction) patchPod(ctx context.Context, pod *corev1.Pod, mutate func(*corev1.Pod)) error {
	target := pod.DeepCopy()
	mutate(target)
	if err := action.client.Patch(ctx, target, ctrl.MergeFrom(pod)); err != nil {
		return err
	}
	*pod = *target
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

// setConditionsFromTerminationMessages sets a condition for all those containers which have been terminated (successfully or not).
func (action *monitorPodAction) setConditionsFromTerminationMessages(ctx context.Context, pod *corev1.Pod, buildStatus *v1.BuildStatus) {
	var containers []corev1.ContainerStatus
	containers = append(containers, pod.Status.InitContainerStatuses...)
	containers = append(containers, pod.Status.ContainerStatuses...)

	for _, container := range containers {
		if t := container.State.Terminated; t != nil {
			var err error
			terminationMessage := t.Message
			// Dynamic condition type (it depends on each container name)
			containerConditionType := v1.BuildConditionType(fmt.Sprintf("Container %s succeeded", container.Name))
			containerSucceeded := corev1.ConditionTrue
			if t.ExitCode != 0 {
				containerSucceeded = corev1.ConditionFalse
			}
			if terminationMessage == "" {
				// TODO we can make it a user variable !?
				var maxLines int64 = 10
				logOptions := corev1.PodLogOptions{
					Container: container.Name,
					TailLines: &maxLines,
				}
				terminationMessage, err = log.DumpLog(ctx, action.client, pod, logOptions)
				if err != nil {
					action.L.Errorf(err, "Dumping log for %s container in %s Pod failed", container.Name, pod.Name)
					terminationMessage = fmt.Sprintf(
						"Operator was not able to retrieve the error message, please, check the container %s log directly from %s Pod",
						container.Name,
						pod.Name,
					)
				}
			}

			terminationReason := fmt.Sprintf("%s (%d)", t.Reason, t.ExitCode)
			buildStatus.SetCondition(containerConditionType, containerSucceeded, terminationReason, terminationMessage)
		}
	}

}

// we expect that the last task is any of the supported publishing task
// or a custom user task
func publishTask(tasks []v1.Task) *v1.Task {
	if len(tasks) > 0 {
		return &tasks[len(tasks)-1]

	}

	return nil
}

func publishTaskImageName(tasks []v1.Task) string {
	t := publishTask(tasks)
	if t == nil {
		return ""
	}
	if t.Custom != nil {
		return t.Custom.PublishingImage
	} else if t.Spectrum != nil {
		return t.Spectrum.Image
	} else if t.Jib != nil {
		return t.Jib.Image
	} else if t.Buildah != nil {
		return t.Buildah.Image
	} else if t.Kaniko != nil {
		return t.Kaniko.Image
	}

	return ""
}

func publishTaskName(tasks []v1.Task) string {
	t := publishTask(tasks)
	if t == nil {
		return ""
	}
	if t.Custom != nil {
		return t.Custom.Name
	} else if t.Spectrum != nil {
		return t.Spectrum.Name
	} else if t.Jib != nil {
		return t.Jib.Name
	} else if t.Buildah != nil {
		return t.Buildah.Name
	} else if t.Kaniko != nil {
		return t.Kaniko.Name
	}

	return ""
}

func publishTaskDigest(tasks []v1.Task, cntStates []corev1.ContainerStatus) string {
	taskName := publishTaskName(tasks)
	// Reconcile image digest from build container status if available
	for _, container := range cntStates {
		if container.Name == taskName {
			return container.State.Terminated.Message
		}
	}
	return ""
}

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
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

// NewSchedulePodAction creates a new schedule action
func NewSchedulePodAction(reader client.Reader) Action {
	return &schedulePodAction{
		reader: reader,
	}
}

type schedulePodAction struct {
	baseAction
	lock          sync.Mutex
	reader        client.Reader
	operatorImage string
}

// Name returns a common name of the action
func (action *schedulePodAction) Name() string {
	return "schedule-pod"
}

// CanHandle tells whether this action can handle the build
func (action *schedulePodAction) CanHandle(build *v1.Build) bool {
	return build.Status.Phase == v1.BuildPhaseScheduling
}

// Handle handles the builds
func (action *schedulePodAction) Handle(ctx context.Context, build *v1.Build) (*v1.Build, error) {
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

	pod, err := getBuilderPod(ctx, action.client, build)
	if err != nil {
		return nil, err
	}

	if pod == nil {
		// We may want to explicitly manage build priority as opposed to relying on
		// the reconcile loop to handle the queuing
		pod, err = action.newBuildPod(ctx, build)
		if err != nil {
			return nil, err
		}

		// Set the Build instance as the owner and controller
		if err := controllerutil.SetControllerReference(build, pod, action.client.GetScheme()); err != nil {
			return nil, err
		}

		if err := action.client.Create(ctx, pod); err != nil {
			return nil, errors.Wrap(err, "cannot create build pod")
		}

		// Report the duration the Build has been waiting in the build queue
		queueDuration.Observe(time.Now().Sub(getBuildQueuingTime(build)).Seconds())
	}

	build.Status.Phase = v1.BuildPhasePending

	return build, nil
}

func (action *schedulePodAction) newBuildPod(ctx context.Context, build *v1.Build) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: build.Namespace,
			Name:      buildPodName(build),
			Labels: map[string]string{
				"camel.apache.org/build":     build.Name,
				"camel.apache.org/component": "builder",
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: platform.BuilderServiceAccount,
			RestartPolicy:      corev1.RestartPolicyNever,
		},
	}

	pod.Labels = kubernetes.MergeCamelCreatorLabels(build.Labels, pod.Labels)

	for _, task := range build.Spec.Tasks {
		if task.Builder != nil {
			// TODO: Move the retrieval of the operator image into the controller
			operatorImage, err := platform.GetCurrentOperatorImage(ctx, action.client)
			if err != nil {
				return nil, err
			}
			if operatorImage == "" {
				action.operatorImage = defaults.ImageName + ":" + defaults.Version
			} else {
				action.operatorImage = operatorImage
			}
			action.addBuilderTaskToPod(build, task.Builder, pod)
		} else if task.Image != nil {
			action.addImageTaskToPod(task.Image, pod)
		}
	}

	// Make sure there is one container defined
	pod.Spec.Containers = pod.Spec.InitContainers[len(pod.Spec.InitContainers)-1 : len(pod.Spec.InitContainers)]
	pod.Spec.InitContainers = pod.Spec.InitContainers[:len(pod.Spec.InitContainers)-1]

	return pod, nil
}

func (action *schedulePodAction) addBuilderTaskToPod(build *v1.Build, task *v1.BuilderTask, pod *corev1.Pod) {
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
		Name:            task.Name,
		Image:           action.operatorImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"kamel",
			"builder",
			"--namespace",
			pod.Namespace,
			"--build-name",
			build.Name,
			"--task-name",
			task.Name,
		},
		VolumeMounts: task.VolumeMounts,
	})

	action.addBaseTaskToPod(&task.BaseTask, pod)
}

func (action *schedulePodAction) addImageTaskToPod(task *v1.ImageTask, pod *corev1.Pod) {
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
		Name:            task.Name,
		Image:           task.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         task.Container.Command,
		Args:            task.Container.Args,
		Env:             task.Container.Env,
		WorkingDir:      task.Container.WorkingDir,
		VolumeMounts:    task.VolumeMounts,
		SecurityContext: task.Container.SecurityContext,
	})

	action.addBaseTaskToPod(&task.BaseTask, pod)
}

func (action *schedulePodAction) addBaseTaskToPod(task *v1.BaseTask, pod *corev1.Pod) {
	pod.Spec.Volumes = append(pod.Spec.Volumes, task.Volumes...)

	if task.Affinity != nil {
		// We may want to handle possible conflicts
		pod.Spec.Affinity = task.Affinity
	}
}

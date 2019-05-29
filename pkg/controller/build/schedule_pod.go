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

	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/defaults"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

// NewSchedulePodAction creates a new schedule action
func NewSchedulePodAction(reader k8sclient.Reader) Action {
	return &schedulePodAction{
		reader: reader,
	}
}

type schedulePodAction struct {
	baseAction
	lock   sync.Mutex
	reader k8sclient.Reader
}

// Name returns a common name of the action
func (action *schedulePodAction) Name() string {
	return "schedule-pod"
}

// CanHandle tells whether this action can handle the build
func (action *schedulePodAction) CanHandle(build *v1alpha1.Build) bool {
	return build.Status.Phase == v1alpha1.BuildPhaseScheduling &&
		build.Spec.Platform.Build.BuildStrategy == v1alpha1.IntegrationPlatformBuildStrategyPod
}

// Handle handles the builds
func (action *schedulePodAction) Handle(ctx context.Context, build *v1alpha1.Build) error {
	// Enter critical section
	action.lock.Lock()
	defer action.lock.Unlock()

	builds := &v1alpha1.BuildList{}
	options := &k8sclient.ListOptions{Namespace: build.Namespace}
	// We use the non-caching client as informers cache is not invalidated nor updated
	// atomically by write operations
	err := action.reader.List(ctx, options, builds)
	if err != nil {
		return err
	}

	// Emulate a serialized working queue to only allow one build to run at a given time.
	// This is currently necessary for the incremental build to work as expected.
	hasScheduledBuild := false
	for _, b := range builds.Items {
		if b.Status.Phase == v1alpha1.BuildPhasePending || b.Status.Phase == v1alpha1.BuildPhaseRunning {
			hasScheduledBuild = true
			break
		}
	}

	if hasScheduledBuild {
		// Let's requeue the build in case one is already running
		return nil
	}

	// Try to get operator image name before starting the build
	operatorImage, err := platform.GetCurrentOperatorImage(ctx, action.client)
	if err != nil {
		return err
	}

	// Otherwise, let's create the build pod
	// We may want to explicitly manage build priority as opposed to relying on
	// the reconcile loop to handle the queuing
	pod := newBuildPod(build, operatorImage)

	// Set the Build instance as the owner and controller
	if err := controllerutil.SetControllerReference(build, pod, action.client.GetScheme()); err != nil {
		return err
	}

	err = action.client.Delete(ctx, pod)
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrap(err, "cannot delete build pod")
	}

	err = action.client.Create(ctx, pod)
	if err != nil {
		return errors.Wrap(err, "cannot create build pod")
	}

	target := build.DeepCopy()
	target.Status.Phase = v1alpha1.BuildPhasePending
	action.L.Info("Build state transition", "phase", target.Status.Phase)

	return action.client.Status().Update(ctx, target)
}

func newBuildPod(build *v1alpha1.Build, operatorImage string) *corev1.Pod {
	builderImage := operatorImage
	if builderImage == "" {
		builderImage = defaults.ImageName + ":" + defaults.Version
	}
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: build.Namespace,
			Name:      buildPodName(build.Spec.Meta),
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "camel-k-operator",
			Containers: []corev1.Container{
				{
					Name:  "builder",
					Image: builderImage,
					ImagePullPolicy: "IfNotPresent",
					Args: []string{
						"camel-k-builder",
						build.Namespace,
						build.Name,
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	if build.Spec.Platform.Build.PublishStrategy == v1alpha1.IntegrationPlatformBuildPublishStrategyKaniko {
		// Mount persistent volume used to coordinate build output with Kaniko cache and image build input
		pod.Spec.Volumes = []corev1.Volume{
			{
				Name: "camel-k-builder",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: build.Spec.Platform.Build.PersistentVolumeClaim,
					},
				},
			},
		}
		pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "camel-k-builder",
				MountPath: build.Spec.BuildDir,
			},
		}
		// Co-locate with the builder pod for sharing the host path volume as the current
		// persistent volume claim uses the default storage class which is likely relying
		// on the host path provisioner.
		pod.Spec.Affinity = &corev1.Affinity{
			PodAffinity: &corev1.PodAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"camel.apache.org/component": "operator",
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		}
	}

	return pod
}

func buildPodName(object metav1.ObjectMeta) string {
	return "camel-k-" + object.Name + "-builder"
}

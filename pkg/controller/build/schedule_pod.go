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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/defaults"

	"github.com/pkg/errors"
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
func (action *schedulePodAction) Handle(ctx context.Context, build *v1alpha1.Build) (*v1alpha1.Build, error) {
	// Enter critical section
	action.lock.Lock()
	defer action.lock.Unlock()

	builds := &v1alpha1.BuildList{}
	// We use the non-caching client as informers cache is not invalidated nor updated
	// atomically by write operations
	err := action.reader.List(ctx, builds, k8sclient.InNamespace(build.Namespace))
	if err != nil {
		return nil, err
	}

	// Emulate a serialized working queue to only allow one build to run at a given time.
	// This is currently necessary for the incremental build to work as expected.
	for _, b := range builds.Items {
		if b.Status.Phase == v1alpha1.BuildPhasePending || b.Status.Phase == v1alpha1.BuildPhaseRunning {
			// Let's requeue the build in case one is already running
			return nil, nil
		}
	}

	pod, err := getBuilderPod(ctx, action.client, build)
	if err != nil {
		return nil, err
	}

	if pod == nil {
		// Try to get operator image name before starting the build
		operatorImage, err := platform.GetCurrentOperatorImage(ctx, action.client)
		if err != nil {
			return nil, err
		}

		// We may want to explicitly manage build priority as opposed to relying on
		// the reconcile loop to handle the queuing
		pod, err = action.newBuildPod(ctx, build, operatorImage)
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
	}

	build.Status.Phase = v1alpha1.BuildPhasePending

	return build, nil
}

func (action *schedulePodAction) newBuildPod(ctx context.Context, build *v1alpha1.Build, operatorImage string) (*corev1.Pod, error) {
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
			Labels: map[string]string{
				"camel.apache.org/build":     build.Name,
				"camel.apache.org/component": "builder",
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "camel-k-builder",
			Containers: []corev1.Container{
				{
					Name:            "builder",
					Image:           builderImage,
					ImagePullPolicy: "IfNotPresent",
					Command: []string{
						"kamel",
						"builder",
						"--namespace",
						build.Namespace,
						"--build-name",
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

		// In case the kaniko cache has not run, the /workspace dir needs to have the right permissions set
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:            "prepare-kaniko-workspace",
			Image:           "busybox",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/bin/sh", "-c"},
			Args:            []string{"chmod -R a+rwx /workspace"},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "camel-k-builder",
					MountPath: "/workspace",
				},
			},
		})

		// Use affinity when Kaniko cache warming is enabled
		if build.Spec.Platform.Build.IsKanikoCacheEnabled() {
			// Co-locate with the Kaniko warmer pod for sharing the host path volume as the current
			// persistent volume claim uses the default storage class which is likely relying
			// on the host path provisioner.
			// This has to be done manually by retrieving the Kaniko warmer pod node name and using
			// node affinity as pod affinity only works for running pods and the Kaniko warmer pod
			// has already completed at that stage.

			// Locate the kaniko warmer pod
			pods := &corev1.PodList{}
			err := action.client.List(ctx, pods,
				k8sclient.InNamespace(build.Namespace),
				k8sclient.MatchingLabels{
					"camel.apache.org/component": "kaniko-warmer",
				})
			if err != nil {
				return nil, err
			}

			if len(pods.Items) != 1 {
				return nil, errors.New("failed to locate the Kaniko cache warmer pod")
			}

			// Use node affinity with the Kaniko warmer pod node name
			pod.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "kubernetes.io/hostname",
										Operator: "In",
										Values:   []string{pods.Items[0].Spec.NodeName},
									},
								},
							},
						},
					},
				},
			}
		}
	}

	return pod, nil
}

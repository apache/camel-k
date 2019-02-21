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

package integrationplatform

import (
	"context"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes"

	"github.com/pkg/errors"
)

func warmKanikoCache(ctx context.Context, client client.Client, platform *v1alpha1.IntegrationPlatform) error {
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: platform.Namespace,
			Name:      platform.Name + "-cache",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "warm-kaniko-cache",
					Image: "gcr.io/kaniko-project/warmer:v0.9.0",
					Args: []string{
						"--cache-dir=/workspace/cache",
						"--image=" + platform.Spec.Build.BaseImage,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "camel-k-builder",
							MountPath: "/workspace",
						},
					},
				},
			},
			// Create the cache directory otherwise Kaniko warmer skips caching
			InitContainers: []corev1.Container{
				{
					Name:            "create-kaniko-cache",
					Image:           "busybox",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"mkdir",
						"-p",
						"/workspace/cache",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "camel-k-builder",
							MountPath: "/workspace",
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				{
					Name: "camel-k-builder",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "camel-k-builder",
						},
					},
				},
			},
		},
	}

	// Co-locate with the builder pod for sharing the host path volume as the current
	// persistent volume claim uses the default storage class which is likely relying
	// on the host path provisioner.
	// Note that pod affinity is skipped when not running in-cluster to be able to run
	// the operator in development mode.
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) > 0 && len(port) > 0 {
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

	err := client.Delete(ctx, &pod)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "cannot delete Kaniko warmer pod")
	}

	err = client.Create(ctx, &pod)
	if err != nil {
		return errors.Wrap(err, "cannot create Kaniko warmer pod")
	}

	err = kubernetes.WaitCondition(ctx, client, &pod, func(obj interface{}) (bool, error) {
		if val, ok := obj.(*corev1.Pod); ok {
			if val.Status.Phase == corev1.PodSucceeded {
				return true, nil
			} else if val.Status.Phase == corev1.PodFailed {
				return false, errors.New("Kaniko cache warming failed")
			}
		}
		return false, nil
	}, 10*time.Minute)

	if err != nil {
		return err
	}

	return nil
}

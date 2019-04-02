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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder/kaniko"
	"github.com/apache/camel-k/pkg/client"

	"github.com/pkg/errors"
)

func createKanikoCacheWarmerPod(ctx context.Context, client client.Client, platform *v1alpha1.IntegrationPlatform) error {
	// The pod will be scheduled to nodes that are selected by the persistent volume
	// node affinity spec, if any, as provisioned by the persistent volume claim storage
	// class provisioner.
	// See:
	// - https://kubernetes.io/docs/concepts/storage/persistent-volumes/#node-affinity
	// - https://kubernetes.io/docs/concepts/storage/volumes/#local
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
							MountPath: kaniko.BuildDir,
						},
					},
				},
			},
			// Create the cache directory otherwise Kaniko warmer skips caching silently
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
							ClaimName: platform.Spec.Build.PersistentVolumeClaim,
						},
					},
				},
			},
		},
	}

	err := client.Delete(ctx, &pod)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "cannot delete Kaniko warmer pod")
	}

	err = client.Create(ctx, &pod)
	if err != nil {
		return errors.Wrap(err, "cannot create Kaniko warmer pod")
	}

	return nil
}

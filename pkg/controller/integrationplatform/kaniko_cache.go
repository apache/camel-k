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
	"fmt"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/defaults"
)

func createKanikoCacheWarmerPod(ctx context.Context, client client.Client, platform *v1.IntegrationPlatform) error {
	// The pod will be scheduled to nodes that are selected by the persistent volume
	// node affinity spec, if any, as provisioned by the persistent volume claim storage
	// class provisioner.
	// See:
	// - https://kubernetes.io/docs/concepts/storage/persistent-volumes/#node-affinity
	// - https://kubernetes.io/docs/concepts/storage/volumes/#local
	// nolint: staticcheck
	pvcName := platform.Status.Build.PersistentVolumeClaim
	if persistentVolumeClaim, found := platform.Status.Build.PublishStrategyOptions[builder.KanikoPVCName]; found {
		pvcName = persistentVolumeClaim
	}

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: platform.Namespace,
			Name:      platform.Name + "-cache",
			Labels: map[string]string{
				"camel.apache.org/component": "kaniko-warmer",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "warm-kaniko-cache",
					Image: fmt.Sprintf("gcr.io/kaniko-project/warmer:v%s", defaults.KanikoVersion),
					Args: []string{
						"--cache-dir=" + builder.KanikoCacheDir,
						"--image=" + platform.Status.Build.BaseImage,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "kaniko-cache",
							MountPath: builder.KanikoCacheDir,
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
					Command:         []string{"/bin/sh", "-c"},
					Args:            []string{"mkdir -p " + builder.KanikoCacheDir + "&& chmod -R a+rwx " + builder.KanikoCacheDir},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "kaniko-cache",
							MountPath: builder.KanikoCacheDir,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyOnFailure,
			Volumes: []corev1.Volume{
				{
					Name: "kaniko-cache",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
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

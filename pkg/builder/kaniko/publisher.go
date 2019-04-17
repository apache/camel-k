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

package kaniko

import (
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/tar"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func publisher(ctx *builder.Context) error {
	organization := ctx.Build.Platform.Build.Registry.Organization
	if organization == "" {
		organization = ctx.Namespace
	}
	image := ctx.Build.Platform.Build.Registry.Address + "/" + organization + "/camel-k-" + ctx.Build.Meta.Name + ":" + ctx.Build.Meta.ResourceVersion
	baseDir, _ := path.Split(ctx.Archive)
	contextDir := path.Join(baseDir, "context")
	if err := tar.Extract(ctx.Archive, contextDir); err != nil {
		return err
	}

	// #nosec G202
	dockerFileContent := []byte(`
		FROM ` + ctx.Image + `
		ADD . /deployments
	`)

	err := ioutil.WriteFile(path.Join(contextDir, "Dockerfile"), dockerFileContent, 0777)
	if err != nil {
		return err
	}

	volumes := []corev1.Volume{
		{
			Name: "camel-k-builder",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: ctx.Build.Platform.Build.PersistentVolumeClaim,
				},
			},
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "camel-k-builder",
			MountPath: BuildDir,
		},
	}
	envs := make([]corev1.EnvVar, 0)
	baseArgs := []string{
		"--dockerfile=Dockerfile",
		"--context=" + contextDir,
		"--destination=" + image,
		"--cache",
		"--cache-dir=/workspace/cache",
	}

	args := make([]string, 0, len(baseArgs))
	args = append(args, baseArgs...)

	if ctx.Build.Platform.Build.Registry.Insecure {
		args = append(args, "--insecure")
		args = append(args, "--insecure-pull")
	}

	if ctx.Build.Platform.Build.Registry.Secret != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "kaniko-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: ctx.Build.Platform.Build.Registry.Secret,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "kaniko-secret",
			MountPath: "/secret",
		})
		envs = append(envs, corev1.EnvVar{
			Name:  "GOOGLE_APPLICATION_CREDENTIALS",
			Value: "/secret/kaniko-secret.json",
		})
		args = baseArgs
	}

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ctx.Namespace,
			Name:      "camel-k-" + ctx.Build.Meta.Name,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:         "kaniko",
					Image:        "gcr.io/kaniko-project/executor:v0.9.0",
					Args:         args,
					Env:          envs,
					VolumeMounts: volumeMounts,
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes:       volumes,
			// Co-locate with builder pod for sharing the volume
			Affinity: &corev1.Affinity{
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
			},
		},
	}

	err = ctx.Client.Delete(ctx.C, &pod)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "cannot delete kaniko builder pod")
	}

	err = ctx.Client.Create(ctx.C, &pod)
	if err != nil {
		return errors.Wrap(err, "cannot create kaniko builder pod")
	}

	err = kubernetes.WaitCondition(ctx.C, ctx.Client, &pod, func(obj interface{}) (bool, error) {
		if val, ok := obj.(*corev1.Pod); ok {
			if val.Status.Phase == corev1.PodSucceeded {
				return true, nil
			}
			if val.Status.Phase == corev1.PodFailed {
				return false, fmt.Errorf("build failed: %s", val.Status.Message)
			}
		}
		return false, nil
	}, 10*time.Minute)

	if err != nil {
		return err
	}

	ctx.Image = image
	return nil
}

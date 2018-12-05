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
	"io/ioutil"
	"path"
	"time"

	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/tar"

	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Publisher --
func Publisher(ctx *builder.Context) error {
	organization := ctx.Request.Platform.Build.Organization
	if organization == "" {
		organization = ctx.Namespace
	}
	image := ctx.Request.Platform.Build.Registry + "/" + organization + "/camel-k-" + ctx.Request.Meta.Name + ":" + ctx.Request.Meta.ResourceVersion
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

	volumes := []v1.Volume{
		{
			Name: "camel-k-builder",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: "camel-k-builder",
				},
			},
		},
	}
	volumeMounts := []v1.VolumeMount{
		{
			Name:      "camel-k-builder",
			MountPath: "/workspace",
		},
	}
	envs := []v1.EnvVar{}
	baseArgs := []string{
		"--dockerfile=Dockerfile",
		"--context=" + contextDir,
		"--destination=" + image}
	args := append(baseArgs, "--insecure")

	if ctx.Request.Platform.Build.PushSecret != "" {
		volumes = append(volumes, v1.Volume{
			Name: "kaniko-secret",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: ctx.Request.Platform.Build.PushSecret,
				},
			},
		})
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      "kaniko-secret",
			MountPath: "/secret",
		})
		envs = append(envs, v1.EnvVar{
			Name:  "GOOGLE_APPLICATION_CREDENTIALS",
			Value: "/secret/kaniko-secret.json",
		})
		args = baseArgs
	}

	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ctx.Namespace,
			Name:      "camel-k-" + ctx.Request.Meta.Name,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:         "kaniko",
					Image:        "gcr.io/kaniko-project/executor@sha256:f29393d9c8d40296e1692417089aa2023494bce9afd632acac7dd0aea763e5bc",
					Args:         args,
					Env:          envs,
					VolumeMounts: volumeMounts,
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes:       volumes,
			Affinity: &v1.Affinity{
				// Co-locate with builder pod for sharing the volume
				PodAffinity: &v1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
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

	err = sdk.Delete(&pod)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "cannot delete kaniko builder pod")
	}

	err = sdk.Create(&pod)
	if err != nil {
		return errors.Wrap(err, "cannot create kaniko builder pod")
	}

	err = kubernetes.WaitCondition(&pod, func(obj interface{}) (bool, error) {
		if val, ok := obj.(*v1.Pod); ok {
			if val.Status.Phase == v1.PodSucceeded {
				return true, nil
			} else if val.Status.Phase == v1.PodFailed {
				return false, errors.New("build failed")
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

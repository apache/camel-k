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
	"os"
	"path"
	"strconv"
	"time"

	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/tar"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type secretKind struct {
	fileName    string
	mountPath   string
	destination string
	refEnv      string
}

var (
	secretKindGCR = secretKind{
		fileName:    "kaniko-secret.json",
		mountPath:   "/secret",
		destination: "kaniko-secret.json",
		refEnv:      "GOOGLE_APPLICATION_CREDENTIALS",
	}
	secretKindPlainDocker = secretKind{
		fileName:    "config.json",
		mountPath:   "/kaniko/.docker",
		destination: "config.json",
	}
	secretKindStandardDocker = secretKind{
		fileName:    corev1.DockerConfigJsonKey,
		mountPath:   "/kaniko/.docker",
		destination: "config.json",
	}

	allSecretKinds = []secretKind{secretKindGCR, secretKindPlainDocker, secretKindStandardDocker}
)

func publisher(ctx *builder.Context) error {
	organization := ctx.Build.Platform.Build.Registry.Organization
	if organization == "" {
		organization = ctx.Namespace
	}
	image := ctx.Build.Platform.Build.Registry.Address + "/" + organization + "/camel-k-" + ctx.Build.Meta.Name + ":" + ctx.Build.Meta.ResourceVersion
	baseDir, _ := path.Split(ctx.Archive)
	contextDir := path.Join(baseDir, "context")

	err := os.Mkdir(contextDir, 0777)
	if err != nil {
		return err
	}

	if err := tar.Extract(ctx.Archive, contextDir); err != nil {
		return err
	}

	// #nosec G202
	dockerFileContent := []byte(`
		FROM ` + ctx.Image + `
		ADD . /deployments
	`)

	err = ioutil.WriteFile(path.Join(contextDir, "Dockerfile"), dockerFileContent, 0777)
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
		"--cache=" + strconv.FormatBool(ctx.Build.Platform.Build.IsKanikoCacheEnabled()),
		"--cache-dir=/workspace/cache",
	}

	args := make([]string, 0, len(baseArgs))
	args = append(args, baseArgs...)

	if ctx.Build.Platform.Build.Registry.Insecure {
		args = append(args, "--insecure")
		args = append(args, "--insecure-pull")
	}

	if ctx.Build.Platform.Build.Registry.Secret != "" {
		secretKind, err := getSecretKind(ctx, ctx.Build.Platform.Build.Registry.Secret)
		if err != nil {
			return err
		}

		volumes = append(volumes, corev1.Volume{
			Name: "kaniko-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: ctx.Build.Platform.Build.Registry.Secret,
					Items: []corev1.KeyToPath{
						{
							Key:  secretKind.fileName,
							Path: secretKind.destination,
						},
					},
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "kaniko-secret",
			MountPath: secretKind.mountPath,
		})

		if secretKind.refEnv != "" {
			envs = append(envs, corev1.EnvVar{
				Name:  secretKind.refEnv,
				Value: path.Join(secretKind.mountPath, secretKind.destination),
			})
		}
		args = baseArgs
	}

	if ctx.Build.Platform.Build.HTTPProxySecret != "" {
		optional := true
		envs = append(envs, corev1.EnvVar{
			Name: "HTTP_PROXY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ctx.Build.Platform.Build.HTTPProxySecret,
					},
					Key:      "HTTP_PROXY",
					Optional: &optional,
				},
			},
		})
		envs = append(envs, corev1.EnvVar{
			Name: "HTTPS_PROXY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ctx.Build.Platform.Build.HTTPProxySecret,
					},
					Key:      "HTTPS_PROXY",
					Optional: &optional,
				},
			},
		})
		envs = append(envs, corev1.EnvVar{
			Name: "NO_PROXY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ctx.Build.Platform.Build.HTTPProxySecret,
					},
					Key:      "NO_PROXY",
					Optional: &optional,
				},
			},
		})
	}

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ctx.Namespace,
			Name:      "camel-k-" + ctx.Build.Meta.Name,
			Labels: map[string]string{
				"app": "camel-k",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:         "kaniko",
					Image:        fmt.Sprintf("gcr.io/kaniko-project/executor:v%s", defaults.KanikoVersion),
					Args:         args,
					Env:          envs,
					VolumeMounts: volumeMounts,
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes:       volumes,
		},
	}

	// Co-locate with the build pod for sharing the volume
	pod.Spec.Affinity = &corev1.Affinity{
		PodAffinity: &corev1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"camel.apache.org/build": ctx.Build.Meta.Name,
						},
					},
					TopologyKey: "kubernetes.io/hostname",
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

func getSecretKind(ctx *builder.Context, name string) (secretKind, error) {
	secret := corev1.Secret{}
	key := client.ObjectKey{Namespace: ctx.Namespace, Name: name}
	if err := ctx.Client.Get(ctx.C, key, &secret); err != nil {
		return secretKind{}, err
	}
	for _, k := range allSecretKinds {
		if _, ok := secret.Data[k.fileName]; ok {
			return k, nil
		}
	}
	return secretKind{}, errors.New("unsupported secret type for registry authentication")
}

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
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

const (
	builderDir    = "/builder"
	builderVolume = "camel-k-builder"
)

type registryConfigMap struct {
	fileName    string
	mountPath   string
	destination string
}

var (
	serviceCABuildahRegistryConfigMap = registryConfigMap{
		fileName:    "service-ca.crt",
		mountPath:   "/etc/containers/certs.d",
		destination: "service-ca.crt",
	}

	buildahRegistryConfigMaps = []registryConfigMap{
		serviceCABuildahRegistryConfigMap,
	}
)

type registrySecret struct {
	fileName    string
	mountPath   string
	destination string
	refEnv      string
}

var (
	plainDockerBuildahRegistrySecret = registrySecret{
		fileName:    corev1.DockerConfigKey,
		mountPath:   "/buildah/.docker",
		destination: "config.json",
	}
	standardDockerBuildahRegistrySecret = registrySecret{
		fileName:    corev1.DockerConfigJsonKey,
		mountPath:   "/buildah/.docker",
		destination: "config.json",
		refEnv:      "REGISTRY_AUTH_FILE",
	}

	buildahRegistrySecrets = []registrySecret{
		plainDockerBuildahRegistrySecret,
		standardDockerBuildahRegistrySecret,
	}
)

var (
	gcrKanikoRegistrySecret = registrySecret{
		fileName:    "kaniko-secret.json",
		mountPath:   "/secret",
		destination: "kaniko-secret.json",
		refEnv:      "GOOGLE_APPLICATION_CREDENTIALS",
	}
	plainDockerKanikoRegistrySecret = registrySecret{
		fileName:    "config.json",
		mountPath:   "/kaniko/.docker",
		destination: "config.json",
	}
	standardDockerKanikoRegistrySecret = registrySecret{
		fileName:    corev1.DockerConfigJsonKey,
		mountPath:   "/kaniko/.docker",
		destination: "config.json",
	}

	kanikoRegistrySecrets = []registrySecret{
		gcrKanikoRegistrySecret,
		plainDockerKanikoRegistrySecret,
		standardDockerKanikoRegistrySecret,
	}
)

func newBuildPod(ctx context.Context, c ctrl.Reader, build *v1.Build) (*corev1.Pod, error) {
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
		switch {
		case task.Builder != nil:
			addBuildTaskToPod(build, task.Builder.Name, pod)
		case task.Buildah != nil:
			err := addBuildahTaskToPod(ctx, c, build, task.Buildah, pod)
			if err != nil {
				return nil, err
			}
		case task.Kaniko != nil:
			err := addKanikoTaskToPod(ctx, c, build, task.Kaniko, pod)
			if err != nil {
				return nil, err
			}
		case task.S2i != nil:
			addBuildTaskToPod(build, task.S2i.Name, pod)
		case task.Spectrum != nil:
			addBuildTaskToPod(build, task.Spectrum.Name, pod)
		}
	}

	// Make sure there is one container defined
	pod.Spec.Containers = pod.Spec.InitContainers[len(pod.Spec.InitContainers)-1 : len(pod.Spec.InitContainers)]
	pod.Spec.InitContainers = pod.Spec.InitContainers[:len(pod.Spec.InitContainers)-1]

	return pod, nil
}

func deleteBuilderPod(ctx context.Context, c ctrl.Writer, build *v1.Build) error {
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: build.Namespace,
			Name:      buildPodName(build),
		},
	}

	err := c.Delete(ctx, &pod)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	return err
}

func getBuilderPod(ctx context.Context, c ctrl.Reader, build *v1.Build) (*corev1.Pod, error) {
	pod := corev1.Pod{}
	err := c.Get(ctx, ctrl.ObjectKey{Namespace: build.Namespace, Name: buildPodName(build)}, &pod)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &pod, nil
}

func buildPodName(build *v1.Build) string {
	return "camel-k-" + build.Name + "-builder"
}

func addBuildTaskToPod(build *v1.Build, taskName string, pod *corev1.Pod) {
	if !hasBuilderVolume(pod) {
		// Add the EmptyDir volume used to share the build state across tasks
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: builderVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	operatorImage := platform.OperatorImage
	if operatorImage == "" {
		operatorImage = defaults.ImageName + ":" + defaults.Version
	}

	container := corev1.Container{
		Name:            taskName,
		Image:           operatorImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"kamel",
			"builder",
			"--namespace",
			build.Namespace,
			"--build-name",
			build.Name,
			"--task-name",
			taskName,
		},
		WorkingDir: path.Join(builderDir, build.Name),
		Env:        proxyFromEnvironment(),
	}

	addContainerToPod(build, container, pod)
}

func addBuildahTaskToPod(ctx context.Context, c ctrl.Reader, build *v1.Build, task *v1.BuildahTask, pod *corev1.Pod) error {
	bud := []string{
		"buildah",
		"bud",
		"--storage-driver=vfs",
		"--platform",
		task.Platform,
		"--pull-always",
		"-f",
		"Dockerfile",
		"-t",
		task.Image,
		".",
	}

	push := []string{
		"buildah",
		"push",
		"--storage-driver=vfs",
		"--digestfile=/dev/termination-log",
		task.Image,
		"docker://" + task.Image,
	}

	if task.Verbose != nil && *task.Verbose {
		bud = append(bud[:2], append([]string{"--log-level=debug"}, bud[2:]...)...)
		push = append(push[:2], append([]string{"--log-level=debug"}, push[2:]...)...)
	}

	env := make([]corev1.EnvVar, 0)
	volumes := make([]corev1.Volume, 0)
	volumeMounts := make([]corev1.VolumeMount, 0)

	if task.Registry.CA != "" {
		config, err := getRegistryConfigMap(ctx, c, build.Namespace, task.Registry.CA, buildahRegistryConfigMaps)
		if err != nil {
			return err
		}
		addRegistryConfigMap(task.Registry.CA, config, &volumes, &volumeMounts)
		// This is easier to use the --cert-dir option, otherwise Buildah defaults to looking up certificates
		// into a directory named after the registry address
		bud = append(bud[:2], append([]string{"--cert-dir=/etc/containers/certs.d"}, bud[2:]...)...)
		push = append(push[:2], append([]string{"--cert-dir=/etc/containers/certs.d"}, push[2:]...)...)
	}

	var auth string
	if task.Registry.Secret != "" {
		secret, err := getRegistrySecret(ctx, c, build.Namespace, task.Registry.Secret, buildahRegistrySecrets)
		if err != nil {
			return err
		}
		if secret == plainDockerBuildahRegistrySecret {
			// Handle old format and make it compatible with Buildah
			auth = "(echo '{ \"auths\": ' ; cat /buildah/.docker/config.json ; echo \"}\") > /tmp/.dockercfg"
			env = append(env, corev1.EnvVar{
				Name:  "REGISTRY_AUTH_FILE",
				Value: "/tmp/.dockercfg",
			})
		}
		addRegistrySecret(task.Registry.Secret, secret, &volumes, &volumeMounts, &env)
	}

	if task.Registry.Insecure {
		bud = append(bud[:2], append([]string{"--tls-verify=false"}, bud[2:]...)...)
		push = append(push[:2], append([]string{"--tls-verify=false"}, push[2:]...)...)
	}

	env = append(env, proxyFromEnvironment()...)

	args := []string{
		strings.Join(bud, " "),
		strings.Join(push, " "),
	}
	if auth != "" {
		args = append([]string{auth}, args...)
	}

	container := corev1.Container{
		Name:            task.Name,
		Image:           fmt.Sprintf("quay.io/buildah/stable:v%s", defaults.BuildahVersion),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{strings.Join(args, " && ")},
		Env:             env,
		WorkingDir:      path.Join(builderDir, build.Name, builder.ContextDir),
		VolumeMounts:    volumeMounts,
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, volumes...)

	addContainerToPod(build, container, pod)

	return nil
}

func addKanikoTaskToPod(ctx context.Context, c ctrl.Reader, build *v1.Build, task *v1.KanikoTask, pod *corev1.Pod) error {
	cache := false
	if task.Cache.Enabled != nil && *task.Cache.Enabled {
		cache = true
	}

	args := []string{
		"--dockerfile=Dockerfile",
		"--context=" + path.Join(builderDir, build.Name, builder.ContextDir),
		"--destination=" + task.Image,
		"--cache=" + strconv.FormatBool(cache),
		"--cache-dir=" + builder.KanikoCacheDir,
	}

	if task.Verbose != nil && *task.Verbose {
		args = append(args, "-v=debug")
	}

	affinity := &corev1.Affinity{}
	env := make([]corev1.EnvVar, 0)
	volumes := make([]corev1.Volume, 0)
	volumeMounts := make([]corev1.VolumeMount, 0)

	if task.Registry.Secret != "" {
		secret, err := getRegistrySecret(ctx, c, build.Namespace, task.Registry.Secret, kanikoRegistrySecrets)
		if err != nil {
			return err
		}
		addRegistrySecret(task.Registry.Secret, secret, &volumes, &volumeMounts, &env)
	}

	if task.Registry.Insecure {
		args = append(args, "--insecure")
		args = append(args, "--insecure-pull")
	}

	env = append(env, proxyFromEnvironment()...)

	if cache {
		// Co-locate with the Kaniko warmer pod for sharing the host path volume as the current
		// persistent volume claim uses the default storage class which is likely relying
		// on the host path provisioner.
		// This has to be done manually by retrieving the Kaniko warmer pod node name and using
		// node affinity as pod affinity only works for running pods and the Kaniko warmer pod
		// has already completed at that stage.

		// Locate the kaniko warmer pod
		pods := &corev1.PodList{}
		err := c.List(ctx, pods,
			ctrl.InNamespace(build.Namespace),
			ctrl.MatchingLabels{
				"camel.apache.org/component": "kaniko-warmer",
			})
		if err != nil {
			return err
		}

		if len(pods.Items) != 1 {
			return errors.New("failed to locate the Kaniko cache warmer pod")
		}

		// Use node affinity with the Kaniko warmer pod node name
		affinity = &corev1.Affinity{
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
		// Mount the PV used to warm the Kaniko cache into the Kaniko image build
		volumes = append(volumes, corev1.Volume{
			Name: "kaniko-cache",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: task.Cache.PersistentVolumeClaim,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "kaniko-cache",
			MountPath: builder.KanikoCacheDir,
		})
	}

	container := corev1.Container{
		Name:            task.Name,
		Image:           fmt.Sprintf("gcr.io/kaniko-project/executor:v%s", defaults.KanikoVersion),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            args,
		Env:             env,
		WorkingDir:      path.Join(builderDir, build.Name, builder.ContextDir),
		VolumeMounts:    volumeMounts,
	}

	// We may want to handle possible conflicts
	pod.Spec.Affinity = affinity
	pod.Spec.Volumes = append(pod.Spec.Volumes, volumes...)

	addContainerToPod(build, container, pod)

	return nil
}

func addContainerToPod(build *v1.Build, container corev1.Container, pod *corev1.Pod) {
	if hasBuilderVolume(pod) {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      builderVolume,
			MountPath: path.Join(builderDir, build.Name),
		})
	}

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)
}

func hasBuilderVolume(pod *corev1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == builderVolume {
			return true
		}
	}
	return false
}

func getRegistryConfigMap(ctx context.Context, c ctrl.Reader, ns, name string, registryConfigMaps []registryConfigMap) (registryConfigMap, error) {
	config := corev1.ConfigMap{}
	err := c.Get(ctx, ctrl.ObjectKey{Namespace: ns, Name: name}, &config)
	if err != nil {
		return registryConfigMap{}, err
	}
	for _, k := range registryConfigMaps {
		if _, ok := config.Data[k.fileName]; ok {
			return k, nil
		}
	}
	return registryConfigMap{}, errors.New("unsupported registry config map")
}

func addRegistryConfigMap(name string, config registryConfigMap, volumes *[]corev1.Volume, volumeMounts *[]corev1.VolumeMount) {
	*volumes = append(*volumes, corev1.Volume{
		Name: "registry-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
				Items: []corev1.KeyToPath{
					{
						Key:  config.fileName,
						Path: config.destination,
					},
				},
			},
		},
	})

	*volumeMounts = append(*volumeMounts, corev1.VolumeMount{
		Name:      "registry-config",
		MountPath: config.mountPath,
		ReadOnly:  true,
	})
}

func getRegistrySecret(ctx context.Context, c ctrl.Reader, ns, name string, registrySecrets []registrySecret) (registrySecret, error) {
	secret := corev1.Secret{}
	err := c.Get(ctx, ctrl.ObjectKey{Namespace: ns, Name: name}, &secret)
	if err != nil {
		return registrySecret{}, err
	}
	for _, k := range registrySecrets {
		if _, ok := secret.Data[k.fileName]; ok {
			return k, nil
		}
	}
	return registrySecret{}, errors.New("unsupported secret type for registry authentication")
}

func addRegistrySecret(name string, secret registrySecret, volumes *[]corev1.Volume, volumeMounts *[]corev1.VolumeMount, env *[]corev1.EnvVar) {
	*volumes = append(*volumes, corev1.Volume{
		Name: "registry-secret",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: name,
				Items: []corev1.KeyToPath{
					{
						Key:  secret.fileName,
						Path: secret.destination,
					},
				},
			},
		},
	})

	*volumeMounts = append(*volumeMounts, corev1.VolumeMount{
		Name:      "registry-secret",
		MountPath: secret.mountPath,
		ReadOnly:  true,
	})

	if secret.refEnv != "" {
		*env = append(*env, corev1.EnvVar{
			Name:  secret.refEnv,
			Value: path.Join(secret.mountPath, secret.destination),
		})
	}
}

func proxyFromEnvironment() []corev1.EnvVar {
	var envVars []corev1.EnvVar

	if httpProxy, ok := os.LookupEnv("HTTP_PROXY"); ok {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "HTTP_PROXY",
			Value: httpProxy,
		})
	}

	if httpsProxy, ok := os.LookupEnv("HTTPS_PROXY"); ok {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "HTTPS_PROXY",
			Value: httpsProxy,
		})
	}

	if noProxy, ok := os.LookupEnv("NO_PROXY"); ok {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "NO_PROXY",
			Value: noProxy,
		})
	}

	return envVars
}

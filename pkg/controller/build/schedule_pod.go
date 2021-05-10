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
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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

type schedulePodAction struct {
	baseAction
	lock          sync.Mutex
	reader        client.Reader
	operatorImage string
}

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

func newSchedulePodAction(reader client.Reader) Action {
	return &schedulePodAction{
		reader: reader,
	}
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

	// Reset the Build status, and transition it to pending phase
	build.Status = v1.BuildStatus{
		Phase:      v1.BuildPhasePending,
		StartedAt:  &pod.CreationTimestamp,
		Failure:    build.Status.Failure,
		Platform:   build.Status.Platform,
		Conditions: build.Status.Conditions,
	}

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

	for _, task := range build.Spec.Tasks {
		if task.Builder != nil {
			action.addBuildTaskToPod(build, task.Builder.Name, pod)
		} else if task.Buildah != nil {
			err := action.addBuildahTaskToPod(ctx, build, task.Buildah, pod)
			if err != nil {
				return nil, err
			}
		} else if task.Kaniko != nil {
			err := action.addKanikoTaskToPod(ctx, build, task.Kaniko, pod)
			if err != nil {
				return nil, err
			}
		} else if task.S2i != nil {
			action.addBuildTaskToPod(build, task.S2i.Name, pod)
		} else if task.Spectrum != nil {
			action.addBuildTaskToPod(build, task.Spectrum.Name, pod)
		}
	}

	// Make sure there is one container defined
	pod.Spec.Containers = pod.Spec.InitContainers[len(pod.Spec.InitContainers)-1 : len(pod.Spec.InitContainers)]
	pod.Spec.InitContainers = pod.Spec.InitContainers[:len(pod.Spec.InitContainers)-1]

	return pod, nil
}

func (action *schedulePodAction) addBuildTaskToPod(build *v1.Build, taskName string, pod *corev1.Pod) {
	if !action.hasBuilderVolume(pod) {
		// Add the EmptyDir volume used to share the build state across tasks
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: builderVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	container := corev1.Container{
		Name:            taskName,
		Image:           action.operatorImage,
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
	}

	action.addContainerToPod(build, container, pod)
}

func (action *schedulePodAction) addBuildahTaskToPod(ctx context.Context, build *v1.Build, task *v1.BuildahTask, pod *corev1.Pod) error {
	bud := []string{
		"buildah",
		"bud",
		"--storage-driver=vfs",
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
		config, err := action.getRegistryConfigMap(ctx, build.Namespace, task.Registry.CA, buildahRegistryConfigMaps)
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
		secret, err := action.getRegistrySecret(ctx, build.Namespace, task.Registry.Secret, buildahRegistrySecrets)
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

	env = append(env, proxySecretEnvVars(task.HttpProxySecret)...)

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

	action.addContainerToPod(build, container, pod)

	return nil
}

func (action *schedulePodAction) addKanikoTaskToPod(ctx context.Context, build *v1.Build, task *v1.KanikoTask, pod *corev1.Pod) error {
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
		secret, err := action.getRegistrySecret(ctx, build.Namespace, task.Registry.Secret, kanikoRegistrySecrets)
		if err != nil {
			return err
		}
		addRegistrySecret(task.Registry.Secret, secret, &volumes, &volumeMounts, &env)
	}

	if task.Registry.Insecure {
		args = append(args, "--insecure")
		args = append(args, "--insecure-pull")
	}

	env = append(env, proxySecretEnvVars(task.HttpProxySecret)...)

	if cache {
		// Co-locate with the Kaniko warmer pod for sharing the host path volume as the current
		// persistent volume claim uses the default storage class which is likely relying
		// on the host path provisioner.
		// This has to be done manually by retrieving the Kaniko warmer pod node name and using
		// node affinity as pod affinity only works for running pods and the Kaniko warmer pod
		// has already completed at that stage.

		// Locate the kaniko warmer pod
		pods := &corev1.PodList{}
		err := action.client.List(ctx, pods,
			client.InNamespace(build.Namespace),
			client.MatchingLabels{
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

	action.addContainerToPod(build, container, pod)

	return nil
}

func (action *schedulePodAction) addContainerToPod(build *v1.Build, container corev1.Container, pod *corev1.Pod) {
	if action.hasBuilderVolume(pod) {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      builderVolume,
			MountPath: path.Join(builderDir, build.Name),
		})
	}

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)
}

func (action *schedulePodAction) hasBuilderVolume(pod *corev1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == builderVolume {
			return true
		}
	}
	return false
}

func (action *schedulePodAction) getRegistryConfigMap(ctx context.Context, ns, name string, registryConfigMaps []registryConfigMap) (registryConfigMap, error) {
	config := corev1.ConfigMap{}
	err := action.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &config)
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

func (action *schedulePodAction) getRegistrySecret(ctx context.Context, ns, name string, registrySecrets []registrySecret) (registrySecret, error) {
	secret := corev1.Secret{}
	err := action.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &secret)
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

func proxySecretEnvVars(secret string) []corev1.EnvVar {
	if secret == "" {
		return []corev1.EnvVar{}
	}

	return []corev1.EnvVar{
		proxySecretEnvVar("HTTP_PROXY", secret),
		proxySecretEnvVar("HTTPS_PROXY", secret),
		proxySecretEnvVar("NO_PROXY", secret),
	}
}

func proxySecretEnvVar(name string, secret string) corev1.EnvVar {
	optional := true
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secret,
				},
				Key:      name,
				Optional: &optional,
			},
		},
	}
}

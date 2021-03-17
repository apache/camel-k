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

package trait

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/defaults"
)

const builderDir = "/builder"

// The builder trait is internally used to determine the best strategy to
// build and configure IntegrationKits.
//
// +camel-k:trait=builder
type builderTrait struct {
	BaseTrait `property:",squash"`
	// Enable verbose logging on build components that support it (e.g. Kaniko build pod).
	Verbose bool `property:"verbose" json:"verbose,omitempty"`
}

func newBuilderTrait() Trait {
	return &builderTrait{
		BaseTrait: NewBaseTrait("builder", 600),
	}
}

// IsPlatformTrait overrides base class method
func (t *builderTrait) IsPlatformTrait() bool {
	return true
}

// InfluencesKit overrides base class method
func (t *builderTrait) InfluencesKit() bool {
	return true
}

func (t *builderTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted), nil
}

func (t *builderTrait) Apply(e *Environment) error {
	builderTask := t.builderTask(e)

	switch e.Platform.Status.Build.PublishStrategy {
	case v1.IntegrationPlatformBuildPublishStrategyBuildah, v1.IntegrationPlatformBuildPublishStrategyKaniko:
		builderTask.BuildDir = path.Join(builderDir, e.IntegrationKit.Name)
	default:
		tmpDir, err := ioutil.TempDir(os.TempDir(), e.IntegrationKit.Name+"-")
		if err != nil {
			return err
		}
		builderTask.BuildDir = tmpDir
	}
	e.BuildTasks = append(e.BuildTasks, v1.Task{Builder: builderTask})

	switch e.Platform.Status.Build.PublishStrategy {
	case v1.IntegrationPlatformBuildPublishStrategySpectrum:
		e.BuildTasks = append(e.BuildTasks, v1.Task{Spectrum: &v1.SpectrumTask{
			BaseTask: v1.BaseTask{
				Name: "spectrum",
			},
			PublishTask: v1.PublishTask{
				ContextDir: path.Join(builderTask.BuildDir, "context"),
				BaseImage:  e.Platform.Status.Build.BaseImage,
				Image:      getImageName(e),
			},
			Registry: e.Platform.Status.Build.Registry,
		}})

	case v1.IntegrationPlatformBuildPublishStrategyS2I:
		e.BuildTasks = append(e.BuildTasks, v1.Task{S2i: &v1.S2iTask{
			BaseTask: v1.BaseTask{
				Name: "s2i",
			},
			PublishTask: v1.PublishTask{
				BaseImage:  e.Platform.Status.Build.BaseImage,
				ContextDir: path.Join(builderTask.BuildDir, "context"),
			},
			Tag: e.IntegrationKit.ResourceVersion,
		}})

	case v1.IntegrationPlatformBuildPublishStrategyBuildah:
		imageTask, err := t.buildahTask(e)
		if err != nil {
			return err
		}
		t.addVolumeMounts(builderTask, imageTask)
		e.BuildTasks = append(e.BuildTasks, v1.Task{Image: imageTask})

	case v1.IntegrationPlatformBuildPublishStrategyKaniko:
		imageTask, err := t.kanikoTask(e)
		if err != nil {
			return err
		}

		if e.Platform.Status.Build.IsKanikoCacheEnabled() {
			// Co-locate with the Kaniko warmer pod for sharing the host path volume as the current
			// persistent volume claim uses the default storage class which is likely relying
			// on the host path provisioner.
			// This has to be done manually by retrieving the Kaniko warmer pod node name and using
			// node affinity as pod affinity only works for running pods and the Kaniko warmer pod
			// has already completed at that stage.

			// Locate the kaniko warmer pod
			pods := &corev1.PodList{}
			err := e.Client.List(e.C, pods,
				client.InNamespace(e.Platform.Namespace),
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
			imageTask.Affinity = &corev1.Affinity{
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
			imageTask.Volumes = append(imageTask.Volumes, corev1.Volume{
				Name: "kaniko-cache",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: e.Platform.Status.Build.PersistentVolumeClaim,
					},
				},
			})
			imageTask.VolumeMounts = append(imageTask.VolumeMounts, corev1.VolumeMount{
				Name:      "kaniko-cache",
				MountPath: builder.KanikoCacheDir,
			})
		}

		t.addVolumeMounts(builderTask, imageTask)
		e.BuildTasks = append(e.BuildTasks, v1.Task{Image: imageTask})
	}

	return nil
}

func (t *builderTrait) builderTask(e *Environment) *v1.BuilderTask {
	task := &v1.BuilderTask{
		BaseTask: v1.BaseTask{
			Name: "builder",
		},
		Runtime:      e.CamelCatalog.Runtime,
		Dependencies: e.IntegrationKit.Spec.Dependencies,
		Properties:   e.Platform.Status.Build.Properties,
		Timeout:      e.Platform.Status.Build.GetTimeout(),
		Maven:        e.Platform.Status.Build.Maven,
	}

	steps := make([]builder.Step, 0)
	steps = append(steps, builder.DefaultSteps...)

	quarkus := e.Catalog.GetTrait("quarkus").(*quarkusTrait)
	quarkus.addBuildSteps(&steps)

	// sort steps by phase
	sort.SliceStable(steps, func(i, j int) bool {
		return steps[i].Phase() < steps[j].Phase()
	})

	task.Steps = builder.StepIDsFor(steps...)

	return task
}

func (t *builderTrait) buildahTask(e *Environment) (*v1.ImageTask, error) {
	image := getImageName(e)

	bud := []string{
		"buildah",
		"bud",
		"--storage-driver=vfs",
		"-f",
		"Dockerfile",
		"-t",
		image,
		".",
	}

	push := []string{
		"buildah",
		"push",
		"--storage-driver=vfs",
		"--digestfile=/dev/termination-log",
		image,
		"docker://" + image,
	}

	if t.Verbose {
		bud = append(bud[:2], append([]string{"--log-level=debug"}, bud[2:]...)...)
		push = append(push[:2], append([]string{"--log-level=debug"}, push[2:]...)...)
	}

	env := make([]corev1.EnvVar, 0)
	volumes := make([]corev1.Volume, 0)
	volumeMounts := make([]corev1.VolumeMount, 0)

	if e.Platform.Status.Build.Registry.CA != "" {
		config, err := getRegistryConfigMapFor(e, buildahRegistryConfigMaps)
		if err != nil {
			return nil, err
		}
		mountRegistryConfigMap(e.Platform.Status.Build.Registry.CA, config, &volumes, &volumeMounts)
		// This is easier to use the --cert-dir option, otherwise Buildah defaults to looking up certificates
		// into a directory named after the registry address
		bud = append(bud[:2], append([]string{"--cert-dir=/etc/containers/certs.d"}, bud[2:]...)...)
		push = append(push[:2], append([]string{"--cert-dir=/etc/containers/certs.d"}, push[2:]...)...)
	}

	var auth string
	if e.Platform.Status.Build.Registry.Secret != "" {
		secret, err := getRegistrySecretFor(e, buildahRegistrySecrets)
		if err != nil {
			return nil, err
		}
		if secret == plainDockerBuildahRegistrySecret {
			// Handle old format and make it compatible with Buildah
			auth = "(echo '{ \"auths\": ' ; cat /buildah/.docker/config.json ; echo \"}\") > /tmp/.dockercfg"
			env = append(env, corev1.EnvVar{
				Name:  "REGISTRY_AUTH_FILE",
				Value: "/tmp/.dockercfg",
			})
		}
		mountRegistrySecret(e.Platform.Status.Build.Registry.Secret, secret, &volumes, &volumeMounts, &env)
	}

	if e.Platform.Status.Build.Registry.Insecure {
		bud = append(bud[:2], append([]string{"--tls-verify=false"}, bud[2:]...)...)
		push = append(push[:2], append([]string{"--tls-verify=false"}, push[2:]...)...)
	}

	env = append(env, proxySecretEnvVars(e)...)

	args := []string{
		strings.Join(bud, " "),
		strings.Join(push, " "),
	}
	if auth != "" {
		args = append([]string{auth}, args...)
	}

	var sc *corev1.SecurityContext
	if e.Platform.Status.Cluster == v1.IntegrationPlatformClusterOpenShift {
		// This requires the builder service account to have privileged SCC on OpenShift
		// It should be removed when Buildah fully supports unprivileged build
		sc = &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"SETGID",
					"SETUID",
				},
			},
		}
	}

	return &v1.ImageTask{
		BaseTask: v1.BaseTask{
			Name:         "buildah",
			Volumes:      volumes,
			VolumeMounts: volumeMounts,
		},
		PublishTask: v1.PublishTask{
			Image: image,
		},
		Container: v1.ContainerTask{
			Image:           fmt.Sprintf("quay.io/buildah/stable:v%s", defaults.BuildahVersion),
			Command:         []string{"/bin/sh", "-c"},
			Args:            []string{strings.Join(args, " && ")},
			Env:             env,
			WorkingDir:      path.Join(builderDir, e.IntegrationKit.Name, "context"),
			SecurityContext: sc,
		},
	}, nil
}

func (t *builderTrait) kanikoTask(e *Environment) (*v1.ImageTask, error) {
	image := getImageName(e)

	args := []string{
		"--dockerfile=Dockerfile",
		"--context=" + path.Join(builderDir, e.IntegrationKit.Name, "context"),
		"--destination=" + image,
		"--cache=" + strconv.FormatBool(e.Platform.Status.Build.IsKanikoCacheEnabled()),
		"--cache-dir=" + builder.KanikoCacheDir,
	}

	if t.Verbose {
		args = append(args, "-v=debug")
	}

	env := make([]corev1.EnvVar, 0)
	volumes := make([]corev1.Volume, 0)
	volumeMounts := make([]corev1.VolumeMount, 0)

	if e.Platform.Status.Build.Registry.Secret != "" {
		secret, err := getRegistrySecretFor(e, kanikoRegistrySecrets)
		if err != nil {
			return nil, err
		}
		mountRegistrySecret(e.Platform.Status.Build.Registry.Secret, secret, &volumes, &volumeMounts, &env)
	}

	if e.Platform.Status.Build.Registry.Insecure {
		args = append(args, "--insecure")
		args = append(args, "--insecure-pull")

		// We use these 2 flags to force Kaniko to try HTTP instead of HTTPS
		args = append(args, "--skip-tls-verify")
		args = append(args, "--skip-tls-verify-pull")
	}

	env = append(env, proxySecretEnvVars(e)...)

	return &v1.ImageTask{
		BaseTask: v1.BaseTask{
			Name:         "kaniko",
			Volumes:      volumes,
			VolumeMounts: volumeMounts,
		},
		PublishTask: v1.PublishTask{
			Image: image,
		},
		Container: v1.ContainerTask{
			Image: fmt.Sprintf("gcr.io/kaniko-project/executor:v%s", defaults.KanikoVersion),
			Args:  args,
			Env:   env,
		},
	}, nil
}

func (t *builderTrait) addVolumeMounts(builderTask *v1.BuilderTask, imageTask *v1.ImageTask) {
	mount := corev1.VolumeMount{Name: "camel-k-builder", MountPath: builderDir}
	builderTask.VolumeMounts = append(builderTask.VolumeMounts, mount)
	imageTask.VolumeMounts = append(imageTask.VolumeMounts, mount)

	// Use an emptyDir volume to coordinate the Maven build and the image build
	builderTask.Volumes = append(builderTask.Volumes, corev1.Volume{
		Name: "camel-k-builder",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
}

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

func proxySecretEnvVars(e *Environment) []corev1.EnvVar {
	if e.Platform.Status.Build.HTTPProxySecret == "" {
		return []corev1.EnvVar{}
	}

	return []corev1.EnvVar{
		proxySecretEnvVar("HTTP_PROXY", e.Platform.Status.Build.HTTPProxySecret),
		proxySecretEnvVar("HTTPS_PROXY", e.Platform.Status.Build.HTTPProxySecret),
		proxySecretEnvVar("NO_PROXY", e.Platform.Status.Build.HTTPProxySecret),
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

func getRegistrySecretFor(e *Environment, registrySecrets []registrySecret) (registrySecret, error) {
	secret := corev1.Secret{}
	err := e.Client.Get(e.C, client.ObjectKey{Namespace: e.Platform.Namespace, Name: e.Platform.Status.Build.Registry.Secret}, &secret)
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

func mountRegistrySecret(name string, secret registrySecret, volumes *[]corev1.Volume, volumeMounts *[]corev1.VolumeMount, env *[]corev1.EnvVar) {
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

func getRegistryConfigMapFor(e *Environment, registryConfigMaps []registryConfigMap) (registryConfigMap, error) {
	config := corev1.ConfigMap{}
	err := e.Client.Get(e.C, client.ObjectKey{Namespace: e.Platform.Namespace, Name: e.Platform.Status.Build.Registry.CA}, &config)
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

func mountRegistryConfigMap(name string, config registryConfigMap, volumes *[]corev1.Volume, volumeMounts *[]corev1.VolumeMount) {
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

func getImageName(e *Environment) string {
	organization := e.Platform.Status.Build.Registry.Organization
	if organization == "" {
		organization = e.Platform.Namespace
	}
	return e.Platform.Status.Build.Registry.Address + "/" + organization + "/camel-k-" + e.IntegrationKit.Name + ":" + e.IntegrationKit.ResourceVersion
}

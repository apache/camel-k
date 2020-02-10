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
	"path"
	"strconv"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/builder/kaniko"
	"github.com/apache/camel-k/pkg/builder/runtime"
	"github.com/apache/camel-k/pkg/builder/s2i"
	"github.com/apache/camel-k/pkg/platform"
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
	Verbose bool `property:"verbose"`
}

func newBuilderTrait() *builderTrait {
	return &builderTrait{
		BaseTrait: newBaseTrait("builder"),
	}
}

func (t *builderTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted), nil
}

func (t *builderTrait) Apply(e *Environment) error {
	builderTask := t.builderTask(e)
	e.BuildTasks = append(e.BuildTasks, v1.Task{Builder: builderTask})

	if platform.SupportsKanikoPublishStrategy(e.Platform) {
		kanikoTask, err := t.kanikoTask(e)
		if err != nil {
			return err
		}

		mount := corev1.VolumeMount{Name: "camel-k-builder", MountPath: builderDir}
		builderTask.VolumeMounts = append(builderTask.VolumeMounts, mount)
		kanikoTask.VolumeMounts = append(kanikoTask.VolumeMounts, mount)

		// Use an emptyDir volume to coordinate the Camel Maven build and the Kaniko image build
		builderTask.Volumes = append(builderTask.Volumes, corev1.Volume{
			Name: "camel-k-builder",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})

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
			kanikoTask.Affinity = &corev1.Affinity{
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
			kanikoTask.Volumes = append(kanikoTask.Volumes, corev1.Volume{
				Name: "kaniko-cache",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: e.Platform.Status.Build.PersistentVolumeClaim,
					},
				},
			})
			kanikoTask.VolumeMounts = append(kanikoTask.VolumeMounts, corev1.VolumeMount{
				Name:      "kaniko-cache",
				MountPath: kaniko.CacheDir,
			})
		}

		e.BuildTasks = append(e.BuildTasks, v1.Task{Image: kanikoTask})
	}

	return nil
}

// IsPlatformTrait overrides base class method
func (t *builderTrait) IsPlatformTrait() bool {
	return true
}

// InfluencesKit overrides base class method
func (t *builderTrait) InfluencesKit() bool {
	return true
}

func (t *builderTrait) builderTask(e *Environment) *v1.BuilderTask {
	task := &v1.BuilderTask{
		BaseTask: v1.BaseTask{
			Name: "builder",
		},
		Meta:      e.IntegrationKit.ObjectMeta,
		BaseImage: e.Platform.Status.Build.BaseImage,
		Runtime:   e.CamelCatalog.Runtime,
		//Sources:         e.Integration.Spec.Sources,
		//Resources:       e.Integration.Spec.Resources,
		Dependencies: e.IntegrationKit.Spec.Dependencies,
		//TODO: sort steps for easier read
		Steps:      builder.StepIDsFor(builder.DefaultSteps...),
		Properties: e.Platform.Status.Build.Properties,
		Timeout:    e.Platform.Status.Build.GetTimeout(),
		Maven:      e.Platform.Status.Build.Maven,
	}

	if platform.SupportsS2iPublishStrategy(e.Platform) {
		task.Steps = append(task.Steps, builder.StepIDsFor(s2i.S2iSteps...)...)
	} else if platform.SupportsKanikoPublishStrategy(e.Platform) {
		task.Steps = append(task.Steps, builder.StepIDsFor(kaniko.KanikoSteps...)...)
		task.BuildDir = path.Join(builderDir, e.IntegrationKit.Name)
	}

	quarkus := e.Catalog.GetTrait("quarkus").(*quarkusTrait)
	if quarkus.isEnabled() {
		// Add build steps for Quarkus runtime
		quarkus.addBuildSteps(task)
	} else {
		// Add build steps for default runtime
		task.Steps = append(task.Steps, builder.StepIDsFor(runtime.MainSteps...)...)
	}

	return task
}

func (t *builderTrait) kanikoTask(e *Environment) (*v1.ImageTask, error) {
	organization := e.Platform.Status.Build.Registry.Organization
	if organization == "" {
		organization = e.Platform.Namespace
	}
	image := e.Platform.Status.Build.Registry.Address + "/" + organization + "/camel-k-" + e.IntegrationKit.Name + ":" + e.IntegrationKit.ResourceVersion

	env := make([]corev1.EnvVar, 0)
	baseArgs := []string{
		"--dockerfile=Dockerfile",
		"--context=" + path.Join(builderDir, e.IntegrationKit.Name, "package", "context"),
		"--destination=" + image,
		"--cache=" + strconv.FormatBool(e.Platform.Status.Build.IsKanikoCacheEnabled()),
		"--cache-dir=" + kaniko.CacheDir,
	}

	args := make([]string, 0, len(baseArgs))
	args = append(args, baseArgs...)

	if t.Verbose {
		args = append(args, "-v=debug")
	}

	if e.Platform.Status.Build.Registry.Insecure {
		args = append(args, "--insecure")
		args = append(args, "--insecure-pull")
	}

	volumes := make([]corev1.Volume, 0)
	volumeMounts := make([]corev1.VolumeMount, 0)

	if e.Platform.Status.Build.Registry.Secret != "" {
		secretKind, err := getSecretKind(e)
		if err != nil {
			return nil, err
		}

		volumes = append(volumes, corev1.Volume{
			Name: "kaniko-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: e.Platform.Status.Build.Registry.Secret,
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
			env = append(env, corev1.EnvVar{
				Name:  secretKind.refEnv,
				Value: path.Join(secretKind.mountPath, secretKind.destination),
			})
		}
		args = baseArgs
	}

	if e.Platform.Status.Build.HTTPProxySecret != "" {
		optional := true
		env = append(env, corev1.EnvVar{
			Name: "HTTP_PROXY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: e.Platform.Status.Build.HTTPProxySecret,
					},
					Key:      "HTTP_PROXY",
					Optional: &optional,
				},
			},
		})
		env = append(env, corev1.EnvVar{
			Name: "HTTPS_PROXY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: e.Platform.Status.Build.HTTPProxySecret,
					},
					Key:      "HTTPS_PROXY",
					Optional: &optional,
				},
			},
		})
		env = append(env, corev1.EnvVar{
			Name: "NO_PROXY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: e.Platform.Status.Build.HTTPProxySecret,
					},
					Key:      "NO_PROXY",
					Optional: &optional,
				},
			},
		})
	}

	return &v1.ImageTask{
		ContainerTask: v1.ContainerTask{
			BaseTask: v1.BaseTask{
				Name:         "kaniko",
				Volumes:      volumes,
				VolumeMounts: volumeMounts,
			},
			Image: fmt.Sprintf("gcr.io/kaniko-project/executor:v%s", defaults.KanikoVersion),
			Args:  args,
			Env:   env,
		},
		BuiltImage: image,
	}, nil
}

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

func getSecretKind(e *Environment) (secretKind, error) {
	secret := corev1.Secret{}
	err := e.Client.Get(e.C, client.ObjectKey{Namespace: e.Platform.Namespace, Name: e.Platform.Status.Build.Registry.Secret}, &secret)
	if err != nil {
		return secretKind{}, err
	}
	for _, k := range allSecretKinds {
		if _, ok := secret.Data[k.fileName]; ok {
			return k, nil
		}
	}
	return secretKind{}, errors.New("unsupported secret type for registry authentication")
}

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
	"strings"

	"github.com/apache/camel-k/pkg/util/kubernetes"

	"github.com/apache/camel-k/pkg/util/envvar"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type deploymentTrait struct {
	BaseTrait      `property:",squash"`
	ContainerImage bool `property:"container-image"`
}

func newDeploymentTrait() *deploymentTrait {
	return &deploymentTrait{
		BaseTrait: BaseTrait{
			id: ID("deployment"),
		},
	}
}

func (t *deploymentTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		//
		// Don't deploy when a different strategy is needed (e.g. Knative)
		//
		var strategy ControllerStrategy
		var err error
		if strategy, err = e.DetermineControllerStrategy(t.ctx, t.client); err != nil {
			return false, err
		}
		return strategy == ControllerStrategyDeployment, nil
	}

	if t.ContainerImage && e.InPhase(v1alpha1.IntegrationContextPhaseReady, v1alpha1.IntegrationPhaseBuildingContext) {
		return true, nil
	}

	if !t.ContainerImage && e.InPhase(v1alpha1.IntegrationContextPhaseReady, v1alpha1.IntegrationPhaseBuildingContext) {
		return true, nil
	}

	return false, nil
}

func (t *deploymentTrait) Apply(e *Environment) error {
	if t.ContainerImage && e.InPhase(v1alpha1.IntegrationContextPhaseReady, v1alpha1.IntegrationPhaseBuildingContext) {
		// trigger container image build
		e.Integration.Status.Phase = v1alpha1.IntegrationPhaseBuildImageSubmitted
	}

	if !t.ContainerImage && e.InPhase(v1alpha1.IntegrationContextPhaseReady, v1alpha1.IntegrationPhaseBuildingContext) {
		// trigger integration deploy
		e.Integration.Status.Phase = v1alpha1.IntegrationPhaseDeploying
	}

	if e.Integration != nil && e.Integration.Status.Phase == v1alpha1.IntegrationPhaseDeploying {
		e.Resources.AddAll(t.getConfigMapsFor(e))
		e.Resources.Add(t.getDeploymentFor(e))
	}

	return nil
}

// **********************************
//
// ConfigMap
//
// **********************************

func (t *deploymentTrait) getConfigMapsFor(e *Environment) []runtime.Object {
	sources := e.Integration.Sources()
	maps := make([]runtime.Object, 0, len(sources)+1)

	// combine properties of integration with context, integration
	// properties have the priority
	properties := ""

	VisitKeyValConfigurations("property", e.Context, e.Integration, func(key string, val string) {
		properties += fmt.Sprintf("%s=%s\n", key, val)
	})

	maps = append(
		maps,
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      e.Integration.Name + "-properties",
				Namespace: e.Integration.Namespace,
				Labels: map[string]string{
					"camel.apache.org/integration": e.Integration.Name,
				},
			},
			Data: map[string]string{
				"properties": properties,
			},
		},
	)

	if !t.ContainerImage {

		// do not create 'source' or 'resource' ConfigMap if a docker images for deployment
		// is required

		for i, s := range sources {
			if s.ContentRef != "" {
				continue
			}

			cm := corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-source-%03d", e.Integration.Name, i),
					Namespace: e.Integration.Namespace,
					Labels: map[string]string{
						"camel.apache.org/integration": e.Integration.Name,
					},
					Annotations: map[string]string{
						"camel.apache.org/source.language":    string(s.InferLanguage()),
						"camel.apache.org/source.name":        s.Name,
						"camel.apache.org/source.compression": strconv.FormatBool(s.Compression),
					},
				},
				Data: map[string]string{
					"content": s.Content,
				},
			}

			maps = append(maps, &cm)
		}

		for i, s := range e.Integration.Spec.Resources {
			if s.Type != v1alpha1.ResourceTypeData {
				continue
			}

			cm := corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-resource-%03d", e.Integration.Name, i),
					Namespace: e.Integration.Namespace,
					Labels: map[string]string{
						"camel.apache.org/integration": e.Integration.Name,
					},
					Annotations: map[string]string{
						"camel.apache.org/resource.name":        s.Name,
						"camel.apache.org/resource.compression": strconv.FormatBool(s.Compression),
					},
				},
				Data: map[string]string{
					"content": s.Content,
				},
			}

			maps = append(maps, &cm)
		}
	}

	return maps
}

// **********************************
//
// Deployment
//
// **********************************

func (t *deploymentTrait) getSources(e *Environment) []string {
	sources := e.Integration.Sources()
	paths := make([]string, 0, len(sources))

	for _, s := range sources {
		root := "/etc/camel/sources"

		if t.ContainerImage {

			// assume sources are copied over the standard deployments folder
			root = "/deployments/sources"
		}

		srcName := strings.TrimPrefix(s.Name, "/")
		src := path.Join(root, srcName)
		src = "file:" + src

		params := make([]string, 0)
		if s.InferLanguage() != "" {
			params = append(params, "language="+string(s.InferLanguage()))
		}
		if s.Compression {
			params = append(params, "compression=true")
		}

		if len(params) > 0 {
			src = fmt.Sprintf("%s?%s", src, strings.Join(params, "&"))
		}

		paths = append(paths, src)
	}

	return paths
}

func (t *deploymentTrait) getDeploymentFor(e *Environment) *appsv1.Deployment {
	paths := t.getSources(e)
	environment := make([]corev1.EnvVar, 0)

	// combine Environment of integration with context, integration
	// Environment has the priority
	VisitKeyValConfigurations("env", e.Context, e.Integration, func(key string, value string) {
		envvar.SetVal(&environment, key, value)
	})

	// set env vars needed by the runtime
	envvar.SetVal(&environment, "JAVA_MAIN_CLASS", "org.apache.camel.k.jvm.Application")

	// camel-k runtime
	envvar.SetVal(&environment, "CAMEL_K_ROUTES", strings.Join(paths, ","))
	envvar.SetVal(&environment, "CAMEL_K_CONF", "/etc/camel/conf/application.properties")
	envvar.SetVal(&environment, "CAMEL_K_CONF_D", "/etc/camel/conf.d")

	// add a dummy env var to trigger deployment if everything but the code
	// has been changed
	envvar.SetVal(&environment, "CAMEL_K_DIGEST", e.Integration.Status.Digest)

	// optimizations
	envvar.SetVal(&environment, "AB_JOLOKIA_OFF", True)

	// add env vars from traits
	for _, envVar := range e.EnvVars {
		envvar.SetVar(&environment, envVar)
	}

	labels := map[string]string{
		"camel.apache.org/integration": e.Integration.Name,
	}

	annotations := e.Integration.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	// Resolve registry host names when used
	annotations["alpha.image.policy.openshift.io/resolve-names"] = "*"

	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Integration.Name,
			Namespace: e.Integration.Namespace,
			Labels: map[string]string{
				"camel.apache.org/integration": e.Integration.Name,
			},
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: e.Integration.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: e.Integration.Spec.ServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:  e.Integration.Name,
							Image: e.Integration.Status.Image,
							Env:   environment,
						},
					},
				},
			},
		},
	}

	//
	// Volumes :: Setup
	//

	vols := make([]corev1.Volume, 0)
	mnts := make([]corev1.VolumeMount, 0)

	//
	// Volumes :: Properties
	//

	vols = append(vols, corev1.Volume{
		Name: "integration-properties",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: e.Integration.Name + "-properties",
				},
				Items: []corev1.KeyToPath{
					{
						Key:  "properties",
						Path: "application.properties",
					},
				},
			},
		},
	})

	mnts = append(mnts, corev1.VolumeMount{
		Name:      "integration-properties",
		MountPath: "/etc/camel/conf",
	})

	//
	// Volumes :: Sources
	//

	if !t.ContainerImage {
		// We can configure the operator to generate a container images that include
		// integration sources instead of mounting it at runtime and in such case we
		// do not need to mount any 'source' ConfigMap to the pod

		for i, s := range e.Integration.Sources() {
			cmName := fmt.Sprintf("%s-source-%03d", e.Integration.Name, i)
			refName := fmt.Sprintf("integration-source-%03d", i)
			resName := strings.TrimPrefix(s.Name, "/")

			if s.ContentRef != "" {
				cmName = s.ContentRef
			}

			vols = append(vols, corev1.Volume{
				Name: refName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: cmName,
						},
					},
				},
			})

			mnts = append(mnts, corev1.VolumeMount{
				Name:      refName,
				MountPath: path.Join("/etc/camel/sources", resName),
				SubPath:   "content",
			})
		}

		for i, r := range e.Integration.Spec.Resources {
			if r.Type != v1alpha1.ResourceTypeData {
				continue
			}

			cmName := fmt.Sprintf("%s-resource-%03d", e.Integration.Name, i)
			refName := fmt.Sprintf("integration-resource-%03d", i)
			resName := strings.TrimPrefix(r.Name, "/")

			vols = append(vols, corev1.Volume{
				Name: refName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: cmName,
						},
					},
				},
			})

			mnts = append(mnts, corev1.VolumeMount{
				Name:      refName,
				MountPath: path.Join("/etc/camel/resources", resName),
				SubPath:   "content",
			})
		}
	}

	//
	// Volumes :: Additional ConfigMaps
	//

	VisitConfigurations("configmap", e.Context, e.Integration, func(cmName string) {
		refName := kubernetes.SanitizeLabel(cmName)
		fileName := "integration-cm-" + strings.ToLower(cmName)

		vols = append(vols, corev1.Volume{
			Name: refName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cmName,
					},
				},
			},
		})

		mnts = append(mnts, corev1.VolumeMount{
			Name:      refName,
			MountPath: path.Join("/etc/camel/conf.d", fileName),
		})
	})

	//
	// Volumes :: Additional Secrets
	//

	VisitConfigurations("secret", e.Context, e.Integration, func(secretName string) {
		refName := kubernetes.SanitizeLabel(secretName)
		fileName := "integration-secret-" + strings.ToLower(secretName)

		vols = append(vols, corev1.Volume{
			Name: refName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		})

		mnts = append(mnts, corev1.VolumeMount{
			Name:      refName,
			MountPath: path.Join("/etc/camel/conf.d", fileName),
		})
	})

	//
	// Volumes
	//

	deployment.Spec.Template.Spec.Volumes = vols
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = mnts

	return &deployment
}

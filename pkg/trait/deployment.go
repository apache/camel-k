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
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type deploymentTrait struct {
	BaseTrait `property:",squash"`
}

func newDeploymentTrait() *deploymentTrait {
	return &deploymentTrait{
		BaseTrait: newBaseTrait("deployment"),
	}
}

func (d *deploymentTrait) appliesTo(e *Environment) bool {
	return e.Integration != nil && e.Integration.Status.Phase == v1alpha1.IntegrationPhaseDeploying
}

func (d *deploymentTrait) apply(e *Environment) error {
	e.Resources.AddAll(d.getConfigMapFor(e))
	e.Resources.Add(d.getDeploymentFor(e))
	return nil
}

// **********************************
//
// ConfigMap
//
// **********************************

func (*deploymentTrait) getConfigMapFor(e *Environment) []runtime.Object {
	maps := make([]runtime.Object, 0, len(e.Integration.Spec.Sources)+1)

	// combine properties of integration with context, integration
	// properties have the priority
	properties := CombineConfigurationAsMap("property", e.Context, e.Integration)

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
				"properties": PropertiesString(properties),
			},
		},
	)

	for i, s := range e.Integration.Spec.Sources {
		maps = append(
			maps,
			&corev1.ConfigMap{
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
						"camel.apache.org/source.language": string(s.Language),
						"camel.apache.org/source.name":     s.Name,
					},
				},
				Data: map[string]string{
					"integration": s.Content,
				},
			},
		)
	}

	return maps
}

// **********************************
//
// Deployment
//
// **********************************

func (*deploymentTrait) getDeploymentFor(e *Environment) *appsv1.Deployment {
	sources := make([]string, 0, len(e.Integration.Spec.Sources))
	for i, s := range e.Integration.Spec.Sources {
		src := fmt.Sprintf("file:/etc/camel/integrations/%03d/%s", i, strings.TrimPrefix(s.Name, "/"))
		if s.Language != "" {
			src = src + "?language=" + string(s.Language)
		}

		sources = append(sources, src)
	}

	// combine Environment of integration with context, integration
	// Environment has the priority
	environment := CombineConfigurationAsMap("env", e.Context, e.Integration)

	// set env vars needed by the runtime
	environment["JAVA_MAIN_CLASS"] = "org.apache.camel.k.jvm.Application"

	// camel-k runtime
	environment["CAMEL_K_ROUTES"] = strings.Join(sources, ",")
	environment["CAMEL_K_CONF"] = "/etc/camel/conf/application.properties"
	environment["CAMEL_K_CONF_D"] = "/etc/camel/conf.d"

	// add a dummy env var to trigger deployment if everything but the code
	// has been changed
	environment["CAMEL_K_DIGEST"] = e.Integration.Status.Digest

	// optimizations
	environment["AB_JOLOKIA_OFF"] = "true"

	labels := map[string]string{
		"camel.apache.org/integration": e.Integration.Name,
	}
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
			Annotations: e.Integration.Annotations,
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
					Containers: []corev1.Container{
						{
							Name:  e.Integration.Name,
							Image: e.Integration.Status.Image,
							Env:   EnvironmentAsEnvVarSlice(environment),
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
	cnt := 0

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

	for i, s := range e.Integration.Spec.Sources {
		vols = append(vols, corev1.Volume{
			Name: fmt.Sprintf("integration-source-%03d", i),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-source-%03d", e.Integration.Name, i),
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "integration",
							Path: strings.TrimPrefix(s.Name, "/"),
						},
					},
				},
			},
		})

		mnts = append(mnts, corev1.VolumeMount{
			Name:      fmt.Sprintf("integration-source-%03d", i),
			MountPath: fmt.Sprintf("/etc/camel/integrations/%03d", i),
		})
	}

	//
	// Volumes :: Additional ConfigMaps
	//

	cmList := CombineConfigurationAsSlice("configmap", e.Context, e.Integration)
	for _, cmName := range cmList {
		cnt++

		vols = append(vols, corev1.Volume{
			Name: "integration-cm-" + cmName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cmName,
					},
				},
			},
		})

		mnts = append(mnts, corev1.VolumeMount{
			Name:      "integration-cm-" + cmName,
			MountPath: fmt.Sprintf("/etc/camel/conf.d/%03d_%s", cnt, cmName),
		})
	}

	//
	// Volumes :: Additional Secrets
	//

	secretList := CombineConfigurationAsSlice("secret", e.Context, e.Integration)
	for _, secretName := range secretList {
		cnt++

		vols = append(vols, corev1.Volume{
			Name: "integration-secret-" + secretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		})

		mnts = append(mnts, corev1.VolumeMount{
			Name:      "integration-secret-" + secretName,
			MountPath: fmt.Sprintf("/etc/camel/conf.d/%03d_%s", cnt, secretName),
		})
	}

	//
	// Volumes
	//

	deployment.Spec.Template.Spec.Volumes = vols
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = mnts

	return &deployment
}

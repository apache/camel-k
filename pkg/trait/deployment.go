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

	"github.com/apache/camel-k/pkg/util/kubernetes"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type deploymentTrait struct {
	BaseTrait `property:",squash"`
}

func newDeploymentTrait() *deploymentTrait {
	return &deploymentTrait{
		BaseTrait: newBaseTrait("deployment"),
	}
}

func (d *deploymentTrait) customize(environment *environment, resources *kubernetes.Collection) error {
	resources.Add(d.getConfigMapFor(environment))
	resources.Add(d.getDeploymentFor(environment))
	return nil
}

// **********************************
//
// ConfigMap
//
// **********************************

func (*deploymentTrait) getConfigMapFor(e *environment) *corev1.ConfigMap {
	// combine properties of integration with context, integration
	// properties have the priority
	properties := CombineConfigurationAsMap("property", e.Context, e.Integration)

	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Integration.Name,
			Namespace: e.Integration.Namespace,
			Labels: map[string]string{
				"camel.apache.org/integration": e.Integration.Name,
			},
			Annotations: map[string]string{
				"camel.apache.org/source.language": string(e.Integration.Spec.Source.Language),
				"camel.apache.org/source.name":     e.Integration.Spec.Source.Name,
			},
		},
		Data: map[string]string{
			"integration": e.Integration.Spec.Source.Content,
			"properties":  PropertiesString(properties),
		},
	}

	return &cm
}

// **********************************
//
// Deployment
//
// **********************************

func (*deploymentTrait) getDeploymentFor(e *environment) *appsv1.Deployment {
	sourceName := strings.TrimPrefix(e.Integration.Spec.Source.Name, "/")

	// combine environment of integration with context, integration
	// environment has the priority
	environment := CombineConfigurationAsMap("env", e.Context, e.Integration)

	// set env vars needed by the runtime
	environment["JAVA_MAIN_CLASS"] = "org.apache.camel.k.jvm.Application"

	// camel-k runtime
	environment["CAMEL_K_ROUTES_URI"] = "file:/etc/camel/conf/" + sourceName
	environment["CAMEL_K_ROUTES_LANGUAGE"] = string(e.Integration.Spec.Source.Language)
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
	// Volumes :: Defaults
	//

	vols = append(vols, corev1.Volume{
		Name: "integration",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: e.Integration.Name,
				},
				Items: []corev1.KeyToPath{
					{
						Key:  "integration",
						Path: sourceName,
					}, {
						Key:  "properties",
						Path: "application.properties",
					},
				},
			},
		},
	})

	mnts = append(mnts, corev1.VolumeMount{
		Name:      "integration",
		MountPath: "/etc/camel/conf",
	})

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

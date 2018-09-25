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

package integration

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewDeployAction create an action that handles integration deploy
func NewDeployAction() Action {
	return &deployAction{}
}

type deployAction struct {
}

func (action *deployAction) Name() string {
	return "deploy"
}

func (action *deployAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseDeploying
}

func (action *deployAction) Handle(integration *v1alpha1.Integration) error {
	ctx, err := LookupContextForIntegration(integration)
	if err != nil {
		return err
	}
	err = createOrUpdateConfigMap(ctx, integration)
	if err != nil {
		return err
	}
	err = createOrUpdateDeployment(ctx, integration)
	if err != nil {
		return err
	}

	target := integration.DeepCopy()
	logrus.Info("Integration ", target.Name, " transitioning to state ", v1alpha1.IntegrationPhaseRunning)
	target.Status.Phase = v1alpha1.IntegrationPhaseRunning

	return sdk.Update(target)
}

// **********************************
//
// ConfigMap
//
// **********************************

func getConfigMapFor(ctx *v1alpha1.IntegrationContext, integration *v1alpha1.Integration) (*corev1.ConfigMap, error) {
	controller := true
	blockOwnerDeletion := true

	// combine properties of integration with context, integration
	// properties have the priority
	properties := CombineConfigurationAsMap("property", ctx, integration)

	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      integration.Name,
			Namespace: integration.Namespace,
			Labels:    integration.Labels,
			Annotations: map[string]string{
				"camel.apache.org/source.language": string(integration.Spec.Source.Language),
				"camel.apache.org/source.name":     integration.Spec.Source.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         integration.APIVersion,
					Kind:               integration.Kind,
					Name:               integration.Name,
					UID:                integration.UID,
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			},
		},
		Data: map[string]string{
			"integration": integration.Spec.Source.Content,
			"properties":  PropertiesString(properties),
		},
	}

	return &cm, nil
}

func createOrUpdateConfigMap(ctx *v1alpha1.IntegrationContext, integration *v1alpha1.Integration) error {
	cm, err := getConfigMapFor(ctx, integration)
	if err != nil {
		return err
	}

	err = sdk.Create(cm)
	if err != nil && k8serrors.IsAlreadyExists(err) {
		err = sdk.Update(cm)
	}
	if err != nil {
		return errors.Wrap(err, "could not create or replace configmap for integration "+integration.Name)
	}

	return err
}

// **********************************
//
// Deployment
//
// **********************************

func getDeploymentFor(ctx *v1alpha1.IntegrationContext, integration *v1alpha1.Integration) (*appsv1.Deployment, error) {
	controller := true
	blockOwnerDeletion := true
	sourceName := strings.TrimPrefix(integration.Spec.Source.Name, "/")

	// combine environment of integration with context, integration
	// environment has the priority
	environment := CombineConfigurationAsMap("env", ctx, integration)

	// set env vars needed by the runtime
	environment["JAVA_MAIN_CLASS"] = "org.apache.camel.k.jvm.Application"

	// camel-k runtime
	environment["CAMEL_K_ROUTES_URI"] = "file:/etc/camel/conf/" + sourceName
	environment["CAMEL_K_ROUTES_LANGUAGE"] = string(integration.Spec.Source.Language)
	environment["CAMEL_K_CONF"] = "/etc/camel/conf/application.properties"
	environment["CAMEL_K_CONF_D"] = "/etc/camel/conf.d"

	// add a dummy env var to trigger deployment if everything but the code
	// has been changed
	environment["CAMEL_K_DIGEST"] = integration.Status.Digest

	// optimizations
	environment["AB_JOLOKIA_OFF"] = "true"

	labels := map[string]string{
		"camel.apache.org/integration": integration.Name,
	}
	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        integration.Name,
			Namespace:   integration.Namespace,
			Labels:      integration.Labels,
			Annotations: integration.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         integration.APIVersion,
					Kind:               integration.Kind,
					Name:               integration.Name,
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
					UID:                integration.UID,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: integration.Spec.Replicas,
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
							Name:  integration.Name,
							Image: integration.Status.Image,
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
					Name: integration.Name,
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

	cmList := CombineConfigurationAsSlice("configmap", ctx, integration)
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

	secretList := CombineConfigurationAsSlice("secret", ctx, integration)
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

	return &deployment, nil
}

func createOrUpdateDeployment(ctx *v1alpha1.IntegrationContext, integration *v1alpha1.Integration) error {
	deployment, err := getDeploymentFor(ctx, integration)
	if err != nil {
		return err
	}

	err = sdk.Create(deployment)
	if err != nil && k8serrors.IsAlreadyExists(err) {
		err = sdk.Update(deployment)
	}
	if err != nil {
		return errors.Wrap(err, "could not create or replace deployment for integration "+integration.Name)
	}

	return nil
}

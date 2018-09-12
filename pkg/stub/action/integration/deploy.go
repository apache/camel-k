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

package action

import (
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeployAction struct {
}

func NewDeployAction() IntegrationAction {
	return &DeployAction{}
}

func (b *DeployAction) Name() string {
	return "deploy"
}

func (a *DeployAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseDeploying
}

func (a *DeployAction) Handle(integration *v1alpha1.Integration) error {
	if err := createOrUpdateConfigMap(integration); err != nil {
		return err
	}
	if err := createOrUpdateDeployment(integration); err != nil {
		return err
	}

	return nil
}

// **********************************
//
// ConfigMap
//
// **********************************

func getConfigMapFor(integration *v1alpha1.Integration) *corev1.ConfigMap {
	controller := true
	blockOwnerDeletion := true

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      integration.Name,
			Namespace: integration.Namespace,
			Labels:    integration.Labels,
			Annotations: map[string]string{
				"camel.apache.org/source.language": integration.Spec.Source.Language,
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
		},
	}
}

func createOrUpdateConfigMap(integration *v1alpha1.Integration) error {
	cm := getConfigMapFor(integration)

	err := sdk.Create(cm)
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

func getDeploymentFor(integration *v1alpha1.Integration) *appsv1.Deployment {
	controller := true
	blockOwnerDeletion := true
	integrationName := strings.TrimPrefix(integration.Spec.Source.Name, "/")

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
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "integration",
									MountPath: "/etc/camel",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "JAVA_MAIN_CLASS",
									Value: "org.apache.camel.k.jvm.Application",
								},
								{
									Name:  "CAMEL_K_ROUTES_URI",
									Value: "file:/etc/camel/" + integrationName,
								},
								{
									Name:  "CAMEL_K_ROUTES_LANGUAGE",
									Value: integration.Spec.Source.Language,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "integration",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: integration.Name,
									},
									Items: []corev1.KeyToPath{
										{
											Key:  "integration",
											Path: integrationName,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return &deployment
}

func createOrUpdateDeployment(integration *v1alpha1.Integration) error {
	deployment := getDeploymentFor(integration)

	err := sdk.Create(deployment)
	if err != nil && k8serrors.IsAlreadyExists(err) {
		err = sdk.Update(deployment)
	}
	if err != nil {
		return errors.Wrap(err, "could not create or replace deployment for integration "+integration.Name)
	}

	target := integration.DeepCopy()
	target.Status.Phase = v1alpha1.IntegrationPhaseRunning

	return sdk.Update(target)
}

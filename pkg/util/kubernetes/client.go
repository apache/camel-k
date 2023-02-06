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

package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/log"
)

func GetIntegrationPlatform(context context.Context, client ctrl.Reader, name string, namespace string) (*v1.IntegrationPlatform, error) {
	platform := v1.NewIntegrationPlatform(namespace, name)
	log.Debugf("Integration Platform [name: %s], [namespace: %s], [objectkey: %s]", name, namespace, ctrl.ObjectKeyFromObject(&platform))
	if err := client.Get(context, ctrl.ObjectKeyFromObject(&platform), &platform); err != nil {
		log.Debugf("Integration platform Error: %v", err)
		return nil, err
	}

	return &platform, nil
}

func GetIntegrationKit(context context.Context, client ctrl.Reader, name string, namespace string) (*v1.IntegrationKit, error) {
	kit := v1.NewIntegrationKit(namespace, name)
	log.Debugf("Integration Kit [name: %s], [namespace: %s], [objectkey: %s]", name, namespace, ctrl.ObjectKeyFromObject(kit))
	if err := client.Get(context, ctrl.ObjectKeyFromObject(kit), kit); err != nil {
		log.Debugf("Integration kit Error: %v", err)
		return nil, err
	}

	return kit, nil
}

func GetBuild(context context.Context, client client.Client, name string, namespace string) (*v1.Build, error) {
	build := v1.NewBuild(namespace, name)
	log.Debugf("Build [name: %s], [namespace: %s], [objectkey: %s]", name, namespace, ctrl.ObjectKeyFromObject(build))
	if err := client.Get(context, ctrl.ObjectKeyFromObject(build), build); err != nil {
		log.Debugf("Integration build Error: %v", err)
		return nil, err
	}

	return build, nil
}

// GetUnstructured provides a generic unstructured K8S object. Useful in order to retrieve a non cached version of an object.
func GetUnstructured(context context.Context, client ctrl.Reader, gvk schema.GroupVersionKind, name string, namespace string) (*unstructured.Unstructured, error) {
	object := &unstructured.Unstructured{}
	object.SetNamespace(namespace)
	object.SetName(name)
	object.SetGroupVersionKind(gvk)
	err := client.Get(context, ctrl.ObjectKeyFromObject(object), object)

	return object, err
}

func GetConfigMap(context context.Context, client ctrl.Reader, name string, namespace string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := client.Get(context, ctrl.ObjectKeyFromObject(configMap), configMap); err != nil {
		return nil, err
	}

	return configMap, nil
}

func GetSecret(context context.Context, client ctrl.Reader, name string, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := client.Get(context, ctrl.ObjectKeyFromObject(secret), secret); err != nil {
		return nil, err
	}

	return secret, nil
}

// GetSecretRefValue returns the value of a secret in the supplied namespace.
func GetSecretRefValue(ctx context.Context, client ctrl.Reader, namespace string, selector *corev1.SecretKeySelector) (string, error) {
	data, err := GetSecretRefData(ctx, client, namespace, selector)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetSecretRefData returns the value of a secret in the supplied namespace.
func GetSecretRefData(ctx context.Context, client ctrl.Reader, namespace string, selector *corev1.SecretKeySelector) ([]byte, error) {
	secret, err := GetSecret(ctx, client, selector.Name, namespace)
	if err != nil {
		return nil, err
	}

	if data, ok := secret.Data[selector.Key]; ok {
		return data, nil
	}

	return nil, fmt.Errorf("key %s not found in secret %s", selector.Key, selector.Name)
}

// GetSecretsRefData returns the value of the secrets in the supplied namespace.
func GetSecretsRefData(ctx context.Context, client ctrl.Reader, namespace string, selector []corev1.SecretKeySelector) ([][]byte, error) {
	certsData := make([][]byte, len(selector))
	for i := range selector {
		certData, err := GetSecretRefData(ctx, client, namespace, &selector[i])
		if err != nil {
			return nil, err
		}
		certsData[i] = certData
	}
	return certsData, nil
}

// GetConfigMapRefValue returns the value of a configmap in the supplied namespace.
func GetConfigMapRefValue(ctx context.Context, client ctrl.Reader, namespace string, selector *corev1.ConfigMapKeySelector) (string, error) {
	cm, err := GetConfigMap(ctx, client, selector.Name, namespace)
	if err != nil {
		return "", err
	}

	if data, ok := cm.Data[selector.Key]; ok {
		return data, nil
	}

	return "", fmt.Errorf("key %s not found in config map %s", selector.Key, selector.Name)
}

func ResolveValueSource(ctx context.Context, client ctrl.Reader, namespace string, valueSource *v1.ValueSource) (string, error) {
	if valueSource.ConfigMapKeyRef != nil && valueSource.SecretKeyRef != nil {
		return "", fmt.Errorf("value source has bot config map and secret configured")
	}
	if valueSource.ConfigMapKeyRef != nil {
		return GetConfigMapRefValue(ctx, client, namespace, valueSource.ConfigMapKeyRef)
	}
	if valueSource.SecretKeyRef != nil {
		return GetSecretRefValue(ctx, client, namespace, valueSource.SecretKeyRef)
	}

	return "", nil
}

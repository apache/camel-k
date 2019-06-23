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
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"

	yaml2 "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ToJSON --
func ToJSON(value runtime.Object) ([]byte, error) {
	return json.Marshal(value)
}

// ToYAML --
func ToYAML(value runtime.Object) ([]byte, error) {
	data, err := ToJSON(value)
	if err != nil {
		return nil, err
	}

	return JSONToYAML(data)
}

// JSONToYAML --
func JSONToYAML(src []byte) ([]byte, error) {
	jsondata := map[string]interface{}{}
	err := json.Unmarshal(src, &jsondata)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json: %v", err)
	}
	yamldata, err := yaml2.Marshal(&jsondata)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to yaml: %v", err)
	}

	return yamldata, nil
}

// GetConfigMap --
func GetConfigMap(context context.Context, client client.Client, name string, namespace string) (*corev1.ConfigMap, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := client.Get(context, key, &answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// GetSecret --
func GetSecret(context context.Context, client client.Client, name string, namespace string) (*corev1.Secret, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := client.Get(context, key, &answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// GetIntegrationKit --
func GetIntegrationKit(context context.Context, client client.Client, name string, namespace string) (*v1alpha1.IntegrationKit, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := v1alpha1.NewIntegrationKit(namespace, name)

	if err := client.Get(context, key, &answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// GetIntegration --
func GetIntegration(context context.Context, client client.Client, name string, namespace string) (*v1alpha1.Integration, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := v1alpha1.NewIntegration(namespace, name)

	if err := client.Get(context, key, &answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// GetBuild --
func GetBuild(context context.Context, client client.Client, name string, namespace string) (*v1alpha1.Build, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := v1alpha1.NewBuild(namespace, name)

	if err := client.Get(context, key, &answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// GetService --
func GetService(context context.Context, client client.Client, name string, namespace string) (*corev1.Service, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := client.Get(context, key, &answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// GetDiscoveryTypes --
func GetDiscoveryTypes(client client.Client) ([]metav1.TypeMeta, error) {
	resources, err := client.Discovery().ServerPreferredNamespacedResources()
	if err != nil {
		return nil, err
	}

	types := make([]metav1.TypeMeta, 0)
	for _, resource := range resources {
		for _, r := range resource.APIResources {
			types = append(types, metav1.TypeMeta{
				Kind:       r.Kind,
				APIVersion: resource.GroupVersion,
			})
		}
	}

	return types, nil
}

// LookUpResources --
func LookUpResources(ctx context.Context, client client.Client, namespace string, selectors []string) ([]unstructured.Unstructured, error) {
	types, err := GetDiscoveryTypes(client)
	if err != nil {
		return nil, err
	}

	selector, err := labels.Parse(strings.Join(selectors, ","))
	if err != nil {
		return nil, err
	}

	res := make([]unstructured.Unstructured, 0)

	for _, t := range types {
		options := k8sclient.ListOptions{
			Namespace:     namespace,
			LabelSelector: selector,
			Raw: &metav1.ListOptions{
				TypeMeta: t,
			},
		}
		list := unstructured.UnstructuredList{
			Object: map[string]interface{}{
				"apiVersion": t.APIVersion,
				"kind":       t.Kind,
			},
		}
		if err := client.List(ctx, &options, &list); err != nil {
			if k8serrors.IsNotFound(err) ||
				k8serrors.IsForbidden(err) ||
				k8serrors.IsMethodNotSupported(err) {
				continue
			}
			return nil, err
		}

		res = append(res, list.Items...)
	}
	return res, nil
}

// GetSecretRefValue returns the value of a secret in the supplied namespace --
func GetSecretRefValue(ctx context.Context, client client.Client, namespace string, selector *corev1.SecretKeySelector) (string, error) {
	secret, err := GetSecret(ctx, client, selector.Name, namespace)
	if err != nil {
		return "", err
	}

	if data, ok := secret.Data[selector.Key]; ok {
		return string(data), nil
	}

	return "", fmt.Errorf("key %s not found in secret %s", selector.Key, selector.Name)

}

// GetConfigMapRefValue returns the value of a configmap in the supplied namespace
func GetConfigMapRefValue(ctx context.Context, client client.Client, namespace string, selector *corev1.ConfigMapKeySelector) (string, error) {
	cm, err := GetConfigMap(ctx, client, selector.Name, namespace)
	if err != nil {
		return "", err
	}

	if data, ok := cm.Data[selector.Key]; ok {
		return data, nil
	}

	return "", fmt.Errorf("key %s not found in config map %s", selector.Key, selector.Name)
}

// ResolveValueSource --
func ResolveValueSource(ctx context.Context, client client.Client, namespace string, valueSource *v1alpha1.ValueSource) (string, error) {
	if valueSource.ConfigMapKeyRef != nil && valueSource.SecretKeyRef != nil {
		return "", fmt.Errorf("value source has bot config map and secret configuired")
	}
	if valueSource.ConfigMapKeyRef != nil {
		return GetConfigMapRefValue(ctx, client, namespace, valueSource.ConfigMapKeyRef)
	}
	if valueSource.SecretKeyRef != nil {
		return GetSecretRefValue(ctx, client, namespace, valueSource.SecretKeyRef)
	}

	return "", nil
}

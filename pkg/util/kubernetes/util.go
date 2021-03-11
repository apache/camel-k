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
	"regexp"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util"
)

var validTaintRegexp = regexp.MustCompile(`^([\w\/_\-\.]+)(=)?([\w_\-\.]+)?:(NoSchedule|NoExecute|PreferNoSchedule):?(\d*)?$`)

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

	return util.JSONToYAML(data)
}

// GetConfigMap --
func GetConfigMap(context context.Context, client k8sclient.Reader, name string, namespace string) (*corev1.ConfigMap, error) {
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
func GetSecret(context context.Context, client k8sclient.Reader, name string, namespace string) (*corev1.Secret, error) {
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

// GetIntegrationPlatform --
func GetIntegrationPlatform(context context.Context, client k8sclient.Reader, name string, namespace string) (*v1.IntegrationPlatform, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := v1.NewIntegrationPlatform(namespace, name)

	if err := client.Get(context, key, &answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// GetIntegrationKit --
func GetIntegrationKit(context context.Context, client k8sclient.Reader, name string, namespace string) (*v1.IntegrationKit, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := v1.NewIntegrationKit(namespace, name)

	if err := client.Get(context, key, &answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// GetIntegration --
func GetIntegration(context context.Context, client k8sclient.Reader, name string, namespace string) (*v1.Integration, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := v1.NewIntegration(namespace, name)

	if err := client.Get(context, key, &answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// GetBuild --
func GetBuild(context context.Context, client client.Client, name string, namespace string) (*v1.Build, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := v1.NewBuild(namespace, name)

	if err := client.Get(context, key, &answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// GetService --
func GetService(context context.Context, client k8sclient.Reader, name string, namespace string) (*corev1.Service, error) {
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

// GetSecretRefValue returns the value of a secret in the supplied namespace --
func GetSecretRefValue(ctx context.Context, client k8sclient.Reader, namespace string, selector *corev1.SecretKeySelector) (string, error) {
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
func GetConfigMapRefValue(ctx context.Context, client k8sclient.Reader, namespace string, selector *corev1.ConfigMapKeySelector) (string, error) {
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
func ResolveValueSource(ctx context.Context, client k8sclient.Reader, namespace string, valueSource *v1.ValueSource) (string, error) {
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

// GetTolerations build an array of Tolerations from an array of string
func GetTolerations(taints []string) ([]corev1.Toleration, error) {
	tolerations := make([]corev1.Toleration, 0)
	for _, t := range taints {
		if !validTaintRegexp.MatchString(t) {
			return nil, fmt.Errorf("could not match taint %v", t)
		}
		toleration := corev1.Toleration{}
		// Parse the regexp groups
		groups := validTaintRegexp.FindStringSubmatch(t)
		toleration.Key = groups[1]
		if groups[2] != "" {
			toleration.Operator = corev1.TolerationOpEqual
		} else {
			toleration.Operator = corev1.TolerationOpExists
		}
		if groups[3] != "" {
			toleration.Value = groups[3]
		}
		toleration.Effect = corev1.TaintEffect(groups[4])

		if groups[5] != "" {
			tolerationSeconds, err := strconv.ParseInt(groups[5], 10, 64)
			if err != nil {
				return nil, err
			}
			toleration.TolerationSeconds = &tolerationSeconds
		}
		tolerations = append(tolerations, toleration)
	}

	return tolerations, nil
}

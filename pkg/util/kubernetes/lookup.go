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

	"github.com/apache/camel-k/pkg/client"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

// LookupConfigmap will look for any k8s Configmap with a given name in a given namespace.
func LookupConfigmap(ctx context.Context, c client.Client, ns string, name string) *corev1.ConfigMap {
	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}
	key := ctrl.ObjectKey{
		Namespace: ns,
		Name:      name,
	}
	if err := c.Get(ctx, key, &cm); err != nil && k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return nil
	}
	return &cm
}

// LookupSecret will look for any k8s Secret with a given name in a given namespace.
func LookupSecret(ctx context.Context, c client.Client, ns string, name string) *corev1.Secret {
	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}
	key := ctrl.ObjectKey{
		Namespace: ns,
		Name:      name,
	}
	if err := c.Get(ctx, key, &secret); err != nil && k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return nil
	}
	return &secret
}

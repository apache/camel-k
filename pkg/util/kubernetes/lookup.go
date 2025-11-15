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

	"github.com/apache/camel-k/v2/pkg/client"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

// LookupConfigmap will look for any k8s Configmap with a given name in a given namespace.
// Deprecated: won't be supported in future releases.
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

// LookupResourceVersion will look for any k8s resource with a given name in a given namespace, returning its resource version only.
// It makes this safe against any resource that the operator is not allowed to inspect.
func LookupResourceVersion(ctx context.Context, c client.Client, object ctrl.Object) string {
	if err := c.Get(ctx, ctrl.ObjectKeyFromObject(object), object); err != nil {
		return ""
	}

	return object.GetResourceVersion()
}

// LookupSecret will look for any k8s Secret with a given name in a given namespace.
// Deprecated: won't be supported in future releases.
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

// LookupPersistentVolumeClaim will look for any k8s PersistentVolumeClaim with a given name in a given namespace.
func LookupPersistentVolumeClaim(ctx context.Context, c client.Client, ns string, name string) (*corev1.PersistentVolumeClaim, error) {
	pvc := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
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
	if err := c.Get(ctx, key, &pvc); err != nil && k8serrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &pvc, nil
}

// LookupStorageClass will look for any k8s StorageClass with a given name in a given namespace.
func LookupStorageClass(ctx context.Context, c client.Client, ns string, name string) (*storagev1.StorageClass, error) {
	sc := storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: storagev1.SchemeGroupVersion.String(),
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
	if err := c.Get(ctx, key, &sc); err != nil && k8serrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &sc, nil
}

// LookupDefaultStorageClass will look for the default k8s StorageClass in the cluster.
func LookupDefaultStorageClass(ctx context.Context, c client.Client) (*storagev1.StorageClass, error) {
	storageClasses, err := c.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}
	for _, sc := range storageClasses.Items {
		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			return &sc, nil
		}
	}

	return nil, nil
}

// LookupServiceAccount will look for any k8s ServiceAccount with a given name in a given namespace.
func LookupServiceAccount(ctx context.Context, c client.Client, ns string, name string) (*corev1.ServiceAccount, error) {
	sa := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
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
	if err := c.Get(ctx, key, &sa); err != nil && k8serrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &sa, nil
}

// LookupRole will look for any k8s Role with a given name in a given namespace.
func LookupRole(ctx context.Context, c client.Client, ns string, name string) (*rbacv1.Role, error) {
	r := rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
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
	if err := c.Get(ctx, key, &r); err != nil && k8serrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &r, nil
}

// LookupRoleBinding will look for any k8s RoleBinding with a given name in a given namespace.
func LookupRoleBinding(ctx context.Context, c client.Client, ns string, name string) (*rbacv1.RoleBinding, error) {
	rb := rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
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
	if err := c.Get(ctx, key, &rb); err != nil && k8serrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &rb, nil
}

// LookupService will look for any k8s Service with a given name in a given namespace.
func LookupService(ctx context.Context, c client.Client, ns string, name string) (*corev1.Service, error) {
	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
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
	if err := c.Get(ctx, key, &svc); err != nil && k8serrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &svc, nil
}

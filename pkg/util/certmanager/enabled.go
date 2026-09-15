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

package certmanager

import (
	"context"
	"sort"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	kubernetesutil "github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

const (
	// CertManagerAPIGroup is the API group for cert-manager.
	CertManagerAPIGroup = "cert-manager.io"
	// CertManagerAPIVersion is the current API version for cert-manager.
	CertManagerAPIVersion = "v1"

	// AnnotationClusterIssuer is the Ingress annotation to specify a ClusterIssuer.
	AnnotationClusterIssuer = "cert-manager.io/cluster-issuer"
	// AnnotationIssuer is the Ingress annotation to specify a namespaced Issuer.
	AnnotationIssuer = "cert-manager.io/issuer"
)

var (
	// ClusterIssuerGVK is the GroupVersionKind for cert-manager ClusterIssuer.
	ClusterIssuerGVK = schema.GroupVersionKind{
		Group:   CertManagerAPIGroup,
		Version: CertManagerAPIVersion,
		Kind:    "ClusterIssuer",
	}

	// IssuerGVK is the GroupVersionKind for cert-manager Issuer.
	IssuerGVK = schema.GroupVersionKind{
		Group:   CertManagerAPIGroup,
		Version: CertManagerAPIVersion,
		Kind:    "Issuer",
	}
)

func isResourceNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	return k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) || kubernetesutil.IsUnknownAPIError(err)
}

// IsInstalled returns true if connected to a cluster with cert-manager installed.
func IsInstalled(c kubernetes.Interface) (bool, error) {
	_, err := c.Discovery().ServerResourcesForGroupVersion(schema.GroupVersion{
		Group:   CertManagerAPIGroup,
		Version: CertManagerAPIVersion,
	}.String())
	if isResourceNotFoundError(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// ListClusterIssuers returns all ClusterIssuer names available in the cluster.
func ListClusterIssuers(ctx context.Context, c ctrl.Reader) ([]string, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   CertManagerAPIGroup,
		Version: CertManagerAPIVersion,
		Kind:    "ClusterIssuerList",
	})

	if err := c.List(ctx, list); err != nil {
		if isResourceNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	names := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		names = append(names, item.GetName())
	}
	sort.Strings(names)

	return names, nil
}

// GetClusterIssuer checks if a specific ClusterIssuer exists in the cluster.
func GetClusterIssuer(ctx context.Context, c ctrl.Reader, name string) (bool, error) {
	_, err := kubernetesutil.GetUnstructured(ctx, c, ClusterIssuerGVK, name, "")
	if err != nil {
		if isResourceNotFoundError(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// ListIssuers returns all Issuer names available in a specific namespace.
func ListIssuers(ctx context.Context, c ctrl.Reader, namespace string) ([]string, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   CertManagerAPIGroup,
		Version: CertManagerAPIVersion,
		Kind:    "IssuerList",
	})

	if err := c.List(ctx, list, ctrl.InNamespace(namespace)); err != nil {
		if isResourceNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	names := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		names = append(names, item.GetName())
	}
	sort.Strings(names)

	return names, nil
}

// GetIssuer checks if a specific namespaced Issuer exists.
func GetIssuer(ctx context.Context, c ctrl.Reader, namespace, name string) (bool, error) {
	_, err := kubernetesutil.GetUnstructured(ctx, c, IssuerGVK, name, namespace)
	if err != nil {
		if isResourceNotFoundError(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

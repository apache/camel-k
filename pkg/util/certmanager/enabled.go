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
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	certmanagerv1 "github.com/apache/camel-k/v2/pkg/apis/duck/certmanager/v1"
	kubernetesutil "github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

const (
	// AnnotationClusterIssuer is the Ingress annotation to specify a ClusterIssuer.
	AnnotationClusterIssuer = "cert-manager.io/cluster-issuer"
	// AnnotationIssuer is the Ingress annotation to specify a namespaced Issuer.
	AnnotationIssuer = "cert-manager.io/issuer"
)

func isResourceNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	return k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) || kubernetesutil.IsUnknownAPIError(err)
}

// ListClusterIssuers returns all ClusterIssuer names available in the cluster.
func ListClusterIssuers(ctx context.Context, c ctrl.Reader) ([]string, error) {
	list := &certmanagerv1.ClusterIssuerList{}
	if err := c.List(ctx, list); err != nil {
		if isResourceNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	names := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		names = append(names, item.Name)
	}
	sort.Strings(names)

	return names, nil
}

// GetClusterIssuer checks if a specific ClusterIssuer exists in the cluster.
func GetClusterIssuer(ctx context.Context, c ctrl.Reader, name string) (bool, error) {
	err := c.Get(ctx, ctrl.ObjectKey{Name: name}, &certmanagerv1.ClusterIssuer{})
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
	list := &certmanagerv1.IssuerList{}
	if err := c.List(ctx, list, ctrl.InNamespace(namespace)); err != nil {
		if isResourceNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	names := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		names = append(names, item.Name)
	}
	sort.Strings(names)

	return names, nil
}

// GetIssuer checks if a specific namespaced Issuer exists.
func GetIssuer(ctx context.Context, c ctrl.Reader, namespace, name string) (bool, error) {
	err := c.Get(ctx, ctrl.ObjectKey{Name: name, Namespace: namespace}, &certmanagerv1.Issuer{})
	if err != nil {
		if isResourceNotFoundError(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

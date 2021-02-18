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

package platform

import (
	"context"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DefaultPlatformName is the standard name used for the integration platform
	DefaultPlatformName = "camel-k"
)

// GetCurrent returns the currently installed platform (local or global)
func GetCurrent(ctx context.Context, c k8sclient.Reader, namespace string) (*v1.IntegrationPlatform, error) {
	return find(ctx, c, namespace, true)
}

// GetOrFind returns the named platform or any other platform in the local namespace or the global one
func GetOrFind(ctx context.Context, c k8sclient.Reader, namespace string, name string, active bool) (*v1.IntegrationPlatform, error) {
	if name != "" {
		return get(ctx, c, namespace, name)
	}

	return find(ctx, c, namespace, active)
}

// GetOrFindLocal returns the named platform or any other platform in the local namespace
func GetOrFindLocal(ctx context.Context, c k8sclient.Reader, namespace string, name string, active bool) (*v1.IntegrationPlatform, error) {
	if name != "" {
		return kubernetes.GetIntegrationPlatform(ctx, c, name, namespace)
	}

	return findLocal(ctx, c, namespace, active)
}

// get returns the given platform in the given namespace or the global one
func get(ctx context.Context, c k8sclient.Reader, namespace string, name string) (*v1.IntegrationPlatform, error) {
	p, err := kubernetes.GetIntegrationPlatform(ctx, c, name, namespace)
	if err != nil && k8serrors.IsNotFound(err) {
		operatorNamespace := GetOperatorNamespace()
		if operatorNamespace != "" && operatorNamespace != namespace {
			p, err = kubernetes.GetIntegrationPlatform(ctx, c, name, operatorNamespace)
		}
	}
	return p, err
}

// find returns the currently installed platform or any platform existing in local or operator namespace
func find(ctx context.Context, c k8sclient.Reader, namespace string, active bool) (*v1.IntegrationPlatform, error) {
	p, err := findLocal(ctx, c, namespace, active)
	if err != nil && k8serrors.IsNotFound(err) {
		operatorNamespace := GetOperatorNamespace()
		if operatorNamespace != "" && operatorNamespace != namespace {
			p, err = findLocal(ctx, c, operatorNamespace, active)
		}
	}
	return p, err
}

// findLocal returns the currently installed platform or any platform existing in local namespace
func findLocal(ctx context.Context, c k8sclient.Reader, namespace string, active bool) (*v1.IntegrationPlatform, error) {
	lst, err := ListPlatforms(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	for _, platform := range lst.Items {
		platform := platform // pin
		if IsActive(&platform) {
			return &platform, nil
		}
	}

	if !active && len(lst.Items) > 0 {
		// does not require the platform to be active, just return one if present
		res := lst.Items[0]
		return &res, nil
	}

	return nil, k8serrors.NewNotFound(v1.Resource("IntegrationPlatform"), DefaultPlatformName)
}

// ListPlatforms returns all platforms installed in a given namespace (only one will be active)
func ListPlatforms(ctx context.Context, c k8sclient.Reader, namespace string) (*v1.IntegrationPlatformList, error) {
	lst := v1.NewIntegrationPlatformList()
	if err := c.List(ctx, &lst, k8sclient.InNamespace(namespace)); err != nil {
		return nil, err
	}
	return &lst, nil
}

// IsActive determines if the given platform is being used
func IsActive(p *v1.IntegrationPlatform) bool {
	return p.Status.Phase != "" && p.Status.Phase != v1.IntegrationPlatformPhaseDuplicate
}

// GetProfile returns the current profile of the platform (if present) or returns the default one for the cluster
func GetProfile(p *v1.IntegrationPlatform) v1.TraitProfile {
	if p.Status.Profile != "" {
		return p.Status.Profile
	}

	switch p.Status.Cluster {
	case v1.IntegrationPlatformClusterKubernetes:
		return v1.TraitProfileKubernetes
	case v1.IntegrationPlatformClusterOpenShift:
		return v1.TraitProfileOpenShift
	}
	return ""
}

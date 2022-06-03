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
	// DefaultPlatformName is the standard name used for the integration platform.
	DefaultPlatformName = "camel-k"
)

func GetForResource(ctx context.Context, c k8sclient.Reader, o k8sclient.Object) (*v1.IntegrationPlatform, error) {
	return GetOrFindForResource(ctx, c, o, true)
}

func GetOrFindForResource(ctx context.Context, c k8sclient.Reader, o k8sclient.Object, active bool) (*v1.IntegrationPlatform, error) {
	return getOrFindForResource(ctx, c, o, active, false)
}

func GetOrFindLocalForResource(ctx context.Context, c k8sclient.Reader, o k8sclient.Object, active bool) (*v1.IntegrationPlatform, error) {
	return getOrFindForResource(ctx, c, o, active, true)
}

func getOrFindForResource(ctx context.Context, c k8sclient.Reader, o k8sclient.Object, active bool, local bool) (*v1.IntegrationPlatform, error) {
	if selectedPlatform, ok := o.GetAnnotations()[v1.PlatformSelectorAnnotation]; ok {
		return get(ctx, c, o.GetNamespace(), selectedPlatform)
	}
	if it, ok := o.(*v1.Integration); ok {
		return getOrFind(ctx, c, it.Namespace, it.Status.Platform, active, local)
	} else if ik, ok := o.(*v1.IntegrationKit); ok {
		return getOrFind(ctx, c, ik.Namespace, ik.Status.Platform, active, local)
	}
	return find(ctx, c, o.GetNamespace(), active, local)
}

func getOrFind(ctx context.Context, c k8sclient.Reader, namespace string, name string, active bool, local bool) (*v1.IntegrationPlatform, error) {
	if local {
		return getOrFindLocal(ctx, c, namespace, name, active)
	}
	return getOrFindAny(ctx, c, namespace, name, active)
}

// getOrFindAny returns the named platform or any other platform in the local namespace or the global one.
func getOrFindAny(ctx context.Context, c k8sclient.Reader, namespace string, name string, active bool) (*v1.IntegrationPlatform, error) {
	if name != "" {
		return get(ctx, c, namespace, name)
	}

	return findAny(ctx, c, namespace, active)
}

// getOrFindLocal returns the named platform or any other platform in the local namespace.
func getOrFindLocal(ctx context.Context, c k8sclient.Reader, namespace string, name string, active bool) (*v1.IntegrationPlatform, error) {
	if name != "" {
		return kubernetes.GetIntegrationPlatform(ctx, c, name, namespace)
	}

	return findLocal(ctx, c, namespace, active)
}

// get returns the given platform in the given namespace or the global one.
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

func find(ctx context.Context, c k8sclient.Reader, namespace string, active bool, local bool) (*v1.IntegrationPlatform, error) {
	if local {
		return findLocal(ctx, c, namespace, active)
	}
	return findAny(ctx, c, namespace, active)
}

// findAny returns the currently installed platform or any platform existing in local or operator namespace.
func findAny(ctx context.Context, c k8sclient.Reader, namespace string, active bool) (*v1.IntegrationPlatform, error) {
	p, err := findLocal(ctx, c, namespace, active)
	if err != nil && k8serrors.IsNotFound(err) {
		operatorNamespace := GetOperatorNamespace()
		if operatorNamespace != "" && operatorNamespace != namespace {
			p, err = findLocal(ctx, c, operatorNamespace, active)
		}
	}
	return p, err
}

// findLocal returns the currently installed platform or any platform existing in local namespace.
func findLocal(ctx context.Context, c k8sclient.Reader, namespace string, active bool) (*v1.IntegrationPlatform, error) {
	lst, err := ListPrimaryPlatforms(ctx, c, namespace)
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

// ListPrimaryPlatforms returns all non-secondary platforms installed in a given namespace (only one will be active).
func ListPrimaryPlatforms(ctx context.Context, c k8sclient.Reader, namespace string) (*v1.IntegrationPlatformList, error) {
	lst, err := ListAllPlatforms(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	filtered := v1.NewIntegrationPlatformList()
	for _, pl := range lst.Items {
		if !IsSecondary(&pl) {
			filtered.Items = append(filtered.Items, pl)
		}
	}
	return &filtered, nil
}

// ListAllPlatforms returns all platforms installed in a given namespace.
func ListAllPlatforms(ctx context.Context, c k8sclient.Reader, namespace string) (*v1.IntegrationPlatformList, error) {
	lst := v1.NewIntegrationPlatformList()
	if err := c.List(ctx, &lst, k8sclient.InNamespace(namespace)); err != nil {
		return nil, err
	}
	return &lst, nil
}

// IsActive determines if the given platform is being used.
func IsActive(p *v1.IntegrationPlatform) bool {
	return p.Status.Phase != "" && p.Status.Phase != v1.IntegrationPlatformPhaseDuplicate
}

// IsSecondary determines if the given platform is marked as secondary.
func IsSecondary(p *v1.IntegrationPlatform) bool {
	if l, ok := p.Annotations[v1.SecondaryPlatformAnnotation]; ok && l == "true" {
		return true
	}
	return false
}

// GetProfile returns the current profile of the platform (if present) or returns the default one for the cluster.
func GetProfile(p *v1.IntegrationPlatform) v1.TraitProfile {
	if p.Status.Profile != "" {
		return p.Status.Profile
	} else if p.Spec.Profile != "" {
		return p.Spec.Profile
	}

	switch p.Status.Cluster {
	case v1.IntegrationPlatformClusterKubernetes:
		return v1.TraitProfileKubernetes
	case v1.IntegrationPlatformClusterOpenShift:
		return v1.TraitProfileOpenShift
	}
	return ""
}

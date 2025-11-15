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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DefaultPlatformName is the standard name used for the integration platform.
	DefaultPlatformName = "camel-k"
)

// LookupForPlatformName finds integration platform with given operator id as name in any namespace.
func LookupForPlatformName(ctx context.Context, c k8sclient.Reader, name string) (*v1.IntegrationPlatform, error) {
	platformList := v1.NewIntegrationPlatformList()

	// get all integration platform instances on the cluster
	err := c.List(ctx, &platformList)
	if err != nil {
		return nil, err
	}

	// Check if platform with same name as given operator id already exists
	for _, pl := range platformList.Items {
		if pl.Name == name {
			// platform already exists installation not allowed
			return &pl, nil
		}
	}

	return nil, nil
}

func GetForResource(ctx context.Context, c k8sclient.Reader, o k8sclient.Object) (*v1.IntegrationPlatform, error) {
	var ip *v1.IntegrationPlatform
	var err error

	if selectedPlatform, ok := o.GetAnnotations()[v1.PlatformSelectorAnnotation]; ok {
		ip, err = getOrFindAny(ctx, c, o.GetNamespace(), selectedPlatform)
		if err != nil {
			return nil, err
		}
	}

	if ip == nil {
		switch t := o.(type) {
		case *v1.Integration:
			ip, err = getOrFindAny(ctx, c, t.Namespace, t.Status.Platform)
			if err != nil {
				return nil, err
			}
		case *v1.IntegrationKit:
			ip, err = getOrFindAny(ctx, c, t.Namespace, t.Status.Platform)
			if err != nil {
				return nil, err
			}
		}
	}

	if ip == nil {
		ip, err = findAny(ctx, c, o.GetNamespace())
		if err != nil {
			return nil, err
		}
	}

	return ip, nil
}
func GetForName(ctx context.Context, c k8sclient.Reader, namespace string, name string) (*v1.IntegrationPlatform, error) {
	return getOrFindAny(ctx, c, namespace, name)
}

func GetOrFindLocal(ctx context.Context, c k8sclient.Reader, namespace string) (*v1.IntegrationPlatform, error) {
	return findLocal(ctx, c, namespace)
}

// getOrFindAny returns the named platform or any other platform in the local namespace or the global one.
func getOrFindAny(ctx context.Context, c k8sclient.Reader, namespace string, name string) (*v1.IntegrationPlatform, error) {
	if name != "" {
		pl, err := get(ctx, c, namespace, name)
		if pl != nil {
			return pl, err
		}
	}

	return findAny(ctx, c, namespace)
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

// findAny returns the currently installed platform or any platform existing in local or operator namespace.
func findAny(ctx context.Context, c k8sclient.Reader, namespace string) (*v1.IntegrationPlatform, error) {
	p, err := findLocal(ctx, c, namespace)
	if err != nil && k8serrors.IsNotFound(err) {
		operatorNamespace := GetOperatorNamespace()
		if operatorNamespace != "" && operatorNamespace != namespace {
			p, err = findLocal(ctx, c, operatorNamespace)
		}
	}

	return p, err
}

// findLocal returns the currently installed platform or any platform existing in local namespace.
func findLocal(ctx context.Context, c k8sclient.Reader, namespace string) (*v1.IntegrationPlatform, error) {
	log.Debugf("Finding available platforms in namespace %s", namespace)

	operatorNamespace := GetOperatorNamespace()
	if namespace == operatorNamespace {
		operatorID := defaults.OperatorID()
		if operatorID != "" {
			if p, err := get(ctx, c, operatorNamespace, operatorID); err == nil {
				log.Debugf("Found integration platform %s for operator %s in namespace %s", operatorID, operatorID, operatorNamespace)

				return p, nil
			}
		}
	}

	lst, err := ListPlatforms(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	var fallback *v1.IntegrationPlatform
	for i := range lst.Items {
		platform := lst.Items[i]
		if IsActive(&platform) {
			log.Debugf("Found active integration platform %s in namespace %s", platform.Name, namespace)

			return &platform, nil
		} else {
			fallback = &platform
		}
	}

	if fallback != nil {
		log.Debugf("Found inactive integration platform %s in namespace %s", fallback.Name, namespace)

		return fallback, nil
	}

	log.Debugf("Unable to find integration platform in namespace %s", namespace)

	return nil, k8serrors.NewNotFound(v1.Resource("IntegrationPlatform"), DefaultPlatformName)
}

// ListPlatforms returns all platforms installed in a given namespace.
func ListPlatforms(ctx context.Context, c k8sclient.Reader, namespace string) (*v1.IntegrationPlatformList, error) {
	lst := v1.NewIntegrationPlatformList()
	if err := c.List(ctx, &lst, k8sclient.InNamespace(namespace)); err != nil {
		return nil, err
	}

	return &lst, nil
}

// IsActive determines if the given platform is being used.
func IsActive(p *v1.IntegrationPlatform) bool {
	return p.Status.Phase != v1.IntegrationPlatformPhaseNone
}

// GetTraitProfile returns the current profile of the platform (if present) or returns the default one for the cluster.
func GetTraitProfile(p *v1.IntegrationPlatform) v1.TraitProfile {
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

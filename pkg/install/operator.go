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

package install

import (
	"context"
	"errors"

	"github.com/apache/camel-k/deploy"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/knative"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/minishift"
	"github.com/apache/camel-k/pkg/util/openshift"
)

// Operator installs the operator resources in the given namespace
func Operator(ctx context.Context, c client.Client, namespace string) error {
	return OperatorOrCollect(ctx, c, namespace, nil)
}

// OperatorOrCollect installs the operator resources or adds them to the collector if present
func OperatorOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection) error {
	isOpenshift, err := openshift.IsOpenShift(c)
	if err != nil {
		return err
	}
	if isOpenshift {
		if err := installOpenshift(ctx, c, namespace, collection); err != nil {
			return err
		}
	} else {
		if err := installKubernetes(ctx, c, namespace, collection); err != nil {
			return err
		}
	}
	// Additionally, install Knative resources (roles and bindings)
	isKnative, err := knative.IsInstalled(ctx, c)
	if err != nil {
		return err
	}
	if isKnative {
		return installKnative(ctx, c, namespace, collection)
	}
	return nil
}

func installOpenshift(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection) error {
	return ResourcesOrCollect(ctx, c, namespace, collection,
		"operator-service-account.yaml",
		"operator-role-openshift.yaml",
		"operator-role-binding.yaml",
		"operator-deployment-openshift.yaml",
		"operator-service.yaml",
	)
}

func installKubernetes(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection) error {
	return ResourcesOrCollect(ctx, c, namespace, collection,
		"operator-service-account.yaml",
		"operator-role-kubernetes.yaml",
		"operator-role-binding.yaml",
		"builder-pvc.yaml",
		"operator-deployment-kubernetes.yaml",
		"operator-service.yaml",
	)
}

func installKnative(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection) error {
	return ResourcesOrCollect(ctx, c, namespace, collection,
		"operator-role-knative.yaml",
		"operator-role-binding-knative.yaml",
	)
}

// Platform installs the platform custom resource
func Platform(ctx context.Context, c client.Client, namespace string, registry v1alpha1.IntegrationPlatformRegistrySpec) (*v1alpha1.IntegrationPlatform, error) {
	return PlatformOrCollect(ctx, c, namespace, registry, nil)
}

// PlatformOrCollect --
// nolint: lll
func PlatformOrCollect(ctx context.Context, c client.Client, namespace string, registry v1alpha1.IntegrationPlatformRegistrySpec, collection *kubernetes.Collection) (*v1alpha1.IntegrationPlatform, error) {
	isOpenshift, err := openshift.IsOpenShift(c)
	if err != nil {
		return nil, err
	}
	platformObject, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), deploy.Resources["platform-cr.yaml"])
	if err != nil {
		return nil, err
	}
	pl := platformObject.(*v1alpha1.IntegrationPlatform)

	if !isOpenshift {

		pl.Spec.Build.Registry = registry

		// Kubernetes only (Minikube)
		if registry.Address == "" {
			// This operation should be done here in the installer
			// because the operator is not allowed to look into the "kube-system" namespace
			minikubeRegistry, err := minishift.FindRegistry(ctx, c)
			if err != nil {
				return nil, err
			}
			if minikubeRegistry == nil {
				return nil, errors.New("cannot find automatically a registry where to push images")
			}

			pl.Spec.Build.Registry.Address = *minikubeRegistry
			pl.Spec.Build.Registry.Insecure = true
		}
	}

	var knativeInstalled bool
	if knativeInstalled, err = knative.IsInstalled(ctx, c); err != nil {
		return nil, err
	}
	if knativeInstalled {
		pl.Spec.Profile = v1alpha1.TraitProfileKnative
	}

	return pl, nil
}

// Example --
func Example(ctx context.Context, c client.Client, namespace string) error {
	return ExampleOrCollect(ctx, c, namespace, nil)
}

// ExampleOrCollect --
func ExampleOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection) error {
	return ResourcesOrCollect(ctx, c, namespace, collection,
		"cr-example.yaml",
	)
}

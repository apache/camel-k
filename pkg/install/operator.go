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
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/rbac/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/apache/camel-k/deploy"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/envvar"
	"github.com/apache/camel-k/pkg/util/knative"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/minishift"
)

// OperatorConfiguration --
type OperatorConfiguration struct {
	CustomImage string
	Namespace   string
	Global      bool
	ClusterType string
}

// Operator installs the operator resources in the given namespace
func Operator(ctx context.Context, c client.Client, cfg OperatorConfiguration, force bool) error {
	return OperatorOrCollect(ctx, c, cfg, nil, force)
}

// OperatorOrCollect installs the operator resources or adds them to the collector if present
func OperatorOrCollect(ctx context.Context, c client.Client, cfg OperatorConfiguration, collection *kubernetes.Collection, force bool) error {
	customizer := func(o runtime.Object) runtime.Object {
		if cfg.CustomImage != "" {
			if d, ok := o.(*appsv1.Deployment); ok {
				if d.Labels["camel.apache.org/component"] == "operator" {
					d.Spec.Template.Spec.Containers[0].Image = cfg.CustomImage
				}
			}
		}

		if cfg.Global {
			if d, ok := o.(*appsv1.Deployment); ok {
				if d.Labels["camel.apache.org/component"] == "operator" {
					// Make the operator watch all namespaces
					envvar.SetVal(&d.Spec.Template.Spec.Containers[0].Env, "WATCH_NAMESPACE", "")
				}
			}

			// Turn Role & RoleBinding into their equivalent cluster types
			if r, ok := o.(*v1beta1.Role); ok {
				if strings.HasPrefix(r.Name, "camel-k-operator") {
					o = &v1beta1.ClusterRole{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: cfg.Namespace,
							Name:      r.Name,
						},
						Rules: r.Rules,
					}
				}
			}

			if rb, ok := o.(*v1beta1.RoleBinding); ok {
				if strings.HasPrefix(rb.Name, "camel-k-operator") {
					rb.Subjects[0].Namespace = cfg.Namespace

					o = &v1beta1.ClusterRoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: cfg.Namespace,
							Name:      rb.Name,
						},
						Subjects: rb.Subjects,
						RoleRef: v1beta1.RoleRef{
							APIGroup: rb.RoleRef.APIGroup,
							Kind:     "ClusterRole",
							Name:     rb.RoleRef.Name,
						},
					}
				}
			}
		}
		return o
	}

	isOpenshift, err := isOpenShift(c, cfg.ClusterType)
	if err != nil {
		return err
	}
	if isOpenshift {
		if err := installOpenshift(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
			return err
		}
	} else {
		if err := installKubernetes(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
			return err
		}
	}
	// Additionally, install Knative resources (roles and bindings)
	isKnative, err := knative.IsInstalled(ctx, c)
	if err != nil {
		return err
	}
	if isKnative {
		return installKnative(ctx, c, cfg.Namespace, customizer, collection, force)
	}

	if errevt := installEvents(ctx, c, cfg.Namespace, customizer, collection, force); errevt != nil {
		if k8serrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Println("Warning: the operator will not be able to publish Kubernetes events. Try installing as cluster-admin to allow it to generate events.")
	}

	return nil
}

func installOpenshift(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"operator-service-account.yaml",
		"operator-role-openshift.yaml",
		"operator-role-binding.yaml",
		"operator-deployment.yaml",
	)
}

func installKubernetes(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"operator-service-account.yaml",
		"operator-role-kubernetes.yaml",
		"operator-role-binding.yaml",
		"operator-deployment.yaml",
	)
}

func installKnative(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"operator-role-knative.yaml",
		"operator-role-binding-knative.yaml",
	)
}

func installEvents(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"operator-role-events.yaml",
		"operator-role-binding-events.yaml",
	)
}

// Platform installs the platform custom resource
// nolint: lll
func Platform(ctx context.Context, c client.Client, clusterType string, namespace string, registry v1.IntegrationPlatformRegistrySpec) (*v1.IntegrationPlatform, error) {
	return PlatformOrCollect(ctx, c, clusterType, namespace, registry, nil)
}

// PlatformOrCollect --
// nolint: lll
func PlatformOrCollect(ctx context.Context, c client.Client, clusterType string, namespace string, registry v1.IntegrationPlatformRegistrySpec, collection *kubernetes.Collection) (*v1.IntegrationPlatform, error) {
	isOpenshift, err := isOpenShift(c, clusterType)
	if err != nil {
		return nil, err
	}
	platformObject, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), deploy.ResourceAsString("platform-cr.yaml"))
	if err != nil {
		return nil, err
	}
	pl := platformObject.(*v1.IntegrationPlatform)

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

	return pl, nil
}

// Example --
func Example(ctx context.Context, c client.Client, namespace string, force bool) error {
	return ExampleOrCollect(ctx, c, namespace, nil, force)
}

// ExampleOrCollect --
func ExampleOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, IdentityResourceCustomizer,
		"cr-example.yaml",
	)
}

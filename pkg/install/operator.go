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
	"fmt"
	"strings"

	"github.com/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/apache/camel-k/deploy"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/envvar"
	"github.com/apache/camel-k/pkg/util/knative"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/minikube"
)

// OperatorConfiguration --
type OperatorConfiguration struct {
	CustomImage           string
	CustomImagePullPolicy string
	Namespace             string
	Global                bool
	ClusterType           string
	Health                OperatorHealthConfiguration
	Monitoring            OperatorMonitoringConfiguration
}

// OperatorHealthConfiguration --
type OperatorHealthConfiguration struct {
	Port int32
}

// OperatorMonitoringConfiguration --
type OperatorMonitoringConfiguration struct {
	Enabled bool
	Port    int32
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

		if cfg.CustomImagePullPolicy != "" {
			if d, ok := o.(*appsv1.Deployment); ok {
				if d.Labels["camel.apache.org/component"] == "operator" {
					d.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullPolicy(cfg.CustomImagePullPolicy)
				}
			}
		}

		if d, ok := o.(*appsv1.Deployment); ok {
			if d.Labels["camel.apache.org/component"] == "operator" {
				// Metrics endpoint port
				d.Spec.Template.Spec.Containers[0].Args = append(d.Spec.Template.Spec.Containers[0].Args,
					fmt.Sprintf("--monitoring-port=%d", cfg.Monitoring.Port))
				d.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort = cfg.Monitoring.Port
				// Health endpoint port
				d.Spec.Template.Spec.Containers[0].Args = append(d.Spec.Template.Spec.Containers[0].Args,
					fmt.Sprintf("--health-port=%d", cfg.Health.Port))
				d.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Port = intstr.FromInt(int(cfg.Health.Port))
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
			if r, ok := o.(*rbacv1.Role); ok {
				if strings.HasPrefix(r.Name, "camel-k-operator") {
					o = &rbacv1.ClusterRole{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: cfg.Namespace,
							Name:      r.Name,
							Labels: map[string]string{
								"app": "camel-k",
							},
						},
						Rules: r.Rules,
					}
				}
			}

			if rb, ok := o.(*rbacv1.RoleBinding); ok {
				if strings.HasPrefix(rb.Name, "camel-k-operator") {
					rb.Subjects[0].Namespace = cfg.Namespace

					o = &rbacv1.ClusterRoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: cfg.Namespace,
							Name:      rb.Name,
							Labels: map[string]string{
								"app": "camel-k",
							},
						},
						Subjects: rb.Subjects,
						RoleRef: rbacv1.RoleRef{
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

	isOpenShift, err := isOpenShift(c, cfg.ClusterType)
	if err != nil {
		return err
	}
	if isOpenShift {
		if err := installOpenShift(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
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
		if err := installKnative(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
			return err
		}
	}

	if errevt := installEvents(ctx, c, cfg.Namespace, customizer, collection, force); errevt != nil {
		if k8serrors.IsAlreadyExists(errevt) {
			return errevt
		}
		fmt.Println("Warning: the operator will not be able to publish Kubernetes events. Try installing as cluster-admin to allow it to generate events.")
	}

	if errmtr := installServiceMonitors(ctx, c, cfg.Namespace, customizer, collection, force); errmtr != nil {
		if k8serrors.IsAlreadyExists(errmtr) {
			return errmtr
		}
		fmt.Println("Warning: the operator will not be able to create servicemonitors for metrics. Try installing as cluster-admin to allow the creation of servicemonitors.")
	}

	if errmtr := installStrimziBindings(ctx, c, cfg.Namespace, customizer, collection, force); errmtr != nil {
		if k8serrors.IsAlreadyExists(errmtr) {
			return errmtr
		}
		fmt.Println("Warning: the operator will not be able to lookup strimzi kafka resources. Try installing as cluster-admin to allow the lookup of strimzi kafka resources.")
	}

	if errmtr := installLeaseBindings(ctx, c, cfg.Namespace, customizer, collection, force); errmtr != nil {
		if k8serrors.IsAlreadyExists(errmtr) {
			return errmtr
		}
		fmt.Println("Warning: the operator will not be able to perform locking using Camel \"master\" component. Try installing as cluster-admin to allow management of lease resources.")
	}

	if cfg.Monitoring.Enabled {
		if err := installMonitoringResources(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
			if k8serrors.IsForbidden(err) {
				fmt.Println("Warning: the creation of monitoring resources is not allowed. Try installing as cluster-admin to allow the creation of monitoring resources.")
			} else if meta.IsNoMatchError(errors.Cause(err)) {
				fmt.Println("Warning: the creation of the monitoring resources failed: ", err)
			} else {
				return err
			}
		}
	}

	return nil
}

func installOpenShift(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
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

func installServiceMonitors(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"operator-role-servicemonitors.yaml",
		"operator-role-binding-servicemonitors.yaml",
	)
}

func installStrimziBindings(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"operator-role-strimzi.yaml",
		"operator-role-binding-strimzi.yaml",
	)
}

func installMonitoringResources(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"operator-pod-monitor.yaml",
		"operator-prometheus-rule.yaml",
	)
}

func installLeaseBindings(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"operator-role-leases.yaml",
		"operator-role-binding-leases.yaml",
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
	isOpenShift, err := isOpenShift(c, clusterType)
	if err != nil {
		return nil, err
	}
	platformObject, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), deploy.ResourceAsString("platform-cr.yaml"))
	if err != nil {
		return nil, err
	}
	pl := platformObject.(*v1.IntegrationPlatform)

	if !isOpenShift {
		pl.Spec.Build.Registry = registry

		// Kubernetes only (Minikube)
		if registry.Address == "" {
			// This operation should be done here in the installer
			// because the operator is not allowed to look into the "kube-system" namespace
			address, err := minikube.FindRegistry(ctx, c)
			if err != nil {
				return nil, err
			}
			if address == nil {
				return nil, errors.New("cannot find automatically a registry where to push images")
			}

			pl.Spec.Build.Registry.Address = *address
			pl.Spec.Build.Registry.Insecure = true
			if pl.Spec.Build.PublishStrategy == "" {
				// Use spectrum in insecure dev clusters by default
				pl.Spec.Build.PublishStrategy = v1.IntegrationPlatformBuildPublishStrategySpectrum
			}
		}
	}

	return pl, nil
}

// ExampleOrCollect --
func ExampleOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, IdentityResourceCustomizer,
		"cr-example.yaml",
	)
}

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

	"github.com/spf13/cobra"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kubectl/pkg/cmd/set/env"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/resources"
	"github.com/apache/camel-k/v2/pkg/util/envvar"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/minikube"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
	"github.com/apache/camel-k/v2/pkg/util/patch"
)

type OperatorConfiguration struct {
	CustomImage           string
	CustomImagePullPolicy string
	Namespace             string
	Global                bool
	ClusterType           string
	Health                OperatorHealthConfiguration
	Monitoring            OperatorMonitoringConfiguration
	Debugging             OperatorDebuggingConfiguration
	Tolerations           []string
	NodeSelectors         []string
	ResourcesRequirements []string
	EnvVars               []string
}

type OperatorHealthConfiguration struct {
	Port int32
}

type OperatorDebuggingConfiguration struct {
	Enabled bool
	Port    int32
	Path    string
}

type OperatorMonitoringConfiguration struct {
	Enabled bool
	Port    int32
}

// OperatorStorageConfiguration represents the configuration required for Camel K operator storage.
type OperatorStorageConfiguration struct {
	Enabled    bool
	ClassName  string
	Capacity   string
	AccessMode string
}

// OperatorOrCollect installs the operator resources or adds them to the collector if present.
//
//nolint:maintidx,goconst
func OperatorOrCollect(ctx context.Context, cmd *cobra.Command, c client.Client, cfg OperatorConfiguration, collection *kubernetes.Collection, force bool) error {
	isOpenShift, err := isOpenShift(c, cfg.ClusterType)
	if err != nil {
		return err
	}

	customizer := func(o ctrl.Object) ctrl.Object {
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

		if cfg.Tolerations != nil {
			if d, ok := o.(*appsv1.Deployment); ok {
				if d.Labels["camel.apache.org/component"] == "operator" {
					tolerations, err := kubernetes.NewTolerations(cfg.Tolerations)
					if err != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), "Warning: could not parse the configured tolerations!")
					}
					d.Spec.Template.Spec.Tolerations = tolerations
				}
			}
		}

		if cfg.ResourcesRequirements != nil {
			if d, ok := o.(*appsv1.Deployment); ok {
				if d.Labels["camel.apache.org/component"] == "operator" {
					resourceReq, err := kubernetes.NewResourceRequirements(cfg.ResourcesRequirements)
					if err != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), "Warning: could not parse the configured resources requests!")
					}
					for i := range len(d.Spec.Template.Spec.Containers) {
						d.Spec.Template.Spec.Containers[i].Resources = resourceReq
					}
				}
			}
		}

		if cfg.EnvVars != nil {
			if d, ok := o.(*appsv1.Deployment); ok {
				if d.Labels["camel.apache.org/component"] == "operator" {
					envVars, _, _, err := env.ParseEnv(cfg.EnvVars, nil)
					if err != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), "Warning: could not parse environment variables!")
					}
					for i := range len(d.Spec.Template.Spec.Containers) {
						for _, envVar := range envVars {
							envvar.SetVar(&d.Spec.Template.Spec.Containers[i].Env, envVar)
						}
					}
				}
			}
		}

		if cfg.NodeSelectors != nil {
			if d, ok := o.(*appsv1.Deployment); ok {
				if d.Labels["camel.apache.org/component"] == "operator" {
					nodeSelector, err := kubernetes.NewNodeSelectors(cfg.NodeSelectors)
					if err != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), "Warning: could not parse the configured node selectors!")
					}
					d.Spec.Template.Spec.NodeSelector = nodeSelector
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
		if cfg.Debugging.Enabled {
			if d, ok := o.(*appsv1.Deployment); ok {
				if d.Labels["camel.apache.org/component"] == "operator" {
					d.Spec.Template.Spec.Containers[0].Command = []string{"dlv",
						fmt.Sprintf("--listen=:%d", cfg.Debugging.Port), "--headless=true", "--api-version=2",
						"exec", cfg.Debugging.Path, "--", "operator", "--leader-election=false"}
					d.Spec.Template.Spec.Containers[0].Ports = append(d.Spec.Template.Spec.Containers[0].Ports, corev1.ContainerPort{
						Name:          "delve",
						ContainerPort: cfg.Debugging.Port,
					})
					// In debug mode, the Liveness probe must be removed otherwise K8s will consider the pod as dead
					// while debugging
					d.Spec.Template.Spec.Containers[0].LivenessProbe = nil
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
			// Configure subject on ClusterRoleBindings
			if crb, ok := o.(*rbacv1.ClusterRoleBinding); ok {
				if strings.HasPrefix(crb.Name, "camel-k-operator") {
					crb.ObjectMeta.Name = fmt.Sprintf("%s-%s", crb.ObjectMeta.Name, cfg.Namespace)
					bound := false
					for i, subject := range crb.Subjects {
						if subject.Name == "camel-k-operator" {
							if subject.Namespace == cfg.Namespace {
								bound = true
								break
							} else if subject.Namespace == "" || subject.Namespace == "placeholder" {
								crb.Subjects[i].Namespace = cfg.Namespace
								bound = true
								break
							}
						}
					}

					if !bound {
						crb.Subjects = append(crb.Subjects, rbacv1.Subject{
							Kind:      "ServiceAccount",
							Namespace: cfg.Namespace,
							Name:      "camel-k-operator",
						})
					}
				}
			}
		}

		if isOpenShift {
			// Remove Ingress permissions as it's not needed on OpenShift
			// This should ideally be removed from the common RBAC manifest.
			RemoveIngressRoleCustomizer(o)

			if d, ok := o.(*appsv1.Deployment); ok {
				securityContext, _ := openshift.GetOpenshiftSecurityContextRestricted(ctx, c, cfg.Namespace)
				if securityContext != nil {
					d.Spec.Template.Spec.Containers[0].SecurityContext = securityContext

				} else {
					d.Spec.Template.Spec.Containers[0].SecurityContext = kubernetes.DefaultOperatorSecurityContext()
				}
			}
		}

		return o
	}

	// Install Kubernetes RBAC resources (roles and bindings)
	if err := installKubernetesRoles(ctx, c, cfg.Namespace, customizer, collection, force, cfg.Global); err != nil {
		return err
	}

	// Install OpenShift RBAC resources if needed (roles and bindings)
	if isOpenShift {
		if err := installOpenShiftRoles(ctx, c, cfg.Namespace, customizer, collection, force, cfg.Global); err != nil {
			return err
		}
		if err := installClusterRoleBinding(ctx, c, collection, cfg.Namespace, "camel-k-operator-console-openshift", "/config/rbac/openshift/operator-cluster-role-console-binding-openshift.yaml"); err != nil {
			if k8serrors.IsForbidden(err) {
				fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to manage ConsoleCLIDownload resources. Try installing the operator as cluster-admin.")
			} else {
				return err
			}
		}
	}

	// Deploy the operator
	if err := installOperator(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
		return err
	}

	if err = installEvents(ctx, c, cfg.Namespace, customizer, collection, force, cfg.Global); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to publish Kubernetes events. Try installing as cluster-admin to allow it to generate events.")
	}

	if err = installKnativeBindings(ctx, c, cfg.Namespace, customizer, collection, force, cfg.Global); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to create Knative resources. Try installing as cluster-admin.")
	}

	if err = installKedaBindings(ctx, c, cfg.Namespace, customizer, collection, force, cfg.Global); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to create KEDA resources. Try installing as cluster-admin.")
	}

	if err = installPodMonitors(ctx, c, cfg.Namespace, customizer, collection, force, cfg.Global); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to create PodMonitor resources. Try installing as cluster-admin.")
	}

	if err := installStrimziBindings(ctx, c, cfg.Namespace, customizer, collection, force, cfg.Global); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to lookup Strimzi Kafka resources. Try installing as cluster-admin to allow the lookup of strimzi kafka resources.")
	}

	if err = installLeaseBindings(ctx, c, cfg.Namespace, customizer, collection, force, cfg.Global); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to create Leases. Try installing as cluster-admin to allow management of Lease resources.")
	}

	if err = installNamespacedRoleBinding(ctx, c, collection, cfg.Namespace, "/config/rbac/operator-role-binding-local-registry.yaml"); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: the operator may not be able to detect a local image registry (%s)\n", err.Error())
		}
	}

	if cfg.Monitoring.Enabled {
		if err := installMonitoringResources(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
			switch {
			case k8serrors.IsForbidden(err):
				fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the creation of monitoring resources is not allowed. Try installing as cluster-admin to allow the creation of monitoring resources.")
				// TU to check ?
			case meta.IsNoMatchError(errors.Unwrap(err)):
				fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the creation of the monitoring resources failed: ", err)
			default:
				return err
			}
		}
	}

	return nil
}

func installNamespacedRoleBinding(ctx context.Context, c client.Client, collection *kubernetes.Collection, namespace string, path string) error {
	yaml, err := resources.ResourceAsString(path)
	if err != nil {
		return err
	}
	if yaml == "" {
		return fmt.Errorf("resource file %v not found", path)
	}
	obj, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), yaml)
	if err != nil {
		return err
	}
	//nolint:forcetypeassert
	target := obj.(*rbacv1.RoleBinding)

	bound := false
	for i, subject := range target.Subjects {
		if subject.Name == "camel-k-operator" {
			if subject.Namespace == namespace {
				bound = true
				break
			} else if subject.Namespace == "" || subject.Namespace == "placeholder" {
				target.Subjects[i].Namespace = namespace
				bound = true
				break
			}
		}
	}

	if !bound {
		target.Subjects = append(target.Subjects, rbacv1.Subject{
			Kind:      "ServiceAccount",
			Namespace: namespace,
			Name:      "camel-k-operator",
		})
	}

	if collection != nil {
		collection.Add(target)
		return nil
	}

	return c.Create(ctx, target)
}

func installClusterRoleBinding(ctx context.Context, c client.Client, collection *kubernetes.Collection, namespace string, name string, path string) error {
	var target *rbacv1.ClusterRoleBinding
	existing, err := c.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	switch {
	case k8serrors.IsNotFound(err):
		content, err := resources.ResourceAsString(path)
		if err != nil {
			return err
		}

		existing = nil
		if content == "" {
			return fmt.Errorf("resource file %v not found", path)
		}

		obj, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), content)
		if err != nil {
			return err
		}
		var ok bool
		if target, ok = obj.(*rbacv1.ClusterRoleBinding); !ok {
			return fmt.Errorf("file %v does not contain a ClusterRoleBinding resource", path)
		}
	case err != nil:
		return err
	default:
		target = existing.DeepCopy()
	}

	bound := false
	for i, subject := range target.Subjects {
		if subject.Name == serviceAccountName {
			if subject.Namespace == namespace {
				bound = true

				break
			} else if subject.Namespace == "" || subject.Namespace == "placeholder" {
				target.Subjects[i].Namespace = namespace
				bound = true

				break
			}
		}
	}

	if !bound {
		target.Subjects = append(target.Subjects, rbacv1.Subject{
			Kind:      "ServiceAccount",
			Namespace: namespace,
			Name:      serviceAccountName,
		})
	}

	if collection != nil {
		collection.Add(target)
		return nil
	}

	if existing == nil {
		return c.Create(ctx, target)
	}

	// The ClusterRoleBinding.Subjects field does not have a patchStrategy key in its field tag,
	// so a strategic merge patch would use the default patch strategy, which is replace.
	// Let's compute a simple JSON merge patch from the existing resource, and patch it.
	p, err := patch.MergePatch(existing, target)
	if err != nil {
		return err
	} else if len(p) == 0 {
		// Avoid triggering a patch request for nothing
		return nil
	}

	return c.Patch(ctx, existing, ctrl.RawPatch(types.MergePatchType, p))
}

func installOpenShiftRoles(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool, global bool) error {
	if global {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/descoped/operator-cluster-role-openshift.yaml",
			"/config/rbac/descoped/operator-cluster-role-binding-openshift.yaml",
		)
	} else {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/namespaced/operator-role-openshift.yaml",
			"/config/rbac/namespaced/operator-role-binding-openshift.yaml",
		)
	}
}

func installKubernetesRoles(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool, global bool) error {
	if global {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/manager/operator-service-account.yaml",
			"/config/rbac/descoped/operator-cluster-role.yaml",
			"/config/rbac/descoped/operator-cluster-role-binding.yaml",
		)
	} else {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/manager/operator-service-account.yaml",
			"/config/rbac/namespaced/operator-role.yaml",
			"/config/rbac/namespaced/operator-role-binding.yaml",
		)
	}
}

func installOperator(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"/config/manager/operator-deployment.yaml",
	)
}

func installKnativeBindings(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool, global bool) error {
	if global {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/descoped/operator-cluster-role-knative.yaml",
			"/config/rbac/descoped/operator-cluster-role-binding-knative.yaml",
		)
	} else {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/namespaced/operator-role-knative.yaml",
			"/config/rbac/namespaced/operator-role-binding-knative.yaml",
		)
	}
}

func installKedaBindings(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool, global bool) error {
	if global {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/descoped/operator-cluster-role-keda.yaml",
			"/config/rbac/descoped/operator-cluster-role-binding-keda.yaml",
		)
	} else {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/namespaced/operator-role-keda.yaml",
			"/config/rbac/namespaced/operator-role-binding-keda.yaml",
		)
	}
}

func installEvents(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool, global bool) error {
	if global {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/descoped/operator-cluster-role-events.yaml",
			"/config/rbac/descoped/operator-cluster-role-binding-events.yaml",
		)
	} else {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/namespaced/operator-role-events.yaml",
			"/config/rbac/namespaced/operator-role-binding-events.yaml",
		)
	}
}

func installPodMonitors(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool, global bool) error {
	if global {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/descoped/operator-cluster-role-podmonitors.yaml",
			"/config/rbac/descoped/operator-cluster-role-binding-podmonitors.yaml",
		)
	} else {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/namespaced/operator-role-podmonitors.yaml",
			"/config/rbac/namespaced/operator-role-binding-podmonitors.yaml",
		)
	}
}

func installStrimziBindings(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool, global bool) error {
	if global {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/descoped/operator-cluster-role-strimzi.yaml",
			"/config/rbac/descoped/operator-cluster-role-binding-strimzi.yaml",
		)
	} else {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/namespaced/operator-role-strimzi.yaml",
			"/config/rbac/namespaced/operator-role-binding-strimzi.yaml",
		)
	}
}

func installMonitoringResources(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"/config/prometheus/operator-pod-monitor.yaml",
		"/config/prometheus/operator-prometheus-rule.yaml",
	)
}

func installLeaseBindings(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool, global bool) error {
	if global {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/descoped/operator-cluster-role-leases.yaml",
			"/config/rbac/descoped/operator-cluster-role-binding-leases.yaml",
		)
	} else {
		return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
			"/config/rbac/namespaced/operator-role-leases.yaml",
			"/config/rbac/namespaced/operator-role-binding-leases.yaml",
		)
	}
}

// NewPlatform creates a new IntegrationPlatform instance.
func NewPlatform(
	ctx context.Context, c client.Client,
	clusterType string, skipRegistrySetup bool, registry v1.RegistrySpec, operatorID string,
) (*v1.IntegrationPlatform, error) {
	isOpenShift, err := isOpenShift(c, clusterType)
	if err != nil {
		return nil, err
	}

	content, err := resources.ResourceAsString("/config/samples/bases/camel_v1_integrationplatform.yaml")
	if err != nil {
		return nil, err
	}

	platformObject, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), content)
	if err != nil {
		return nil, err
	}

	pl, ok := platformObject.(*v1.IntegrationPlatform)
	if !ok {
		return nil, fmt.Errorf("type assertion failed: %v", platformObject)
	}

	if operatorID != "" {
		// We must tell the operator to reconcile this IntegrationPlatform
		pl.SetOperatorID(operatorID)
		pl.Name = operatorID
	}

	if !skipRegistrySetup {
		// Let's apply registry settings whether it's OpenShift or not
		// Some OpenShift variants such as Microshift might not have a built-in registry
		pl.Spec.Build.Registry = registry

		if !isOpenShift && registry.Address == "" {
			// This operation should be done here in the installer
			// because the operator is not allowed to look into the "kube-system" namespace
			address, err := minikube.FindRegistry(ctx, c)
			if err != nil {
				return nil, err
			}

			if address == nil || *address == "" {
				return nil, errors.New("cannot find a registry where to push images")
			}

			pl.Spec.Build.Registry.Address = *address
			pl.Spec.Build.Registry.Insecure = true
			if pl.Spec.Build.PublishStrategy == "" {
				pl.Spec.Build.PublishStrategy = v1.IntegrationPlatformBuildPublishStrategyJib
			}
		}
	}

	return pl, nil
}

func ExampleOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, IdentityResourceCustomizer,
		"/config/samples/bases/camel_v1_integration.yaml",
	)
}

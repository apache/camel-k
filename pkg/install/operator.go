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
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/envvar"
	"github.com/apache/camel-k/v2/pkg/util/knative"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/minikube"
	"github.com/apache/camel-k/v2/pkg/util/patch"
	image "github.com/apache/camel-k/v2/pkg/util/registry"
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
	Storage               OperatorStorageConfiguration
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
// nolint: maintidx // TODO: refactor the code
func OperatorOrCollect(ctx context.Context, cmd *cobra.Command, c client.Client, cfg OperatorConfiguration, collection *kubernetes.Collection, force bool) error {
	isOpenShift, err := isOpenShift(c, cfg.ClusterType)
	if err != nil {
		return err
	}

	var camelKPVC *corev1.PersistentVolumeClaim
	if cfg.Storage.Enabled {
		camelKPVC, err = installPVC(ctx, cmd, c, cfg, collection)
		if err != nil {
			return err
		}
	}

	customizer := func(o ctrl.Object) ctrl.Object {
		if camelKPVC != nil {
			if d, ok := o.(*appsv1.Deployment); ok {
				if d.Labels["camel.apache.org/component"] == "operator" {
					volume := corev1.Volume{
						Name: defaults.DefaultPVC,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: camelKPVC.Name,
							},
						},
					}
					if d.Spec.Template.Spec.Volumes == nil {
						d.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0, 1)
					}
					d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, volume)

					vm := corev1.VolumeMount{
						MountPath: defaults.LocalRepository,
						Name:      volume.Name,
					}
					if d.Spec.Template.Spec.Containers[0].VolumeMounts == nil {
						d.Spec.Template.Spec.Containers[0].VolumeMounts = make([]corev1.VolumeMount, 0, 1)
					}
					d.Spec.Template.Spec.Containers[0].VolumeMounts = append(d.Spec.Template.Spec.Containers[0].VolumeMounts, vm)
				}
			}
		}

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
					for i := 0; i < len(d.Spec.Template.Spec.Containers); i++ {
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
					for i := 0; i < len(d.Spec.Template.Spec.Containers); i++ {
						d.Spec.Template.Spec.Containers[i].Env = append(d.Spec.Template.Spec.Containers[i].Env, envVars...)
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
			var ugfid int64 = 0
			d.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				FSGroup: &ugfid,
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
							Name:      fmt.Sprintf("%s-%s", rb.Name, cfg.Namespace),
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

		if isOpenShift {
			// Remove Ingress permissions as it's not needed on OpenShift
			// This should ideally be removed from the common RBAC manifest.
			RemoveIngressRoleCustomizer(o)
		}

		return o
	}

	// Install Kubernetes RBAC resources (roles and bindings)
	if err := installKubernetesRoles(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
		return err
	}

	// Install OpenShift RBAC resources if needed (roles and bindings)
	if isOpenShift {
		if err := installOpenShiftRoles(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
			return err
		}
		if err := installClusterRoleBinding(ctx, c, collection, cfg.Namespace, "camel-k-operator-console-openshift", "/rbac/openshift/operator-cluster-role-console-binding-openshift.yaml"); err != nil {
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

	// Additionally, install Knative resources (roles and bindings)
	isKnative, err := knative.IsInstalled(c)
	if err != nil {
		return err
	}
	if isKnative {
		if err := installKnative(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
			return err
		}
		if err := installClusterRoleBinding(ctx, c, collection, cfg.Namespace, "camel-k-operator-bind-addressable-resolver", "/rbac/operator-cluster-role-binding-addressable-resolver.yaml"); err != nil {
			if k8serrors.IsForbidden(err) {
				fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to bind Knative addressable-resolver ClusterRole. Try installing the operator as cluster-admin.")
			} else {
				return err
			}
		}
	}

	if err = installEvents(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to publish Kubernetes events. Try installing as cluster-admin to allow it to generate events.")
	}

	if err = installKedaBindings(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to create KEDA resources. Try installing as cluster-admin.")
	}

	if err = installPodMonitors(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to create PodMonitor resources. Try installing as cluster-admin.")
	}

	if err := installStrimziBindings(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to lookup strimzi kafka resources. Try installing as cluster-admin to allow the lookup of strimzi kafka resources.")
	}

	if err = installLeaseBindings(ctx, c, cfg.Namespace, customizer, collection, force); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to create Leases. Try installing as cluster-admin to allow management of Lease resources.")
	}

	if err = installClusterRoleBinding(ctx, c, collection, cfg.Namespace, "camel-k-operator-custom-resource-definitions", "/rbac/operator-cluster-role-binding-custom-resource-definitions.yaml"); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: the operator will not be able to get CustomResourceDefinitions resources and the service-binding trait will fail if used. Try installing the operator as cluster-admin.")
	}

	if err = installNamespacedRoleBinding(ctx, c, collection, cfg.Namespace, "/rbac/operator-role-binding-local-registry.yaml"); err != nil {
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

func installPVC(ctx context.Context, cmd *cobra.Command, c client.Client, cfg OperatorConfiguration, collection *kubernetes.Collection) (*corev1.PersistentVolumeClaim, error) {
	// Verify if a PVC already exists
	camelKPVC, err := kubernetes.LookupPersistentVolumeClaim(ctx, c, cfg.Namespace, defaults.DefaultPVC)
	if err != nil {
		return nil, err
	}
	if camelKPVC != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "A persistent volume claim for \"%s\" already exist, reusing it\n", defaults.DefaultPVC)
		return camelKPVC, nil
	}

	// Use a dynamic volume based on storage classes
	storageClassName, err := getStorageClassName(ctx, c, cfg.Storage.ClassName)
	if err != nil {
		return nil, err
	}
	if storageClassName != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Using storage class \"%s\" to create \"%s\" volume for the operator\n", storageClassName, defaults.DefaultPVC)
		camelKPVC = kubernetes.NewPersistentVolumeClaim(
			cfg.Namespace,
			defaults.DefaultPVC,
			storageClassName,
			cfg.Storage.Capacity,
			corev1.PersistentVolumeAccessMode(cfg.Storage.AccessMode),
		)
		err = ObjectOrCollect(ctx, c, cfg.Namespace, collection, false, camelKPVC)
		return camelKPVC, err
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Could not find a default storage class in the cluster. The operator will be installed with an ephemeral storage. Bear in mind certain build strategies such as \"pod\" may not work as expected.\n")
	return nil, nil
}

func getStorageClassName(ctx context.Context, c client.Client, cfgStorageClassName string) (string, error) {
	if cfgStorageClassName != "" {
		return cfgStorageClassName, nil
	}
	defaultStorageClass, err := kubernetes.LookupDefaultStorageClass(ctx, c)
	if err != nil {
		return "", err
	}
	return defaultStorageClass.Name, nil
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
	// nolint: forcetypeassert
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
			} else if subject.Namespace == "" {
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

func installOpenShiftRoles(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"/rbac/openshift/operator-role-openshift.yaml",
		"/rbac/openshift/operator-role-binding-openshift.yaml",
	)
}

func installKubernetesRoles(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"/manager/operator-service-account.yaml",
		"/rbac/operator-role.yaml",
		"/rbac/operator-role-binding.yaml",
	)
}

func installOperator(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"/manager/operator-deployment.yaml",
	)
}

func installKedaBindings(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"/rbac/operator-role-keda.yaml",
		"/rbac/operator-role-binding-keda.yaml",
	)
}

func installKnative(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"/rbac/operator-role-knative.yaml",
		"/rbac/operator-role-binding-knative.yaml",
	)
}

func installEvents(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"/rbac/operator-role-events.yaml",
		"/rbac/operator-role-binding-events.yaml",
	)
}

func installPodMonitors(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"/rbac/operator-role-podmonitors.yaml",
		"/rbac/operator-role-binding-podmonitors.yaml",
	)
}

func installStrimziBindings(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"/rbac/operator-role-strimzi.yaml",
		"/rbac/operator-role-binding-strimzi.yaml",
	)
}

func installMonitoringResources(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"/prometheus/operator-pod-monitor.yaml",
		"/prometheus/operator-prometheus-rule.yaml",
	)
}

func installLeaseBindings(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, customizer,
		"/rbac/operator-role-leases.yaml",
		"/rbac/operator-role-binding-leases.yaml",
	)
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

	content, err := resources.ResourceAsString("/samples/bases/camel_v1_integrationplatform.yaml")
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
			if address == nil {
				// try KEP-1755
				address, err = image.GetRegistryAddress(ctx, c)
				if err != nil {
					return nil, err
				}
			}

			if address == nil || *address == "" {
				return nil, errors.New("cannot find a registry where to push images")
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

func ExampleOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection, force bool) error {
	return ResourcesOrCollect(ctx, c, namespace, collection, force, IdentityResourceCustomizer,
		"/samples/bases/camel_v1_integration.yaml",
	)
}

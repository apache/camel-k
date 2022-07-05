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

package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kamelets"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/resource"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var namedConfRegExp = regexp.MustCompile("([a-z0-9-.]+)/.*")

// newCmdPromote --.
func newCmdPromote(rootCmdOptions *RootCmdOptions) (*cobra.Command, *promoteCmdOptions) {
	options := promoteCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "promote integration --to [namespace] ...",
		Short:   "Promote an Integration/KameletBinding from an environment to another",
		Long:    "Promote an Integration/KameletBinding from an environment to another, for example from a Development environment to a Production environment",
		PreRunE: decode(&options),
		RunE:    options.run,
	}

	cmd.Flags().String("to", "", "The namespace where to promote the Integration")

	return &cmd, &options
}

type promoteCmdOptions struct {
	*RootCmdOptions
	To string `mapstructure:"to" yaml:",omitempty"`
}

func (o *promoteCmdOptions) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("promote expects an Integration/KameletBinding name argument")
	}
	if o.To == "" {
		return errors.New("promote expects a destination namespace as --to argument")
	}
	return nil
}

func (o *promoteCmdOptions) run(cmd *cobra.Command, args []string) error {
	if err := o.validate(cmd, args); err != nil {
		return err
	}

	name := args[0]
	c, err := o.GetCmdClient()
	if err != nil {
		return errors.Wrap(err, "could not retrieve cluster client")
	}

	opSource, err := operatorInfo(o.Context, c, o.Namespace)
	if err != nil {
		return errors.Wrap(err, "could not retrieve info for Camel K operator source")
	}
	opDest, err := operatorInfo(o.Context, c, o.To)
	if err != nil {
		return errors.Wrap(err, "could not retrieve info for Camel K operator destination")
	}

	err = checkOpsCompatibility(cmd, opSource, opDest)
	if err != nil {
		return errors.Wrap(err, "could not verify operators compatibility")
	}
	promoteKameletBinding := false
	var sourceIntegration *v1.Integration
	// We first look if a KameletBinding with the name exists
	sourceKameletBinding, err := o.getKameletBinding(c, name)
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrap(err, "problems looking for KameletBinding "+name)
	}
	if sourceKameletBinding != nil {
		promoteKameletBinding = true
	}
	sourceIntegration, err = o.getIntegration(c, name)
	if err != nil {
		return errors.Wrap(err, "could not get Integration "+name)
	}
	if sourceIntegration.Status.Phase != v1.IntegrationPhaseRunning {
		return fmt.Errorf("could not promote an Integration in %s status", sourceIntegration.Status.Phase)
	}
	err = o.validateDestResources(c, sourceIntegration)
	if err != nil {
		return errors.Wrap(err, "could not validate destination resources")
	}
	if promoteKameletBinding {
		// KameletBinding promotion
		destKameletBinding := o.editKameletBinding(sourceKameletBinding, sourceIntegration)

		return c.Create(o.Context, destKameletBinding)
	}
	// Plain Integration promotion
	destIntegration := o.editIntegration(sourceIntegration)

	// Ensure the destination namespace has access to the source namespace images
	err = addSystemPullerRoleBinding(o.Context, c, sourceIntegration.Namespace, destIntegration.Namespace)
	if err != nil {
		return err
	}

	return c.Create(o.Context, destIntegration)
}

func checkOpsCompatibility(cmd *cobra.Command, source, dest map[string]string) error {
	if !compatibleVersions(source["Version"], dest["Version"], cmd) {
		return fmt.Errorf("source (%s) and destination (%s) Camel K operator versions are not compatible", source["Version"], dest["Version"])
	}
	if !compatibleVersions(source["Runtime Version"], dest["Runtime Version"], cmd) {
		return fmt.Errorf("source (%s) and destination (%s) Camel K runtime versions are not compatible", source["Runtime Version"], dest["Runtime Version"])
	}
	if source["Registry Address"] != dest["Registry Address"] {
		return fmt.Errorf("source (%s) and destination (%s) Camel K container images registries are not the same", source["Registry Address"], dest["Registry Address"])
	}

	return nil
}

func (o *promoteCmdOptions) getKameletBinding(c client.Client, name string) (*v1alpha1.KameletBinding, error) {
	it := v1alpha1.NewKameletBinding(o.Namespace, name)
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: o.Namespace,
	}
	if err := c.Get(o.Context, key, &it); err != nil {
		return nil, err
	}

	return &it, nil
}

func (o *promoteCmdOptions) getIntegration(c client.Client, name string) (*v1.Integration, error) {
	it := v1.NewIntegration(o.Namespace, name)
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: o.Namespace,
	}
	if err := c.Get(o.Context, key, &it); err != nil {
		return nil, err
	}

	return &it, nil
}

func (o *promoteCmdOptions) validateDestResources(c client.Client, it *v1.Integration) error {
	var configmaps []string
	var secrets []string
	var pvcs []string
	var kamelets []string

	// Mount trait
	mount, err := toPropertyMap(it.Spec.Traits.Mount)
	if err != nil {
		return err
	}
	for t, v := range mount {
		switch t {
		case "configs":
			if list, ok := v.([]string); ok {
				for _, cn := range list {
					if conf, parseErr := resource.ParseConfig(cn); parseErr == nil {
						if conf.StorageType() == resource.StorageTypeConfigmap {
							configmaps = append(configmaps, conf.Name())
						} else if conf.StorageType() == resource.StorageTypeSecret {
							secrets = append(secrets, conf.Name())
						}
					} else {
						return parseErr
					}
				}
			}
		case "resources":
			if list, ok := v.([]string); ok {
				for _, cn := range list {
					if conf, parseErr := resource.ParseResource(cn); parseErr == nil {
						if conf.StorageType() == resource.StorageTypeConfigmap {
							configmaps = append(configmaps, conf.Name())
						} else if conf.StorageType() == resource.StorageTypeSecret {
							secrets = append(secrets, conf.Name())
						}
					} else {
						return parseErr
					}
				}
			}
		case "volumes":
			if list, ok := v.([]string); ok {
				for _, cn := range list {
					if conf, parseErr := resource.ParseVolume(cn); parseErr == nil {
						if conf.StorageType() == resource.StorageTypePVC {
							pvcs = append(pvcs, conf.Name())
						}
					} else {
						return parseErr
					}
				}
			}
		}
	}

	// OpenAPI trait
	openapi, err := toPropertyMap(it.Spec.Traits.OpenAPI)
	if err != nil {
		return err
	}
	for k, v := range openapi {
		if k != "configmaps" {
			continue
		}
		if list, ok := v.([]string); ok {
			configmaps = append(configmaps, list...)
			break
		}
	}

	// Kamelets trait
	kamelet, err := toPropertyMap(it.Spec.Traits.Kamelets)
	if err != nil {
		return err
	}
	if list, ok := kamelet["list"].(string); ok {
		kamelets = strings.Split(list, ",")
	}
	sourceKamelets, err := o.listKamelets(c, it)
	if err != nil {
		return err
	}
	kamelets = append(kamelets, sourceKamelets...)

	anyError := false
	var errorTrace string
	for _, name := range configmaps {
		if !existsCm(o.Context, c, name, o.To) {
			anyError = true
			errorTrace += fmt.Sprintf("Configmap %s is missing from %s namespace\n", name, o.To)
		}
	}
	for _, name := range secrets {
		if !existsSecret(o.Context, c, name, o.To) {
			anyError = true
			errorTrace += fmt.Sprintf("Secret %s is missing from %s namespace\n", name, o.To)
		}
	}
	for _, name := range pvcs {
		if !existsPv(o.Context, c, name, o.To) {
			anyError = true
			errorTrace += fmt.Sprintf("PersistentVolume %s is missing from %s namespace\n", name, o.To)
		}
	}
	for _, name := range kamelets {
		if !existsKamelet(o.Context, c, name, o.To) {
			anyError = true
			errorTrace += fmt.Sprintf("Kamelet %s is missing from %s namespace\n", name, o.To)
		}
	}

	if anyError {
		return fmt.Errorf(errorTrace)
	}

	return nil
}

func toPropertyMap(src interface{}) (map[string]interface{}, error) {
	propMap, err := trait.ToPropertyMap(src)
	if err != nil {
		return nil, err
	}
	// Migrate legacy configuration properties before promoting
	if err := trait.MigrateLegacyConfiguration(propMap); err != nil {
		return nil, err
	}

	return propMap, nil
}

func (o *promoteCmdOptions) listKamelets(c client.Client, it *v1.Integration) ([]string, error) {
	catalog, err := camel.DefaultCatalog()
	if err != nil {
		return nil, err
	}
	kamelets, err := kamelets.ExtractKameletFromSources(o.Context, c, catalog, &kubernetes.Collection{}, it)
	if err != nil {
		return nil, err
	}

	var filtered []string
	for _, k := range kamelets {
		// We must remove any default source/sink
		if k == "source" || k == "sink" {
			continue
		}

		// We must drop any named configurations
		match := namedConfRegExp.FindStringSubmatch(k)
		if len(match) > 0 {
			filtered = append(filtered, match[1])
		} else {
			filtered = append(filtered, k)
		}
	}

	return filtered, nil
}

func existsCm(ctx context.Context, c client.Client, name string, namespace string) bool {
	var obj corev1.ConfigMap
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
	if err := c.Get(ctx, key, &obj); err != nil {
		return false
	}

	return true
}

func existsSecret(ctx context.Context, c client.Client, name string, namespace string) bool {
	var obj corev1.Secret
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
	if err := c.Get(ctx, key, &obj); err != nil {
		return false
	}

	return true
}

func existsPv(ctx context.Context, c client.Client, name string, namespace string) bool {
	var obj corev1.PersistentVolume
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
	if err := c.Get(ctx, key, &obj); err != nil {
		return false
	}

	return true
}

func existsKamelet(ctx context.Context, c client.Client, name string, namespace string) bool {
	var obj v1alpha1.Kamelet
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
	if err := c.Get(ctx, key, &obj); err != nil {
		return false
	}

	return true
}

func (o *promoteCmdOptions) editIntegration(it *v1.Integration) *v1.Integration {
	dst := v1.NewIntegration(o.To, it.Name)
	contImage := it.Status.Image
	dst.Spec = *it.Spec.DeepCopy()
	if dst.Spec.Traits.Container == nil {
		dst.Spec.Traits.Container = &traitv1.ContainerTrait{}
	}
	dst.Spec.Traits.Container.Image = contImage
	return &dst
}

func (o *promoteCmdOptions) editKameletBinding(kb *v1alpha1.KameletBinding, it *v1.Integration) *v1alpha1.KameletBinding {
	dst := v1alpha1.NewKameletBinding(o.To, kb.Name)
	dst.Spec = *kb.Spec.DeepCopy()
	contImage := it.Status.Image
	if dst.Spec.Integration == nil {
		dst.Spec.Integration = &v1.IntegrationSpec{}
	}
	if dst.Spec.Integration.Traits.Container == nil {
		dst.Spec.Integration.Traits.Container = &traitv1.ContainerTrait{}
	}
	dst.Spec.Integration.Traits.Container.Image = contImage
	if dst.Spec.Source.Ref != nil {
		dst.Spec.Source.Ref.Namespace = o.To
	}
	if dst.Spec.Sink.Ref != nil {
		dst.Spec.Sink.Ref.Namespace = o.To
	}
	if dst.Spec.Steps != nil {
		for _, step := range dst.Spec.Steps {
			if step.Ref != nil {
				step.Ref.Namespace = o.To
			}
		}
	}
	return &dst
}

//
// RoleBinding is required to allow access to images in one namespace
// by another namespace. Without this on rbac-enabled clusters, the
// image cannot be pulled.
//
func addSystemPullerRoleBinding(ctx context.Context, c client.Client, sourceNS string, destNS string) error {
	rb := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-image-puller", destNS),
			Namespace: sourceNS,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: destNS,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "system:image-puller",
		},
	}
	applier := c.ServerOrClientSideApplier()
	err := applier.Apply(ctx, rb)

	return err
}

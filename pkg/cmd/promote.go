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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kamelets"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/resource"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

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
	var traits map[string][]string
	var configmaps []string
	var secrets []string
	var pvcs []string
	var kamelets []string
	// Mount trait
	mounts := it.Spec.Traits.Mount
	if err := json.Unmarshal(mounts.Configuration.RawMessage, &traits); err != nil {
		return err
	}
	for t, v := range traits {
		switch t {
		case "configs":
			for _, c := range v {
				if conf, parseErr := resource.ParseConfig(c); parseErr == nil {
					if conf.StorageType() == resource.StorageTypeConfigmap {
						configmaps = append(configmaps, conf.Name())
					} else if conf.StorageType() == resource.StorageTypeSecret {
						secrets = append(secrets, conf.Name())
					}
				} else {
					return parseErr
				}
			}
		case "resources":
			for _, c := range v {
				if conf, parseErr := resource.ParseResource(c); parseErr == nil {
					if conf.StorageType() == resource.StorageTypeConfigmap {
						configmaps = append(configmaps, conf.Name())
					} else if conf.StorageType() == resource.StorageTypeSecret {
						secrets = append(secrets, conf.Name())
					}
				} else {
					return parseErr
				}
			}
		case "volumes":
			for _, c := range v {
				if conf, parseErr := resource.ParseVolume(c); parseErr == nil {
					if conf.StorageType() == resource.StorageTypePVC {
						pvcs = append(pvcs, conf.Name())
					}
				} else {
					return parseErr
				}
			}
		}
	}
	// Openapi trait
	openapis := it.Spec.Traits.OpenAPI
	if err := json.Unmarshal(openapis.Configuration.RawMessage, &traits); err != nil {
		return err
	}
	for k, v := range traits {
		for _, c := range v {
			if k == "configmaps" {
				configmaps = append(configmaps, c)
			}
		}
	}
	// Kamelet trait
	kameletTrait := it.Spec.Traits.Kamelets
	var kameletListTrait map[string]string
	if err := json.Unmarshal(kameletTrait.Configuration.RawMessage, &kameletListTrait); err != nil {
		return err
	}
	kamelets = strings.Split(kameletListTrait["list"], ",")
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

func (o *promoteCmdOptions) listKamelets(c client.Client, it *v1.Integration) ([]string, error) {
	catalog, err := camel.DefaultCatalog()
	if err != nil {
		return nil, err
	}
	kamelets, err := kamelets.ExtractKameletFromSources(o.Context, c, catalog, &kubernetes.Collection{}, it)
	if err != nil {
		return nil, err
	}

	// We must remove any default source/sink
	var filtered []string
	for _, k := range kamelets {
		if k != "source" && k != "sink" {
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

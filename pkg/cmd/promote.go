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
	"errors"
	"fmt"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kamelets"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/resource"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// newCmdPromote --.
func newCmdPromote(rootCmdOptions *RootCmdOptions) (*cobra.Command, *promoteCmdOptions) {
	options := promoteCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "promote integration --to [namespace] ...",
		Short:   "Promote an Integration from an environment to another",
		Long:    "Promote an Integration from an environment to another, for example from a Development environment to a Production environment",
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
		return errors.New("promote expects an Integration name argument")
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

	it := args[0]
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}

	opSource, err := operatorInfo(o.Context, c, o.Namespace)
	if err != nil {
		return fmt.Errorf("could not retrieve info for Camel K operator source")
	}
	opDest, err := operatorInfo(o.Context, c, o.To)
	if err != nil {
		return fmt.Errorf("could not retrieve info for Camel K operator source")
	}

	err = checkOpsCompatibility(cmd, opSource, opDest)
	if err != nil {
		return err
	}
	sourceIntegration, err := o.getIntegration(c, it)
	if err != nil {
		return err
	}
	if sourceIntegration.Status.Phase != v1.IntegrationPhaseRunning {
		return fmt.Errorf("could not promote an integration in %s status", sourceIntegration.Status.Phase)
	}
	err = o.validateDestResources(c, sourceIntegration)
	if err != nil {
		return err
	}
	destIntegration, err := o.editIntegration(sourceIntegration)
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

func (o *promoteCmdOptions) getIntegration(c client.Client, name string) (*v1.Integration, error) {
	it := v1.NewIntegration(o.Namespace, name)
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: o.Namespace,
	}
	if err := c.Get(o.Context, key, &it); err != nil {
		return nil, fmt.Errorf("could not find integration %s in namespace %s", it.Name, o.Namespace)
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
	mounts := it.Spec.Traits["mount"]
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
	openapis := it.Spec.Traits["openapi"]
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
	kameletTrait := it.Spec.Traits["kamelets"]
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

func (o *promoteCmdOptions) editIntegration(it *v1.Integration) (*v1.Integration, error) {
	dst := v1.NewIntegration(o.To, it.Name)
	contImage := it.Status.Image
	dst.Spec = *it.Spec.DeepCopy()
	if dst.Spec.Traits == nil {
		dst.Spec.Traits = map[string]v1.TraitSpec{}
	}
	editedContTrait, err := editContainerImage(dst.Spec.Traits["container"], contImage)
	dst.Spec.Traits["container"] = editedContTrait
	return &dst, err
}

func editContainerImage(contTrait v1.TraitSpec, image string) (v1.TraitSpec, error) {
	var editedTrait v1.TraitSpec
	m := make(map[string]map[string]interface{})
	data, err := json.Marshal(contTrait)
	if err != nil {
		return editedTrait, err
	}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return editedTrait, err
	}
	// We must initialize, if it was not initialized so far
	if m["configuration"] == nil {
		m["configuration"] = make(map[string]interface{})
	}
	m["configuration"]["image"] = image
	newData, err := json.Marshal(m)
	if err != nil {
		return editedTrait, err
	}
	err = json.Unmarshal(newData, &editedTrait)

	return editedTrait, err
}

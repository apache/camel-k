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
	"os"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/source"
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
		Use:     "promote integration -to [namespace] ...",
		Short:   "Promote an Integration from an environment to another",
		Long:    "Promote an Integration from an environment to another, for example from a Development environment to a Production environment",
		Aliases: []string{"cp", "mv"},
		Args:    options.validate,
		PreRunE: decode(&options),
		RunE:    options.run,
	}

	cmd.Flags().StringP("to", "", "", "The namespace where to promote the Integration")

	return &cmd, &options
}

type promoteCmdOptions struct {
	*RootCmdOptions
	To string `mapstructure:"to" yaml:",omitempty"`
}

func (o *promoteCmdOptions) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("promote expects an integration name argument")
	}

	return nil
}

func (o *promoteCmdOptions) run(cmd *cobra.Command, args []string) error {
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

	checkOpsCompatibility(cmd, opSource, opDest)

	sourceIntegration, err := o.getIntegration(c, it)
	o.validateDestResources(c, sourceIntegration)
	//destIntegration := o.editIntegration(sourceIntegration)

	//return c.Create(o.Context, destIntegration)
	return nil
}

func checkOpsCompatibility(cmd *cobra.Command, source, dest map[string]string) {
	if !compatibleVersions(source["Version"], dest["Version"], cmd) {
		panic(fmt.Sprintf("source (%s) and destination (%s) Camel K operator versions are not compatible", source["version"], dest["version"]))
	}
	if !compatibleVersions(source["Runtime Version"], dest["Runtime Version"], cmd) {
		panic(fmt.Sprintf("source (%s) and destination (%s) Camel K runtime versions are not compatible", source["runtime version"], dest["runtime version"]))
	}
	if source["Registry Address"] != source["Registry Address"] {
		panic(fmt.Sprintf("source (%s) and destination (%s) Camel K container images registries are not the same", source["registry address"], dest["registry address"]))
	}
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

func (o *promoteCmdOptions) validateDestResources(c client.Client, it *v1.Integration) {
	var traits map[string][]string
	var configmaps []string
	var secrets []string
	var pvcs []string
	var kamelets []string
	// Mount trait
	mounts := it.Spec.Traits["mount"]
	json.Unmarshal(mounts.Configuration.RawMessage, &traits)
	for t, v := range traits {
		if t == "configs" || t == "resources" {
			for _, c := range v {
				//TODO proper parse resources, now it does not account for complex parsing
				if strings.HasPrefix(c, "configmap:") {
					configmaps = append(configmaps, strings.Split(c, ":")[1])
				}
				if strings.HasPrefix(c, "secret:") {
					secrets = append(secrets, strings.Split(c, ":")[1])
				}
			}
		} else if t == "volumes" {
			for _, c := range v {
				pvcs = append(pvcs, strings.Split(c, ":")[0])
			}
		}
	}
	// Openapi trait
	openapis := it.Spec.Traits["openapi"]
	json.Unmarshal(openapis.Configuration.RawMessage, &traits)
	for k, v := range traits {
		for _, c := range v {
			if k == "configmaps" {
				configmaps = append(configmaps, c)
			}
		}
	}
	// Kamelet trait
	kamelets = o.listKamelets(c, it)

	anyError := false
	for _, name := range configmaps {
		if !existsCm(o.Context, c, name, o.To) {
			anyError = true
			fmt.Printf("Configmap %s is missing from %s namespace\n", name, o.To)
		}
	}
	for _, name := range secrets {
		if !existsSecret(o.Context, c, name, o.To) {
			anyError = true
			fmt.Printf("Secret %s is missing from %s namespace\n", name, o.To)
		}
	}
	for _, name := range pvcs {
		if !existsPv(o.Context, c, name, o.To) {
			anyError = true
			fmt.Printf("PersistentVolume %s is missing from %s namespace\n", name, o.To)
		}
	}
	for _, name := range kamelets {
		if !existsKamelet(o.Context, c, name, o.To) {
			anyError = true
			fmt.Printf("Kamelet %s is missing from %s namespace\n", name, o.To)
		}
	}

	if anyError {
		os.Exit(1)
	}
}

func (o *promoteCmdOptions) listKamelets(c client.Client, it *v1.Integration) []string {
	// TODO collect any kamelets which may be coming into the kamelet trait as well
	var kamelets []string

	sources, _ := kubernetes.ResolveIntegrationSources(o.Context, c, it, &kubernetes.Collection{})
	catalog, _ := camel.DefaultCatalog()
	metadata.Each(catalog, sources, func(_ int, meta metadata.IntegrationMetadata) bool {
		util.StringSliceUniqueConcat(&kamelets, meta.Kamelets)
		return true
	})

	// Check if a Kamelet is configured as default error handler URI
	defaultErrorHandlerURI := it.Spec.GetConfigurationProperty(v1alpha1.ErrorHandlerAppPropertiesPrefix + ".deadLetterUri")
	if defaultErrorHandlerURI != "" {
		if strings.HasPrefix(defaultErrorHandlerURI, "kamelet:") {
			kamelets = append(kamelets, source.ExtractKamelet(defaultErrorHandlerURI))
		}
	}

	return kamelets
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
	dst.Spec.Traits = map[string]v1.TraitSpec{
		"container": traitSpecFromMap(map[string]interface{}{
			"image": contImage,
		}),
	}

	return &dst
}

// TODO refactor properly
func traitSpecFromMap(spec map[string]interface{}) v1.TraitSpec {
	var trait v1.TraitSpec
	data, _ := json.Marshal(spec)
	_ = json.Unmarshal(data, &trait.Configuration)
	return trait
}

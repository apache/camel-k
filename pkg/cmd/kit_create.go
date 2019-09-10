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
	"errors"
	"fmt"
	"strings"

	"github.com/apache/camel-k/pkg/trait"

	"github.com/apache/camel-k/pkg/util"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"

	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func newKitCreateCmd(rootCmdOptions *RootCmdOptions) *cobra.Command {
	impl := &kitCreateCommand{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:   "create <name>",
		Short: "Create an Integration Kit",
		Long:  `Create an Integration Kit.`,
		Args:  impl.validateArgs,
		RunE:  impl.run,
	}

	cmd.Flags().StringVar(&impl.image, "image", "", "Image used to create the kit")
	cmd.Flags().StringSliceVarP(&impl.dependencies, "dependency", "d", nil, "Add a dependency")
	cmd.Flags().StringSliceVarP(&impl.properties, "property", "p", nil, "Add a camel property")
	cmd.Flags().StringSliceVar(&impl.configmaps, "configmap", nil, "Add a ConfigMap")
	cmd.Flags().StringSliceVar(&impl.secrets, "secret", nil, "Add a Secret")
	cmd.Flags().StringSliceVar(&impl.repositories, "repository", nil, "Add a maven repository")
	cmd.Flags().StringSliceVarP(&impl.traits, "trait", "t", nil, "Configure a trait. E.g. \"-t service.enabled=false\"")

	// completion support
	configureKnownCompletions(&cmd)

	return &cmd
}

type kitCreateCommand struct {
	*RootCmdOptions

	image        string
	dependencies []string
	properties   []string
	configmaps   []string
	secrets      []string
	repositories []string
	traits       []string
}

func (command *kitCreateCommand) validateArgs(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("create expects a single name argument")
	}

	return nil
}

func (command *kitCreateCommand) run(_ *cobra.Command, args []string) error {
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}

	catalog := trait.NewCatalog(command.Context, c)
	tp := catalog.ComputeTraitsProperties()
	for _, t := range command.traits {
		kv := strings.SplitN(t, "=", 2)

		if !util.StringSliceExists(tp, kv[0]) {
			fmt.Printf("Error: %s is not a valid trait property\n", t)
			return nil
		}
	}

	ctx := v1alpha1.NewIntegrationKit(command.Namespace, args[0])
	key := k8sclient.ObjectKey{
		Namespace: command.Namespace,
		Name:      args[0],
	}
	if err := c.Get(command.Context, key, &ctx); err == nil {
		// the integration kit already exists, let's check that it is
		// not a platform one which is supposed to be "read only"

		if ctx.Labels["camel.apache.org/kit.type"] == v1alpha1.IntegrationKitTypePlatform {
			fmt.Printf("integration kit \"%s\" is not editable\n", ctx.Name)
			return nil
		}
	}

	ctx = v1alpha1.NewIntegrationKit(command.Namespace, kubernetes.SanitizeName(args[0]))
	ctx.Labels = map[string]string{
		"camel.apache.org/kit.type": v1alpha1.IntegrationKitTypeUser,
	}
	ctx.Spec = v1alpha1.IntegrationKitSpec{
		Dependencies:  make([]string, 0, len(command.dependencies)),
		Configuration: make([]v1alpha1.ConfigurationSpec, 0),
		Repositories:  command.repositories,
	}

	if command.image != "" {
		//
		// if the image is set, the kit do not require any build but
		// is be marked as external as the information about the classpath
		// is missing so it cannot be used as base for other kits
		//
		ctx.Labels["camel.apache.org/kit.type"] = v1alpha1.IntegrationKitTypeExternal

		//
		// Set the image to be used by the kit
		//
		ctx.Spec.Image = command.image
	}
	for _, item := range command.dependencies {
		switch {
		case strings.HasPrefix(item, "mvn:"):
			ctx.Spec.Dependencies = append(ctx.Spec.Dependencies, item)
		case strings.HasPrefix(item, "file:"):
			ctx.Spec.Dependencies = append(ctx.Spec.Dependencies, item)
		case strings.HasPrefix(item, "camel-"):
			ctx.Spec.Dependencies = append(ctx.Spec.Dependencies, "camel:"+strings.TrimPrefix(item, "camel-"))
		}
	}

	for _, item := range command.properties {
		ctx.Spec.Configuration = append(ctx.Spec.Configuration, v1alpha1.ConfigurationSpec{
			Type:  "property",
			Value: item,
		})
	}
	for _, item := range command.configmaps {
		ctx.Spec.Configuration = append(ctx.Spec.Configuration, v1alpha1.ConfigurationSpec{
			Type:  "configmap",
			Value: item,
		})
	}
	for _, item := range command.secrets {
		ctx.Spec.Configuration = append(ctx.Spec.Configuration, v1alpha1.ConfigurationSpec{
			Type:  "secret",
			Value: item,
		})
	}
	for _, item := range command.traits {
		if err := command.configureTrait(&ctx, item); err != nil {
			return nil
		}
	}

	existed := false
	err = c.Create(command.Context, &ctx)
	if err != nil && k8serrors.IsAlreadyExists(err) {
		existed = true
		clone := ctx.DeepCopy()
		err = c.Get(command.Context, key, clone)
		if err != nil {
			fmt.Print(err.Error())
			return nil
		}
		ctx.ResourceVersion = clone.ResourceVersion
		err = c.Update(command.Context, &ctx)
	}

	if err != nil {
		fmt.Print(err.Error())
		return nil
	}

	if !existed {
		fmt.Printf("integration kit \"%s\" created\n", ctx.Name)
	} else {
		fmt.Printf("integration kit \"%s\" updated\n", ctx.Name)
	}

	return nil
}

func (*kitCreateCommand) configureTrait(ctx *v1alpha1.IntegrationKit, config string) error {
	if ctx.Spec.Traits == nil {
		ctx.Spec.Traits = make(map[string]v1alpha1.TraitSpec)
	}

	parts := traitConfigRegexp.FindStringSubmatch(config)
	if len(parts) < 4 {
		return errors.New("unrecognized config format (expected \"<trait>.<prop>=<val>\"): " + config)
	}
	traitID := parts[1]
	prop := parts[2][1:]
	val := parts[3]

	spec, ok := ctx.Spec.Traits[traitID]
	if !ok {
		spec = v1alpha1.TraitSpec{
			Configuration: make(map[string]string),
		}
	}

	spec.Configuration[prop] = val
	ctx.Spec.Traits[traitID] = spec
	return nil
}

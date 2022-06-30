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

	"github.com/spf13/cobra"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func newKitCreateCmd(rootCmdOptions *RootCmdOptions) (*cobra.Command, *kitCreateCommandOptions) {
	options := kitCreateCommandOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "create <name>",
		Short:   "Create an Integration Kit",
		Long:    `Create an Integration Kit.`,
		Args:    options.validateArgs,
		PreRunE: decode(&options),
		RunE:    options.run,
	}

	cmd.Flags().String("image", "", "Image used to create the kit")
	cmd.Flags().StringArrayP("dependency", "d", nil, "Add a dependency")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a camel property")
	cmd.Flags().StringArray("configmap", nil, "Add a ConfigMap")
	cmd.Flags().StringArray("secret", nil, "Add a Secret")
	cmd.Flags().StringArray("repository", nil, "Add a maven repository")
	cmd.Flags().StringArrayP("trait", "t", nil, "Configure a trait. E.g. \"-t service.enabled=false\"")

	// completion support
	configureKnownCompletions(&cmd)

	return &cmd, &options
}

type kitCreateCommandOptions struct {
	*RootCmdOptions

	Image        string   `mapstructure:"image"`
	Dependencies []string `mapstructure:"dependencies"`
	Properties   []string `mapstructure:"properties"`
	Configmaps   []string `mapstructure:"configmaps"`
	Secrets      []string `mapstructure:"secrets"`
	Repositories []string `mapstructure:"repositories"`
	Traits       []string `mapstructure:"traits"`
}

func (command *kitCreateCommandOptions) validateArgs(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("create expects a single name argument")
	}

	return nil
}

func (command *kitCreateCommandOptions) run(cmd *cobra.Command, args []string) error {
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}

	catalog := trait.NewCatalog(c)
	tp := catalog.ComputeTraitsProperties()
	for _, t := range command.Traits {
		kv := strings.SplitN(t, "=", 2)

		if !util.StringSliceExists(tp, kv[0]) {
			fmt.Fprintln(cmd.OutOrStdout(), "Error:", t, "is not a valid trait property")
			return nil
		}
	}

	kit := v1.NewIntegrationKit(command.Namespace, args[0])
	key := ctrl.ObjectKey{
		Namespace: command.Namespace,
		Name:      args[0],
	}
	if err := c.Get(command.Context, key, kit); err == nil {
		// the integration kit already exists, let's check that it is
		// not a platform one which is supposed to be "read only"

		if kit.Labels[v1.IntegrationKitTypeLabel] == v1.IntegrationKitTypePlatform {
			fmt.Fprintln(cmd.OutOrStdout(), `integration kit "`+kit.Name+`" is not editable`)
			return nil
		}
	}

	kit = v1.NewIntegrationKit(command.Namespace, kubernetes.SanitizeName(args[0]))
	kit.Labels = map[string]string{
		v1.IntegrationKitTypeLabel: v1.IntegrationKitTypeUser,
	}
	kit.Spec = v1.IntegrationKitSpec{
		Dependencies:  make([]string, 0, len(command.Dependencies)),
		Configuration: make([]v1.ConfigurationSpec, 0),
		Repositories:  command.Repositories,
	}

	if command.Image != "" {
		// if the Image is set, the kit do not require any build but
		// is be marked as external as the information about the classpath
		// is missing so it cannot be used as base for other Kits
		kit.Labels[v1.IntegrationKitTypeLabel] = v1.IntegrationKitTypeExternal

		// Set the Image to be used by the kit
		kit.Spec.Image = command.Image
	}
	for _, item := range command.Dependencies {
		switch {
		case strings.HasPrefix(item, "camel-quarkus-"):
			kit.Spec.Dependencies = append(kit.Spec.Dependencies, "camel:"+strings.TrimPrefix(item, "camel-quarkus-"))
		case strings.HasPrefix(item, "camel-k-"):
			kit.Spec.Dependencies = append(kit.Spec.Dependencies, "camel-k:"+strings.TrimPrefix(item, "camel-k-"))
		case strings.HasPrefix(item, "camel-"):
			kit.Spec.Dependencies = append(kit.Spec.Dependencies, "camel:"+strings.TrimPrefix(item, "camel-"))
		default:
			kit.Spec.Dependencies = append(kit.Spec.Dependencies, item)
		}
	}

	if err := command.parseAndConvertToTrait(command.Properties, "camel.properties"); err != nil {
		return err
	}

	if err := command.parseAndConvertToTrait(command.Configmaps, "mount.config"); err != nil {
		return err
	}

	if err := command.parseAndConvertToTrait(command.Secrets, "mount.config"); err != nil {
		return err
	}
	if err := configureTraits(command.Traits, &kit.Spec.Traits, catalog); err != nil {
		return err
	}
	existed := false
	err = c.Create(command.Context, kit)
	if err != nil && k8serrors.IsAlreadyExists(err) {
		existed = true
		existing := v1.NewIntegrationKit(kit.Namespace, kit.Name)
		err = c.Get(command.Context, key, existing)
		if err != nil {
			fmt.Fprint(cmd.ErrOrStderr(), err.Error())
			return nil
		}
		kit.ResourceVersion = existing.ResourceVersion
		err = c.Update(command.Context, kit)
	}

	if err != nil {
		fmt.Fprint(cmd.ErrOrStderr(), err.Error())
		return nil
	}

	if !existed {
		fmt.Fprintln(cmd.OutOrStdout(), `integration kit "`+kit.Name+`" created`)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), `integration kit "`+kit.Name+`" updated`)
	}

	return nil
}

func (command *kitCreateCommandOptions) parseAndConvertToTrait(params []string, traitParam string) error {
	for _, param := range params {
		command.Traits = append(command.Traits, convertToTrait(param, traitParam))
	}

	return nil
}

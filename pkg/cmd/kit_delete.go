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

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/spf13/cobra"

	k8errors "k8s.io/apimachinery/pkg/api/errors"
)

func newKitDeleteCmd(rootCmdOptions *RootCmdOptions) (*cobra.Command, *kitDeleteCommandOptions) {
	options := kitDeleteCommandOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "delete <name>",
		Short:   "Delete an Integration Kit",
		Long:    `Delete an Integration Kit.`,
		PreRunE: decode(&options),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := options.validate(args); err != nil {
				return err
			}
			if err := options.run(args); err != nil {
				fmt.Println(err.Error())
			}

			return nil
		},
	}

	cmd.Flags().Bool("all", false, "Delete all integration Kits")

	return &cmd, &options
}

type kitDeleteCommandOptions struct {
	*RootCmdOptions
	All bool `mapstructure:"all"`
}

func (command *kitDeleteCommandOptions) validate(args []string) error {
	if command.All && len(args) > 0 {
		return errors.New("invalid combination: both all flag and named Kits are set")
	}
	if !command.All && len(args) == 0 {
		return errors.New("invalid combination: neither all flag nor named Kits are set")
	}

	return nil
}

func (command *kitDeleteCommandOptions) run(args []string) error {
	names := args

	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}

	if command.All {
		kitList := v1.NewIntegrationKitList()
		if err := c.List(command.Context, &kitList, k8sclient.InNamespace(command.Namespace)); err != nil {
			return err
		}

		names = make([]string, 0, len(kitList.Items))
		for _, item := range kitList.Items {
			// only include non platform Kits
			if item.Labels["camel.apache.org/kit.type"] != v1.IntegrationKitTypePlatform {
				names = append(names, item.Name)
			}
		}
	}

	for _, name := range names {
		if err := command.delete(name); err != nil {
			return err
		}
	}

	return nil
}

func (command *kitDeleteCommandOptions) delete(name string) error {
	ctx := v1.NewIntegrationKit(command.Namespace, name)
	key := k8sclient.ObjectKey{
		Namespace: command.Namespace,
		Name:      name,
	}
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}

	err = c.Get(command.Context, key, &ctx)

	// pass through if the kit is not found
	if err != nil && k8errors.IsNotFound(err) {
		return fmt.Errorf("no integration kit found with name \"%s\"", ctx.Name)
	}

	// fail otherwise
	if err != nil {
		return err
	}

	// check that it is not a platform one which is supposed to be "read only"
	// thus not managed by the end user
	if ctx.Labels["camel.apache.org/kit.type"] == v1.IntegrationKitTypePlatform {
		// skip platform Kits while deleting all Kits
		if command.All {
			return nil
		}

		return fmt.Errorf("integration kit \"%s\" is not editable", ctx.Name)
	}

	err = c.Delete(command.Context, &ctx)

	if err != nil && !k8errors.IsNotFound(err) {
		return fmt.Errorf("error deleting integration kit \"%s\", %s", ctx.Name, err)
	}
	if err != nil && k8errors.IsNotFound(err) {
		return fmt.Errorf("no integration kit found with name \"%s\"", ctx.Name)
	}

	fmt.Printf("integration kit \"%s\" has been deleted\n", ctx.Name)

	return err
}

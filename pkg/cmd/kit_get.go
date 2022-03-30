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
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func newKitGetCmd(rootCmdOptions *RootCmdOptions) (*cobra.Command, *kitGetCommandOptions) {
	options := kitGetCommandOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "get",
		Short:   "Get defined Integration Kit",
		Long:    `Get defined Integration Kit.`,
		PreRunE: decode(&options),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(cmd, args); err != nil {
				return err
			}
			if err := options.run(cmd); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
			}

			return nil
		},
	}

	cmd.Flags().Bool(v1.IntegrationKitTypeUser, true, "Includes user Kits")
	cmd.Flags().Bool(v1.IntegrationKitTypeExternal, true, "Includes external Kits")
	cmd.Flags().Bool(v1.IntegrationKitTypePlatform, true, "Includes platform Kits")

	return &cmd, &options
}

type kitGetCommandOptions struct {
	*RootCmdOptions
	User     bool `mapstructure:"user"`
	External bool `mapstructure:"external"`
	Platform bool `mapstructure:"platform"`
}

func (command *kitGetCommandOptions) validate(cmd *cobra.Command, args []string) error {
	return nil
}

func (command *kitGetCommandOptions) run(cmd *cobra.Command) error {
	kitList := v1.NewIntegrationKitList()
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}
	if err := c.List(command.Context, &kitList, k8sclient.InNamespace(command.Namespace)); err != nil {
		return err
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 8, 1, '\t', 0)
	fmt.Fprintln(w, "NAME\tPHASE\tTYPE\tIMAGE")
	for _, ctx := range kitList.Items {
		t := ctx.Labels[v1.IntegrationKitTypeLabel]
		u := command.User && t == v1.IntegrationKitTypeUser
		e := command.External && t == v1.IntegrationKitTypeExternal
		p := command.Platform && t == v1.IntegrationKitTypePlatform

		if u || e || p {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ctx.Name, string(ctx.Status.Phase), t, ctx.Status.Image)
		}
	}

	return w.Flush()
}

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
	"os"
	"text/tabwriter"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/spf13/cobra"
)

func newContextGetCmd(rootCmdOptions *RootCmdOptions) *cobra.Command {
	impl := contextGetCommand{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:   "get",
		Short: "Get defined Integration Context",
		Long:  `Get defined Integration Context.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := impl.validate(cmd, args); err != nil {
				return err
			}
			if err := impl.run(cmd, args); err != nil {
				fmt.Println(err.Error())
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&impl.user, "user", true, "Includes user contexts")
	cmd.Flags().BoolVar(&impl.platform, "platform", true, "Includes platform contexts")

	return &cmd
}

type contextGetCommand struct {
	*RootCmdOptions
	user     bool
	platform bool
}

func (command *contextGetCommand) validate(cmd *cobra.Command, args []string) error {
	return nil

}

func (command *contextGetCommand) run(cmd *cobra.Command, args []string) error {
	ctxList := v1alpha1.NewIntegrationContextList()
	if err := sdk.List(command.Namespace, &ctxList); err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tSTATUS")
	for _, ctx := range ctxList.Items {
		t := ctx.Labels["camel.apache.org/context.type"]
		u := command.user && t == "user"
		p := command.platform && t == "platform"

		if u || p {
			fmt.Fprintf(w, "%s\t%s\t%s\n", ctx.Name, t, string(ctx.Status.Phase))
		}
	}
	w.Flush()

	return nil
}

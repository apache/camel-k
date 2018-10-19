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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/spf13/cobra"

	k8errors "k8s.io/apimachinery/pkg/api/errors"
)

func newContextDeleteCmd(rootCmdOptions *RootCmdOptions) *cobra.Command {
	impl := contextDeleteCommand{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:   "delete",
		Short: "Delete an Integration Context",
		Long:  `Delete an Integration Context.`,
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

	cmd.Flags().BoolVar(&impl.all, "all", false, "Delete all integration contexts")

	return &cmd
}

type contextDeleteCommand struct {
	*RootCmdOptions
	all bool
}

func (command *contextDeleteCommand) validate(cmd *cobra.Command, args []string) error {
	if command.all && len(args) > 0 {
		return errors.New("invalid combination: both all flag and named contexts are set")
	}
	if !command.all && len(args) == 0 {
		return errors.New("invalid combination: neither all flag nor named contexts are set")
	}

	return nil
}

func (command *contextDeleteCommand) run(cmd *cobra.Command, args []string) error {
	names := args

	if command.all {
		ctxList := v1alpha1.NewIntegrationContextList()
		if err := sdk.List(command.Namespace, &ctxList); err != nil {
			return err
		}

		names = make([]string, 0, len(ctxList.Items))
		for _, item := range ctxList.Items {
			// only include non platform contexts
			if item.Labels["camel.apache.org/context.type"] != "platform" {
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

func (command *contextDeleteCommand) delete(name string) error {
	ctx := v1alpha1.NewIntegrationContext(command.Namespace, name)

	err := sdk.Get(&ctx)

	// pass through if the context is not found
	if err != nil && k8errors.IsNotFound(err) {
		return fmt.Errorf("no integration context found with name \"%s\"", ctx.Name)
	}

	// fail otherwise
	if err != nil {
		return err
	}

	// check that it is not a platform one which is supposed to be "read only"
	// thus not managed by the end user
	if ctx.Labels["camel.apache.org/context.type"] == "platform" {
		// skip platform contexts while deleting all contexts
		if command.all {
			return nil
		}

		return fmt.Errorf("integration context \"%s\" is not editable", ctx.Name)
	}

	err = sdk.Delete(&ctx)

	if err != nil && !k8errors.IsNotFound(err) {
		return fmt.Errorf("error deleting integration context \"%s\", %s", ctx.Name, err)
	}
	if err != nil && k8errors.IsNotFound(err) {
		return fmt.Errorf("no integration context found with name \"%s\"", ctx.Name)
	}

	fmt.Printf("integration context \"%s\" has been deleted\n", ctx.Name)

	return err
}

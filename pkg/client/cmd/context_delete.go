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
	"strconv"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/spf13/cobra"
)

func newContextDeleteCmd(rootCmdOptions *RootCmdOptions) *cobra.Command {
	impl := contextDeleteCommand{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:   "delete",
		Short: "Delete an Integration Context",
		Long:  `Delete anIntegration Context.`,
		Args:  impl.validateArgs,
		RunE:  impl.run,
	}

	return &cmd
}

type contextDeleteCommand struct {
	*RootCmdOptions
}

func (command *contextDeleteCommand) validateArgs(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("accepts 1 arg, received " + strconv.Itoa(len(args)))
	}

	return nil
}

func (command *contextDeleteCommand) run(cmd *cobra.Command, args []string) error {
	ctx := v1alpha1.NewIntegrationContext(command.Namespace, args[0])

	if err := sdk.Get(&ctx); err != nil {
		return err
	}

	// let's check that it is not a platform one which is supposed to be "read only"
	// thus not managed by the end user
	if ctx.Labels["camel.apache.org/context.type"] == "platform" {
		fmt.Printf("integration context \"%s\" is not editable\n", ctx.Name)
		return nil
	}

	if err := sdk.Delete(&ctx); err != nil {
		fmt.Printf("error deleting integration context \"%s\", %s\n", ctx.Name, err)
		return err
	}

	fmt.Printf("integration context \"%s\" has been deleted\n", ctx.Name)

	return nil
}

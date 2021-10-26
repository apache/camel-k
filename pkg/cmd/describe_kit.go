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
	"io"

	"github.com/spf13/cobra"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/indentedwriter"
)

func newDescribeKitCmd(rootCmdOptions *RootCmdOptions) (*cobra.Command, *describeKitCommandOptions) {
	options := describeKitCommandOptions{
		rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "kit",
		Aliases: []string{"ik"},
		Short:   "Describe an Integration Kit",
		Long:    `Describe an Integration Kit.`,
		PreRunE: decode(&options),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(cmd, args); err != nil {
				return err
			}
			if err := options.run(cmd, args); err != nil {
				fmt.Println(err.Error())
			}

			return nil
		},
	}

	return &cmd, &options
}

type describeKitCommandOptions struct {
	*RootCmdOptions
}

func (command *describeKitCommandOptions) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("describe expects a kit name argument")
	}
	return nil
}

func (command *describeKitCommandOptions) run(cmd *cobra.Command, args []string) error {
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}

	kit := v1.NewIntegrationKit(command.Namespace, args[0])
	kitKey := ctrl.ObjectKey{
		Namespace: command.Namespace,
		Name:      args[0],
	}

	if err := c.Get(command.Context, kitKey, kit); err == nil {
		if desc, err := command.describeIntegrationKit(cmd, kit); err == nil {
			fmt.Print(desc)
		} else {
			fmt.Println(err)
		}
	} else {
		fmt.Printf("IntegrationKit '%s' does not exist.\n", args[0])
	}

	return nil
}

func (command *describeKitCommandOptions) describeIntegrationKit(cmd *cobra.Command, kit *v1.IntegrationKit) (string, error) {
	return indentedwriter.IndentedString(func(out io.Writer) error {
		w := indentedwriter.NewWriter(cmd.OutOrStdout())

		describeObjectMeta(w, kit.ObjectMeta)

		w.Writef(0, "Phase:\t%s\n", kit.Status.Phase)
		w.Writef(0, "Runtime Version:\t%s\n", kit.Status.RuntimeVersion)
		w.Writef(0, "Image:\t%s\n", kit.Status.Image)
		w.Writef(0, "Version:\t%s\n", kit.Status.Version)

		if len(kit.Status.Artifacts) > 0 {
			w.Writef(0, "Artifacts:\t\n")
			for _, artifact := range kit.Status.Artifacts {
				w.Writef(1, "%s\n", artifact.ID)
			}
		}

		if len(kit.Spec.Configuration) > 0 {
			w.Writef(0, "Configuration:\n")
			for _, config := range kit.Spec.Configuration {
				w.Writef(1, "Type:\t%s\n", config.Type)
				w.Writef(1, "Value:\t%s\n", config.Value)
			}
		}

		if len(kit.Spec.Dependencies) > 0 {
			w.Writef(0, "Dependencies:\t\n")
			for _, dependency := range kit.Spec.Dependencies {
				w.Writef(1, "%s\n", dependency)
			}
		}

		if len(kit.Spec.Repositories) > 0 {
			w.Writef(0, "Repositories:\n")
			for _, repository := range kit.Spec.Repositories {
				w.Writef(1, "%s\n", repository)
			}
		}

		return describeTraits(w, kit.Spec.Traits)
	})
}

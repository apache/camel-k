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
	"strings"

	"github.com/spf13/cobra"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/indentedwriter"
)

func newDescribeIntegrationCmd(rootCmdOptions *RootCmdOptions) (*cobra.Command, *describeIntegrationCommandOptions) {

	options := describeIntegrationCommandOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "integration",
		Aliases: []string{"it"},
		Short:   "Describe an Integration",
		Long:    `Describe an Integration.`,
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

	cmd.Flags().BoolVar(&options.showSourceContent, "show-source-content", false, "Print source content")

	return &cmd, &options
}

type describeIntegrationCommandOptions struct {
	*RootCmdOptions
	showSourceContent bool `mapstructure:"show-source-content"`
}

func (command *describeIntegrationCommandOptions) validate(args []string) error {
	if len(args) != 1 {
		return errors.New("describe expects an integration name argument")
	}
	return nil
}

func (command *describeIntegrationCommandOptions) run(args []string) error {
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}

	ctx := v1.NewIntegration(command.Namespace, args[0])
	key := k8sclient.ObjectKey{
		Namespace: command.Namespace,
		Name:      args[0],
	}

	if err := c.Get(command.Context, key, &ctx); err == nil {
		if desc, err := command.describeIntegration(ctx); err == nil {
			fmt.Print(desc)
		} else {
			fmt.Println(err)
		}
	} else {
		fmt.Printf("Integration '%s' does not exist.\n", args[0])
	}

	return nil
}

func (command *describeIntegrationCommandOptions) describeIntegration(i v1.Integration) (string, error) {
	return indentedwriter.IndentedString(func(out io.Writer) error {
		w := indentedwriter.NewWriter(out)

		describeObjectMeta(w, i.ObjectMeta)

		w.Write(0, "Phase:\t%s\n", i.Status.Phase)
		w.Write(0, "Runtime Version:\t%s\n", i.Status.RuntimeVersion)
		kit := ""
		if i.Status.IntegrationKit != nil {
			ns := i.GetIntegrationKitNamespace(nil)
			kit = fmt.Sprintf("%s/%s", ns, i.Status.IntegrationKit.Name)
		}
		w.Write(0, "Kit:\t%s\n", kit)
		w.Write(0, "Image:\t%s\n", i.Status.Image)
		w.Write(0, "Version:\t%s\n", i.Status.Version)

		if len(i.Spec.Configuration) > 0 {
			w.Write(0, "Configuration:\n")
			for _, config := range i.Spec.Configuration {
				w.Write(1, "Type:\t%s\n", config.Type)
				w.Write(1, "Value:\t%s\n", config.Value)
			}
		}

		if len(i.Status.Dependencies) > 0 {
			w.Write(0, "Dependencies:\n")
			for _, dependency := range i.Status.Dependencies {
				w.Write(1, "%s\n", dependency)
			}
		}

		if len(i.Spec.Repositories) > 0 {
			w.Write(0, "Repositories:\n")
			for _, repository := range i.Spec.Repositories {
				w.Write(1, "%s\n", repository)
			}
		}

		if len(i.Spec.Resources) > 0 {
			w.Write(0, "Resources:\n")
			for _, resource := range i.Spec.Resources {
				w.Write(1, "Content:\n")
				w.Write(2, "%s\n", strings.TrimSpace(resource.Content))
				w.Write(1, "Name:\t%s\n", resource.Name)
				w.Write(1, "Type:\t%s\n", resource.Type)
			}
		}

		if len(i.Sources()) > 0 {
			w.Write(0, "Sources:\n")
			if command.showSourceContent {
				for _, s := range i.Sources() {
					w.Write(1, "Name:\t%s\n", s.Name)
					w.Write(1, "Language:\t%s\n", s.InferLanguage())
					w.Write(1, "Compression:\t%t\n", s.Compression)
					w.Write(1, "Content:\n")

					if s.ContentRef == "" {
						w.Write(2, "%s\n", strings.TrimSpace(s.Content))
					} else {
						w.Write(2, "Ref:\t%s\n", s.ContentRef)
						w.Write(2, "Ref Key:\t%s\n", s.ContentKey)
					}
				}
			} else {
				w.Write(1, "Name\tLanguage\tCompression\tRef\tRef Key\n")
				for _, s := range i.Sources() {
					w.Write(1, "%s\t%s\t%t\t%s\t%s\n",
						s.Name,
						s.InferLanguage(),
						s.Compression,
						s.ContentRef,
						s.ContentKey)
				}
			}
		}

		if len(i.Status.Conditions) > 0 {
			w.Write(0, "Conditions:\n")
			w.Write(1, "Type\tStatus\tReason\tMessage\n")
			for _, condition := range i.Status.Conditions {
				w.Write(1, "%s\t%s\t%s\t%s\n",
					condition.Type,
					condition.Status,
					condition.Reason,
					condition.Message)
			}
		}

		return describeTraits(w, i.Spec.Traits)
	})
}

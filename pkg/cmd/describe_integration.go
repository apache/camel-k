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
	"io"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/spf13/cobra"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func newDescribeIntegrationCmd(rootCmdOptions *RootCmdOptions) *cobra.Command {

	impl := &describeIntegrationCommand{
		rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:   "integration",
		Short: "Describe an Integration",
		Long:  `Describe an Integration.`,
		RunE: func(_ *cobra.Command, args []string) error {
			if err := impl.validate(args); err != nil {
				return err
			}
			if err := impl.run(args); err != nil {
				fmt.Println(err.Error())
			}

			return nil
		},
	}

	return &cmd
}

type describeIntegrationCommand struct {
	*RootCmdOptions
}

func (command *describeIntegrationCommand) validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("accepts at least 1 arg, received %d", len(args))
	}
	return nil
}

func (command *describeIntegrationCommand) run(args []string) error {
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}

	ctx := v1alpha1.NewIntegration(command.Namespace, args[0])
	key := k8sclient.ObjectKey{
		Namespace: command.Namespace,
		Name:      args[0],
	}

	if err := c.Get(command.Context, key, &ctx); err == nil {
		fmt.Print(command.describeIntegration(ctx))
	} else {
		fmt.Printf("Integration '%s' does not exist.\n", args[0])
	}

	return nil
}

func (command *describeIntegrationCommand) describeIntegration(i v1alpha1.Integration) string {
	return indentedString(func(out io.Writer) {
		w := newIndentedWriter(out)

		describeObjectMeta(w, i.ObjectMeta)

		w.write(0, "Phase:\t%s\n", i.Status.Phase)
		w.write(0, "Camel Version:\t%s\n", i.Status.CamelVersion)
		w.write(0, "Context:\t%s\n", i.Status.Context)
		w.write(0, "Image:\t%s\n", i.Status.Image)

		if len(i.Spec.Configuration) > 0 {
			w.write(0, "Configuration:\n")
			for _, config := range i.Spec.Configuration {
				w.write(1, "Type:\t%s\n", config.Type)
				w.write(1, "Value:\t%s\n", config.Value)
			}
		}

		if len(i.Status.Dependencies) > 0 {
			w.write(0, "Dependencies:\n")
			for _, dependency := range i.Status.Dependencies {
				w.write(1, "%s\n", dependency)
			}
		}

		if len(i.Spec.Repositories) > 0 {
			w.write(0, "Repositories:\n")
			for _, repository := range i.Spec.Repositories {
				w.write(1, "%s\n", repository)
			}
		}

		if len(i.Spec.Resources) > 0 {
			w.write(0, "Resources:\n")
			for _, resource := range i.Spec.Resources {
				w.write(1, "Content:\n")
				w.write(2, "%s\n", strings.TrimSpace(resource.Content))
				w.write(1, "Name:\t%s\n", resource.Name)
				w.write(1, "Type:\t%s\n", resource.Type)
			}
		}

		if len(i.Sources()) > 0 {
			w.write(0, "Sources:\n")
			for _, s := range i.Sources() {
				w.write(1, "Name:\t%s\n", s.Name)
				w.write(1, "Content:\n")
				w.write(2, "%s\n", strings.TrimSpace(s.Content))
			}
		}

		describeTraits(w, i.Spec.Traits)
	})
}

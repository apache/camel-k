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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/indentedwriter"
)

func newDescribeKameletCmd(rootCmdOptions *RootCmdOptions) (*cobra.Command, *describeKameletCommandOptions) {
	options := describeKameletCommandOptions{
		rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "kamelet",
		Aliases: []string{"kl"},
		Short:   "Describe a Kamelet",
		Long:    `Describe a Kamelet.`,
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

type describeKameletCommandOptions struct {
	*RootCmdOptions
}

func (command *describeKameletCommandOptions) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("describe expects a Kamelet name argument")
	}
	return nil
}

func (command *describeKameletCommandOptions) run(cmd *cobra.Command, args []string) error {
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}

	kamelet := v1alpha1.NewKamelet(command.Namespace, args[0])
	kameletKey := k8sclient.ObjectKey{
		Namespace: command.Namespace,
		Name:      args[0],
	}

	if err := c.Get(command.Context, kameletKey, &kamelet); err == nil {
		if desc, err := command.describeKamelet(cmd, kamelet); err == nil {
			fmt.Print(desc)
		} else {
			fmt.Println(err)
		}
	} else {
		fmt.Printf("Kamelet '%s' does not exist.\n", args[0])
	}

	return nil
}

func (command *describeKameletCommandOptions) describeKamelet(cmd *cobra.Command, kamelet v1alpha1.Kamelet) (string, error) {
	return indentedwriter.IndentedString(func(out io.Writer) error {
		w := indentedwriter.NewWriter(cmd.OutOrStdout())

		describeObjectMeta(w, kamelet.ObjectMeta)

		w.Writef(0, "Phase:\t%s\n", kamelet.Status.Phase)

		// Definition
		def := kamelet.Spec.Definition
		if def != nil {
			w.Writef(0, "Definition:\n")
			w.Writef(1, "Title:\t%s\n", def.Title)
			w.Writef(1, "Description:\n")
			for _, s := range strings.Split(def.Description, "\n") {
				w.Writef(2, "%s\n", s)
			}
			if len(def.Required) > 0 {
				w.Writef(1, "Required:\n")
				for _, req := range def.Required {
					w.Writef(2, "%s\n", req)
				}
			}
			if len(def.Properties) > 0 {
				w.Writef(1, "Properties:\n")
				for _, k := range kamelet.SortedDefinitionPropertiesKeys() {
					p := def.Properties[k]
					w.Writef(2, "%s:\n", k)
					w.Writef(3, "Title:\t%s\n", p.Title)
					w.Writef(3, "Description:\t%s\n", p.Description)
					w.Writef(3, "Type:\t%s\n", p.Type)
					if p.Default != nil {
						w.Writef(3, "Default:\t%s\n", p.Default)
					}
				}
			}
		}

		// Types
		if len(kamelet.Spec.Types) > 0 {
			w.Writef(0, "Types:\n")
			for _, k := range kamelet.SortedTypesKeys() {
				t := kamelet.Spec.Types[k]
				w.Writef(1, "%s:\n", k)
				w.Writef(2, "Media Type: %s\n", t.MediaType)
			}
		}

		// Template
		// TODO pretty print template data
		template := kamelet.Spec.Template
		if template != nil {
			w.Writef(0, "Template:\n")
			w.Writef(1, "%s\n", string(template.RawMessage))
		}

		// Dependencies
		if len(kamelet.Spec.Dependencies) > 0 {
			w.Writef(0, "Dependencies:\t\n")
			for _, d := range kamelet.Spec.Dependencies {
				w.Writef(1, "%s\n", d)
			}
		}

		// Sources
		if len(kamelet.Spec.Sources) > 0 {
			w.Writef(0, "Sources:\t\n")
			w.Writef(1, "Name\tLanguage\tCompression\tRef\tRef Key\n")
			for _, s := range kamelet.Spec.Sources {
				w.Writef(1, "%s\t%s\t%t\t%s\t%s\n",
					s.Name,
					s.InferLanguage(),
					s.Compression,
					s.ContentRef,
					s.ContentKey)
			}
		}

		return nil
	})
}

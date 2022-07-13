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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/spf13/cobra"
)

// newCmdLocal -- Add local kamel subcommand with several other subcommands of its own.
func newCmdLocal(rootCmdOptions *RootCmdOptions) (*cobra.Command, *LocalCmdOptions) {
	options := LocalCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:               "local [sub-command]",
		Short:             "Perform integration actions locally.",
		Long:              `Perform integration actions locally given a set of input integration files.`,
		PersistentPreRunE: options.persistentPreRun,
		Annotations: map[string]string{
			offlineCommandLabel: "true",
		},
	}

	cmd.PersistentFlags().StringArrayVarP(&options.Dependencies, "dependency", "d", nil, usageDependency)

	// hidden flags for compatibility with kamel run
	cmd.PersistentFlags().StringArrayVarP(&options.Traits, "trait", "t", nil, "")
	if err := cmd.PersistentFlags().MarkHidden("trait"); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
	}

	cmd.AddCommand(cmdOnly(newCmdLocalBuild(&options)))
	cmd.AddCommand(cmdOnly(newCmdLocalInspect(&options)))
	cmd.AddCommand(cmdOnly(newCmdLocalRun(&options)))

	return &cmd, &options
}

type LocalCmdOptions struct {
	*RootCmdOptions
	Dependencies []string `mapstructure:"dependencies"`
	Traits       []string `mapstructure:"traits"`
}

func (o *LocalCmdOptions) persistentPreRun(cmd *cobra.Command, args []string) error {
	// pre-process dependencies
	for i, dependency := range o.Dependencies {
		o.Dependencies[i] = v1.NormalizeDependency(dependency)
	}

	// validate traits
	warnTraitUsages(cmd, o.Traits)

	return nil
}

func warnTraitUsages(cmd *cobra.Command, traits []string) {
	if len(traits) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Warning: traits are specified but don't take effect for local run: %v\n", traits)
	}
}

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

	"github.com/spf13/cobra"

	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

// VersionVariant may be overridden at build time.
var VersionVariant = ""

func newCmdVersion(rootCmdOptions *RootCmdOptions) (*cobra.Command, *versionCmdOptions) {
	options := versionCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:               "version",
		Short:             "Display client version",
		Long:              `Display Camel K client version.`,
		PersistentPreRunE: decode(&options, options.Flags),
		PreRunE:           options.preRunE,
		RunE:              options.run,
		Annotations:       make(map[string]string),
	}

	cmd.Flags().BoolP("verbose", "v", false, "Display all available extra information")

	return &cmd, &options
}

type versionCmdOptions struct {
	*RootCmdOptions

	Verbose bool `mapstructure:"verbose"`
}

func (o *versionCmdOptions) preRunE(cmd *cobra.Command, args []string) error {
	cmd.Annotations[offlineCommandLabel] = "true"

	return o.preRun(cmd, args)
}

func (o *versionCmdOptions) run(cmd *cobra.Command, _ []string) error {
	o.displayClientVersion(cmd)

	return nil
}

func (o *versionCmdOptions) displayClientVersion(cmd *cobra.Command) {
	if VersionVariant != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Camel K Client %s %s\n", VersionVariant, defaults.Version)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Camel K Client %s\n", defaults.Version)
	}
	if o.Verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Git Commit: %s\n", defaults.GitCommit)
	}
}

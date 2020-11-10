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
	"strings"

	"github.com/spf13/cobra"
)

func newCmdLocalRun(rootCmdOptions *RootCmdOptions) (*cobra.Command, *localRunCmdOptions) {
	options := localRunCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "local-run [files to inspect]",
		Short:   "Run a Camel integration locally.",
		Long:    `Run a Camel integration locally using existing integration files.`,
		PreRunE: decode(&options),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := options.validate(args); err != nil {
				return err
			}
			if err := options.init(); err != nil {
				return err
			}
			if err := options.run(args); err != nil {
				fmt.Println(err.Error())
			}
			if err := options.deinit(); err != nil {
				return err
			}

			return nil
		},
		Annotations: map[string]string{
			offlineCommandLabel: "true",
		},
	}

	cmd.Flags().StringArrayP("properties-file", "p", nil, "File containing integration properties.")
	cmd.Flags().StringArrayP("dependency", "d", nil, `Additional top-level dependency with the format:
<type>:<dependency-name>
where <type> is one of {`+strings.Join(acceptedDependencyTypes, "|")+`}.`)

	return &cmd, &options
}

type localRunCmdOptions struct {
	*RootCmdOptions
	PropertiesFiles        []string `mapstructure:"properties-files"`
	AdditionalDependencies []string `mapstructure:"dependencies"`
}

func (command *localRunCmdOptions) validate(args []string) error {
	for _, additionalDependency := range command.AdditionalDependencies {
		fmt.Printf("Dep: %v\n", additionalDependency)
	}

	for _, prop := range command.PropertiesFiles {
		fmt.Printf("Prop: %v\n", prop)
	}

	// Validate additional dependencies specified by the user.
	err := validateIntegrationForDependencies(args, command.AdditionalDependencies)
	if err != nil {
		return err
	}

	// Validate properties file.
	err = validateFiles(command.PropertiesFiles)
	if err != nil {
		return nil
	}

	return nil
}

func (command *localRunCmdOptions) init() error {
	return createMavenWorkingDirectory()
}

func (command *localRunCmdOptions) run(args []string) error {
	// Fetch dependencies.
	dependencies, err := getDependencies(args, command.AdditionalDependencies, true)
	if err != nil {
		return err
	}

	// Print dependencies.
	err = outputDependencies(dependencies, "")
	if err != nil {
		return err
	}

	// Run the integration locally.
	err = RunLocalIntegration(command.PropertiesFiles, dependencies, args)
	if err != nil {
		return nil
	}

	return nil
}

func (command *localRunCmdOptions) deinit() error {
	return deleteMavenWorkingDirectory()
}

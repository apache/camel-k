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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdLocalRun(rootCmdOptions *RootCmdOptions) (*cobra.Command, *localRunCmdOptions) {
	options := localRunCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "run [integration files]",
		Short:   "Run integration locally.",
		Long:    `Run integration locally using the input integration files.`,
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

	cmd.Flags().Bool("containerize", false, "Run integration in a local container.")
	cmd.Flags().String("docker-registry", "", "Docker registry to store intermediate images.")
	cmd.Flags().StringArray("property-file", nil, "Add a property file to the integration.")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a Camel property to the integration.")
	cmd.Flags().StringArrayP("dependency", "d", nil, additionalDependencyUsageMessage)

	return &cmd, &options
}

type localRunCmdOptions struct {
	*RootCmdOptions
	Containerize           bool     `mapstructure:"containerize"`
	DockerRegistry         string   `mapstructure:"docker-registry"`
	PropertyFiles          []string `mapstructure:"property-files"`
	Properties             []string `mapstructure:"properties"`
	AdditionalDependencies []string `mapstructure:"dependencies"`
}

func (command *localRunCmdOptions) validate(args []string) error {
	// Validate additional dependencies specified by the user.
	err := validateIntegrationForDependencies(args, command.AdditionalDependencies)
	if err != nil {
		return err
	}

	// Validate properties file.
	err = validatePropertyFiles(command.PropertyFiles)
	if err != nil {
		return nil
	}

	// If containerize is set then docker registry must be set.
	if command.Containerize && command.DockerRegistry == "" {
		return errors.New("containerization is active but no registry has been provided")
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

	// Manage integration properties which may come from files or CLI.
	err = updateIntegrationProperties(command)
	if err != nil {
		return nil
	}

	// Run the integration locally.
	err = RunLocalIntegration(command.PropertyFiles, dependencies, args)
	if err != nil {
		return nil
	}

	return nil
}

func (command *localRunCmdOptions) deinit() error {
	return deleteMavenWorkingDirectory()
}

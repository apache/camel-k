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

func newCmdLocalCreate(rootCmdOptions *RootCmdOptions) (*cobra.Command, *localCreateCmdOptions) {
	options := localCreateCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "create [options]",
		Short:   "Create integration images locally.",
		Long:    `Create integration images locally for containerized integrations.`,
		PreRunE: decode(&options),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := options.validate(args); err != nil {
				return err
			}
			if err := options.init(args); err != nil {
				return err
			}
			if err := options.run(args); err != nil {
				fmt.Println(err.Error())
			}
			if err := options.deinit(args); err != nil {
				return err
			}

			return nil
		},
		Annotations: map[string]string{
			offlineCommandLabel: "true",
		},
	}

	cmd.Flags().Bool("base-image", false, "Create base image used as a starting point for any integration.")
	cmd.Flags().String("docker-registry", "", "Docker registry to store intermediate images.")
	cmd.Flags().StringArray("property-file", nil, "Add a property file to the integration.")
	cmd.Flags().StringArrayP("dependency", "d", nil, "Add an additional dependency")

	return &cmd, &options
}

type localCreateCmdOptions struct {
	*RootCmdOptions
	BaseImage              bool     `mapstructure:"base-image"`
	DockerRegistry         string   `mapstructure:"docker-registry"`
	AdditionalDependencies []string `mapstructure:"dependencies"`
	PropertyFiles          []string `mapstructure:"property-files"`
}

func (command *localCreateCmdOptions) validate(args []string) error {
	// Validate integration files.
	if len(args) > 0 {
		err := validateIntegrationFiles(args)
		if err != nil {
			return err
		}
	}

	// Validate additional dependencies specified by the user.
	err := validateAdditionalDependencies(command.AdditionalDependencies)
	if err != nil {
		return err
	}

	// Validate properties file.
	err = validateFiles(command.PropertyFiles)
	if err != nil {
		return nil
	}

	// Docker registry must be set.
	if command.DockerRegistry == "" {
		return errors.New("base image cannot be created as no registry has been provided")
	}

	return nil
}

func (command *localCreateCmdOptions) init(args []string) error {
	// If base image construction is enabled create a directory for it.
	if command.BaseImage {
		err := createDockerBaseWorkingDirectory()
		if err != nil {
			return err
		}
	}

	// If integration files are provided an integration image will be built.
	if len(args) > 0 {
		err := createMavenWorkingDirectory()
		if err != nil {
			return err
		}
	}

	return nil
}

func (command *localCreateCmdOptions) run(args []string) error {
	// Create the Dockerfile and build the base image.
	if command.BaseImage {
		err := createAndBuildBaseImage(command.DockerRegistry)
		if err != nil {
			return err
		}
	}

	// Create integration image if integration files were provided.
	if len(args) > 0 {
		// Fetch dependencies.
		dependencies, err := getDependencies(args, command.AdditionalDependencies, true)
		if err != nil {
			return err
		}

		// Get integration run command.
		cmd := GetIntegrationRunCommand(command.PropertyFiles, dependencies, args)

		// Run integration locally.
		err = cmd.Run()
		if err != nil {
			return nil
		}
	}

	return nil
}

func (command *localCreateCmdOptions) deinit(args []string) error {
	// If base image construction is enabled delete the directory for it.
	if command.BaseImage {
		deleteDockerBaseWorkingDirectory()
	}

	// If integration files are provided delete the maven project folder.
	if len(args) > 0 {
		deleteMavenWorkingDirectory()
	}

	return nil
}

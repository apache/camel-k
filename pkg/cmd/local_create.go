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
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(args); err != nil {
				return err
			}
			if err := options.init(args); err != nil {
				return err
			}
			if err := options.run(cmd, args); err != nil {
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
	cmd.Flags().String("container-registry", "", "Registry that holds intermediate images.")
	cmd.Flags().String("image", "", "Full path to integration image including registry.")
	cmd.Flags().StringArray("property-file", nil, "Add a property file to the integration.")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a Camel property to the integration.")
	cmd.Flags().StringArrayP("dependency", "d", nil, "Add an additional dependency")
	cmd.Flags().StringArray("maven-repository", nil, "Use a maven repository")

	return &cmd, &options
}

type localCreateCmdOptions struct {
	*RootCmdOptions
	BaseImage              bool     `mapstructure:"base-image"`
	ContainerRegistry      string   `mapstructure:"container-registry"`
	Image                  string   `mapstructure:"image"`
	AdditionalDependencies []string `mapstructure:"dependencies"`
	Properties             []string `mapstructure:"properties"`
	PropertyFiles          []string `mapstructure:"property-files"`
	MavenRepositories      []string `mapstructure:"maven-repositories"`
}

func (command *localCreateCmdOptions) validate(args []string) error {
	// Validate integration files.
	if len(args) > 0 {
		err := validateIntegrationFiles(args)
		if err != nil {
			return err
		}

		if command.ContainerRegistry != "" {
			return errors.New("cannot specify container registry when building integration image")
		}

		if command.Image == "" {
			return errors.New("image path not provided for integration")
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
	if command.BaseImage && command.ContainerRegistry == "" {
		return errors.New("base image cannot be created as registry has not been provided")
	}

	return nil
}

func (command *localCreateCmdOptions) init(args []string) error {
	// If base image construction is enabled create a directory for it.
	err := createDockerBaseWorkingDirectory()
	if err != nil {
		return err
	}

	// If integration files are provided an integration image will be built.
	if !command.BaseImage {
		err := createDockerWorkingDirectory()
		if err != nil {
			return err
		}

		err = createMavenWorkingDirectory()
		if err != nil {
			return err
		}
	}

	return nil
}

func (command *localCreateCmdOptions) run(cmd *cobra.Command, args []string) error {
	dependenciesList := []string{}
	propertyFilesList := []string{}
	if !command.BaseImage {
		// Fetch dependencies.
		dependencies, err := getDependencies(args, command.AdditionalDependencies, command.MavenRepositories, true)
		if err != nil {
			return err
		}
		dependenciesList = dependencies

		// Manage integration properties which may come from files or CLI.
		propertyFiles, err := updateIntegrationProperties(command.Properties, command.PropertyFiles)
		if err != nil {
			return err
		}
		propertyFilesList = propertyFiles
	}

	// Create and build integration image.
	err := createAndBuildIntegrationImage(command.Context, command.ContainerRegistry, command.BaseImage,
		command.Image, propertyFilesList, dependenciesList, args, cmd.OutOrStdout(), cmd.ErrOrStderr())
	if err != nil {
		return err
	}

	return nil
}

func (command *localCreateCmdOptions) deinit(args []string) error {
	// If base image construction is enabled delete the directory for it.
	deleteDockerBaseWorkingDirectory()

	// If integration files are provided delete the maven project folder.
	if !command.BaseImage {
		deleteDockerWorkingDirectory()
		deleteMavenWorkingDirectory()
	}

	return nil
}

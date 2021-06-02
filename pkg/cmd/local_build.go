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

	"github.com/apache/camel-k/pkg/util"
)

func newCmdLocalBuild(rootCmdOptions *RootCmdOptions) (*cobra.Command, *localBuildCmdOptions) {
	options := localBuildCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "build [options]",
		Short:   "Build integration images locally.",
		Long:    `Build integration images locally for containerized integrations.`,
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

	cmd.Flags().Bool("base-image", false, "Build base image used as a starting point for any integration.")
	cmd.Flags().Bool("dependencies-only", false, "Only output the integration dependencies. The integration-directory flag must be set.")
	cmd.Flags().String("container-registry", "", "Registry that holds intermediate images. This flag should only be used in conjunction with the base-image flag.")
	cmd.Flags().String("image", "", "Full path to integration image including registry.")
	cmd.Flags().String("integration-directory", "", "Directory to hold local integration files.")
	cmd.Flags().StringArray("property-file", nil, "Add a property file to the integration.")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a Camel property to the integration.")
	cmd.Flags().StringArrayP("dependency", "d", nil, "Add an additional dependency")
	cmd.Flags().StringArray("maven-repository", nil, "Use a maven repository")

	return &cmd, &options
}

type localBuildCmdOptions struct {
	*RootCmdOptions
	BaseImage              bool     `mapstructure:"base-image"`
	DependenciesOnly       bool     `mapstructure:"dependencies-only"`
	ContainerRegistry      string   `mapstructure:"container-registry"`
	Image                  string   `mapstructure:"image"`
	IntegrationDirectory   string   `mapstructure:"integration-directory"`
	AdditionalDependencies []string `mapstructure:"dependencies"`
	Properties             []string `mapstructure:"properties"`
	PropertyFiles          []string `mapstructure:"property-files"`
	MavenRepositories      []string `mapstructure:"maven-repositories"`
}

func (command *localBuildCmdOptions) validate(args []string) error {
	// Validate integration files.
	if len(args) > 0 {
		err := validateIntegrationFiles(args)
		if err != nil {
			return err
		}

		// Cannot have both integration files and the base image construction enabled.
		if command.BaseImage {
			return errors.New("integration files have been provided and the base image construction is enabled")
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
		return err
	}

	// ContainerRegistry should only be specified when building the base image.
	if !command.BaseImage && command.ContainerRegistry != "" {
		return errors.New("cannot specify container registry unless a base integration image is being built")
	}

	// Docker registry must be set.
	if command.BaseImage && command.ContainerRegistry == "" {
		return errors.New("base image cannot be built because container registry has not been provided")
	}

	// If an integration directory is provided then no base image containerization can be enabled.
	if command.BaseImage && command.IntegrationDirectory != "" {
		return errors.New("base image construction does not use integration files")
	}

	// The integration directory must be set when only outputting dependencies.
	if command.DependenciesOnly && command.IntegrationDirectory == "" {
		return errors.New("to output dependencies the integration directory flag must be set")
	}

	return nil
}

func (command *localBuildCmdOptions) init(args []string) error {
	// Create integration directory if one is provided.
	err := util.CreateDirectory(command.IntegrationDirectory)
	if err != nil {
		return err
	}

	if command.BaseImage || command.Image != "" {
		// If base image construction is enabled create a directory for it.
		err := createDockerBaseWorkingDirectory()
		if err != nil {
			return err
		}

		// If integration image construction is enabled, an integration image will be built.
		if command.Image != "" {
			err := createDockerWorkingDirectory()
			if err != nil {
				return err
			}
		}
	}

	err = createMavenWorkingDirectory()
	if err != nil {
		return err
	}

	return nil
}

func (command *localBuildCmdOptions) run(cmd *cobra.Command, args []string) error {
	var dependenciesList, propertyFilesList []string
	routeFiles := args
	if !command.BaseImage {
		dependencies, err := getDependencies(command.Context, args, command.AdditionalDependencies, command.MavenRepositories, true)
		if err != nil {
			return err
		}

		var propertyFiles []string
		if !command.DependenciesOnly {
			// Manage integration properties which may come from files or CLI
			propertyFiles, err = updateIntegrationProperties(command.Properties, command.PropertyFiles, false)
			if err != nil {
				return err
			}
		}

		dependenciesList = dependencies
		propertyFilesList = propertyFiles
		hasIntegrationDir := command.IntegrationDirectory != ""
		if hasIntegrationDir {
			// Create dependencies subdirectory.
			localDependenciesDirectory := getCustomDependenciesDir(command.IntegrationDirectory)

			// Copy dependencies in persistent IntegrationDirectory/dependencies
			dependenciesList, err = util.CopyIntegrationFilesToDirectory(dependencies, localDependenciesDirectory)
			if err != nil {
				return err
			}

			// Once dependencies have been copied to local folder, we can exit.
			if command.DependenciesOnly {
				return nil
			}

			// Create dependencies subdirectory.
			localPropertiesDirectory := getCustomPropertiesDir(command.IntegrationDirectory)

			// Copy dependencies in persistent IntegrationDirectory/dependencies
			propertyFilesList, err = util.CopyIntegrationFilesToDirectory(propertyFiles, localPropertiesDirectory)
			if err != nil {
				return err
			}

			// Save routes.
			localRoutesDirectory := getCustomRoutesDir(command.IntegrationDirectory)

			// Copy routes in persistent IntegrationDirectory/dependencies
			routeFiles, err = util.CopyIntegrationFilesToDirectory(args, localRoutesDirectory)
			if err != nil {
				return err
			}
		}
	}

	// Integration directory can only be used when building an integration image or when we just
	// build the integration without also building the image. A local build of the integration is
	// represented by all the files that define	 the integration: dependencies, properties and routes.

	// The only case where we should not execute the image integration creation is when we want to
	// just output the files that comprise the integration locally.
	if command.IntegrationDirectory != "" && command.Image == "" {
		return nil
	}

	// Create and build integration image.
	err := createAndBuildIntegrationImage(command.Context, command.ContainerRegistry, command.BaseImage,
		command.Image, propertyFilesList, dependenciesList, routeFiles, cmd.OutOrStdout(), cmd.ErrOrStderr())
	if err != nil {
		return err
	}

	return nil
}

func (command *localBuildCmdOptions) deinit(args []string) error {
	// If base image construction is enabled delete the directory for it.
	deleteDockerBaseWorkingDirectory()

	// If integration files are provided delete the maven project folder.
	if !command.BaseImage {
		deleteDockerWorkingDirectory()
		deleteMavenWorkingDirectory()
	}

	return nil
}

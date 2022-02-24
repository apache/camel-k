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

func newCmdLocalRun(rootCmdOptions *RootCmdOptions) (*cobra.Command, *localRunCmdOptions) {
	options := localRunCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "run [integration files]",
		Short:   "Run integration locally.",
		Long:    `Run integration locally using the input integration files.`,
		PreRunE: decode(&options),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(cmd, args); err != nil {
				return err
			}
			if err := options.init(); err != nil {
				return err
			}
			if err := options.run(cmd, args); err != nil {
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
	cmd.Flags().String("image", "", "Full path to integration image including registry.")
	cmd.Flags().String("network", "", "Custom network name to be used by the underlying Docker command.")
	cmd.Flags().String("integration-directory", "", "Directory which holds the locally built integration and is the result of a local build action.")
	cmd.Flags().StringArrayP("env", "e", nil, "Flag to specify an environment variable [--env VARIABLE=value].")
	cmd.Flags().StringArray("property-file", nil, "Add a property file to the integration.")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a Camel property to the integration.")
	cmd.Flags().StringArrayP("dependency", "d", nil, additionalDependencyUsageMessage)
	cmd.Flags().StringArray("maven-repository", nil, "Use a maven repository")

	// hidden flags for compatibility with kamel run
	cmd.Flags().StringArrayP("trait", "t", nil, "")
	if err := cmd.Flags().MarkHidden("trait"); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
	}

	return &cmd, &options
}

type localRunCmdOptions struct {
	*RootCmdOptions
	Containerize           bool     `mapstructure:"containerize"`
	Image                  string   `mapstructure:"image"`
	Network                string   `mapstructure:"network"`
	IntegrationDirectory   string   `mapstructure:"integration-directory"`
	EnvironmentVariables   []string `mapstructure:"envs"`
	PropertyFiles          []string `mapstructure:"property-files"`
	Properties             []string `mapstructure:"properties"`
	AdditionalDependencies []string `mapstructure:"dependencies"`
	MavenRepositories      []string `mapstructure:"maven-repositories"`
	Traits                 []string `mapstructure:"traits"`
}

func (command *localRunCmdOptions) validate(cmd *cobra.Command, args []string) error {
	// Validate integration files when no image is provided and we are
	// not running an already locally-built integration.
	if command.Image == "" && command.IntegrationDirectory == "" {
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
	err = validatePropertyFiles(command.PropertyFiles)
	if err != nil {
		return err
	}

	// If containerize is set then docker image name must be set.
	if command.Containerize && command.Image == "" {
		return errors.New("containerization is active but no image name has been provided")
	}

	warnTraitUsages(cmd, command.Traits)

	return nil
}

func (command *localRunCmdOptions) init() error {
	if command.Containerize {
		err := createDockerBaseWorkingDirectory()
		if err != nil {
			return err
		}

		err = createDockerWorkingDirectory()
		if err != nil {
			return err
		}
	}

	setDockerNetworkName(command.Network)

	setDockerEnvVars(command.EnvironmentVariables)

	return createMavenWorkingDirectory()
}

func (command *localRunCmdOptions) run(cmd *cobra.Command, args []string) error {
	// If local run is provided with an image name, it will just run the image locally and exit.
	if command.Image != "" && !command.Containerize {
		// Run image locally.
		err := runIntegrationImage(command.Context, command.Image, cmd.OutOrStdout(), cmd.ErrOrStderr())
		if err != nil {
			return err
		}

		return nil
	}

	hasIntegrationDir := command.IntegrationDirectory != ""

	var dependencies []string
	if hasIntegrationDir {
		// Fetch local dependencies
		localBuildDependencies, err := getLocalBuildDependencies(command.IntegrationDirectory)
		if err != nil {
			return err
		}
		dependencies = localBuildDependencies

		// Local dependencies directory
		localDependenciesDirectory := getCustomDependenciesDir(command.IntegrationDirectory)

		// The quarkus application files need to be at a specific location i.e.:
		// <current_working_folder>/quarkus/quarkus-application.dat
		// <current_working_folder>/quarkus/generated-bytecode.jar
		localQuarkusDir, err := getCustomQuarkusDir()
		if err != nil {
			return err
		}
		err = util.CopyQuarkusAppFiles(localDependenciesDirectory, localQuarkusDir)
		if err != nil {
			return err
		}

		// The dependency jar files need to be at a specific location i.e.:
		// <current_working_folder>/lib/main/*.jar
		localLibDirectory, err := getCustomLibDir()
		if err != nil {
			return err
		}
		err = util.CopyLibFiles(localDependenciesDirectory, localLibDirectory)
		if err != nil {
			return err
		}

		// The Camel-K jar file needs to be at a specific location i.e.:
		// <current_working_folder>/app/camel-k-integration-X.X.X{-SNAPSHOT}.jar
		localAppDirectory, err := getCustomAppDir()
		if err != nil {
			return err
		}
		err = util.CopyAppFile(localDependenciesDirectory, localAppDirectory)
		if err != nil {
			return err
		}
	} else {
		computedDependencies, err := getDependencies(command.Context, args, command.AdditionalDependencies, command.MavenRepositories, true)
		if err != nil {
			return err
		}
		dependencies = computedDependencies
	}

	// Manage integration properties which may come from files or CLI.
	propertyFiles := command.PropertyFiles
	if hasIntegrationDir {
		localBuildPropertyFiles, err := getLocalBuildProperties(command.IntegrationDirectory)
		if err != nil {
			return err
		}
		propertyFiles = localBuildPropertyFiles
	}

	updatedPropertyFiles, err := updateIntegrationProperties(command.Properties, propertyFiles, hasIntegrationDir)
	if err != nil {
		return err
	}
	propertyFiles = updatedPropertyFiles

	routes := args
	if hasIntegrationDir {
		localBuildRoutes, err := getLocalBuildRoutes(command.IntegrationDirectory)
		if err != nil {
			return err
		}
		routes = localBuildRoutes
	}

	// If this is a containerized local run, create, build and run the container image.
	if command.Containerize {
		// Create and build integration image.
		err := createAndBuildIntegrationImage(command.Context, "", false, command.Image, propertyFiles, dependencies, routes, hasIntegrationDir, cmd.OutOrStdout(), cmd.ErrOrStderr())
		if err != nil {
			return err
		}

		// Run integration image.
		err = runIntegrationImage(command.Context, command.Image, cmd.OutOrStdout(), cmd.ErrOrStderr())
		if err != nil {
			return err
		}
	} else {
		propertiesDir := util.GetLocalPropertiesDir()
		if hasIntegrationDir {
			propertiesDir = getCustomPropertiesDir(command.IntegrationDirectory)
		}

		// Run integration locally.
		err := RunLocalIntegrationRunCommand(command.Context, propertyFiles, dependencies, routes, propertiesDir, cmd.OutOrStdout(), cmd.ErrOrStderr())
		if err != nil {
			return err
		}
	}

	return nil
}

func (command *localRunCmdOptions) deinit() error {
	if command.Containerize {
		err := deleteDockerBaseWorkingDirectory()
		if err != nil {
			return err
		}

		err = deleteDockerWorkingDirectory()
		if err != nil {
			return err
		}
	}

	if command.IntegrationDirectory != "" {
		err := deleteLocalIntegrationDirs()
		if err != nil {
			return err
		}
	}

	return deleteMavenWorkingDirectory()
}

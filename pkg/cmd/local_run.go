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
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdLocalRun(localCmdOptions *LocalCmdOptions) (*cobra.Command, *localRunCmdOptions) {
	options := localRunCmdOptions{
		LocalCmdOptions: localCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "run [integration files]",
		Short:   "Run integration locally.",
		Long:    `Run integration locally using the input integration files.`,
		PreRunE: decode(&options),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(args); err != nil {
				return err
			}
			if err := options.init(); err != nil {
				return err
			}

			// make sure cleanup is done when process is stopped externally
			cs := make(chan os.Signal, 1)
			signal.Notify(cs, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-cs
				if err := options.deinit(); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), err)
					os.Exit(1)
				}
				os.Exit(0)
			}()

			if err := options.run(cmd, args); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
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
	cmd.Flags().String("integration-directory", "",
		"Directory which holds the locally built integration and is the result of a local build action.")
	cmd.Flags().StringArrayP("env", "e", nil, "Flag to specify an environment variable [--env VARIABLE=value].")
	cmd.Flags().StringArray("property-file", nil, "Add a property file to the integration.")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a Camel property to the integration.")
	cmd.Flags().StringArray("maven-repository", nil, "Use a maven repository")

	return &cmd, &options
}

type localRunCmdOptions struct {
	*LocalCmdOptions
	Containerize         bool     `mapstructure:"containerize"`
	Image                string   `mapstructure:"image"`
	Network              string   `mapstructure:"network"`
	IntegrationDirectory string   `mapstructure:"integration-directory"`
	EnvironmentVariables []string `mapstructure:"envs"`
	PropertyFiles        []string `mapstructure:"property-files"`
	Properties           []string `mapstructure:"properties"`
	MavenRepositories    []string `mapstructure:"maven-repositories"`
}

func (o *localRunCmdOptions) validate(args []string) error {
	// Validate integration files when no image is provided and we are
	// not running an already locally-built integration.
	if o.Image == "" && o.IntegrationDirectory == "" {
		if len(args) == 0 {
			return errors.New("no integration files have been provided")
		}

		// Validate integration files.
		if err := validateFiles(args); err != nil {
			return err
		}
	}

	// Validate additional dependencies specified by the user.
	if err := validateDependencies(o.Dependencies); err != nil {
		return err
	}

	// Validate properties file.
	if err := validatePropertyFiles(o.PropertyFiles); err != nil {
		return err
	}

	// If containerize is set then docker image name must be set.
	if o.Containerize && o.Image == "" {
		return errors.New("containerization is active but no image name has been provided")
	}

	return nil
}

func (o *localRunCmdOptions) init() error {
	if o.Containerize {
		if err := createDockerBaseWorkingDirectory(); err != nil {
			return err
		}

		if err := createDockerWorkingDirectory(); err != nil {
			return err
		}
	}

	setDockerNetworkName(o.Network)

	setDockerEnvVars(o.EnvironmentVariables)

	return createMavenWorkingDirectory()
}

func (o *localRunCmdOptions) run(cmd *cobra.Command, args []string) error {
	// If local run is provided with an image name, it will just run the image locally and exit.
	if o.Image != "" && !o.Containerize {
		// Run image locally.
		return runIntegrationImage(o.Context, o.Image, cmd.OutOrStdout(), cmd.ErrOrStderr())
	}

	dependencies, err := o.processDependencies(args)
	if err != nil {
		return err
	}
	propertyFiles, err := o.processPropertyFiles()
	if err != nil {
		return err
	}
	routes, err := o.processRoutes(args)
	if err != nil {
		return err
	}

	if o.Containerize {
		// If this is a containerized local run, create, build and run the container image.
		if err := createAndBuildIntegrationImage(o.Context, "", false, o.Image,
			propertyFiles, dependencies, routes, o.IntegrationDirectory != "",
			cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
			return err
		}

		// Run integration image.
		return runIntegrationImage(o.Context, o.Image, cmd.OutOrStdout(), cmd.ErrOrStderr())
	}

	// Run integration locally.
	return runLocalIntegration(o.Context, propertyFiles, dependencies, routes, o.getPropertiesDir(),
		cmd.OutOrStdout(), cmd.ErrOrStderr())
}

func (o *localRunCmdOptions) processDependencies(args []string) ([]string, error) {
	if o.IntegrationDirectory == "" {
		return getDependencies(o.Context, args, o.Dependencies, o.MavenRepositories, true)
	}

	// Set up on the integration directory

	// Fetch local dependencies
	dependencies, err := getLocalBuildDependencies(o.IntegrationDirectory)
	if err != nil {
		return nil, err
	}
	if err := o.setupDependenciesForQuarkusRun(); err != nil {
		return dependencies, err
	}

	return dependencies, nil
}

// setupDependenciesForQuarkusRun sets up resources under the integration directory for running Quarkus app.
func (o *localRunCmdOptions) setupDependenciesForQuarkusRun() error {
	// Local dependencies directory
	localDependenciesDir := getCustomDependenciesDir(o.IntegrationDirectory)

	// The quarkus application files need to be at a specific location i.e.:
	// <integration_directory>/../quarkus/quarkus-application.dat
	// <integration_directory>/../quarkus/generated-bytecode.jar
	localQuarkusDir := getCustomQuarkusDir(o.IntegrationDirectory)
	if err := copyQuarkusAppFiles(localDependenciesDir, localQuarkusDir); err != nil {
		return err
	}

	// The dependency jar files need to be at a specific location i.e.:
	// <integration_directory>/../lib/main/*.jar
	localLibDir := getCustomLibDir(o.IntegrationDirectory)
	if err := copyLibFiles(localDependenciesDir, localLibDir); err != nil {
		return err
	}

	// The Camel K jar file needs to be at a specific location i.e.:
	// <integration_directory>/../app/camel-k-integration-X.X.X{-SNAPSHOT}.jar
	localAppDir := getCustomAppDir(o.IntegrationDirectory)
	if err := copyAppFile(localDependenciesDir, localAppDir); err != nil {
		return err
	}

	return nil
}

// processPropertyFiles processes integration properties which may come from files or CLI.
func (o *localRunCmdOptions) processPropertyFiles() ([]string, error) {
	propertyFiles := o.PropertyFiles
	hasIntegrationDir := o.IntegrationDirectory != ""
	if hasIntegrationDir {
		localPropertyFiles, err := getLocalBuildProperties(o.IntegrationDirectory)
		if err != nil {
			return nil, err
		}
		propertyFiles = localPropertyFiles
	}

	return updateIntegrationProperties(o.Properties, propertyFiles, hasIntegrationDir)
}

func (o *localRunCmdOptions) processRoutes(args []string) ([]string, error) {
	if o.IntegrationDirectory == "" {
		return args, nil
	}

	return getLocalBuildRoutes(o.IntegrationDirectory)
}

func (o *localRunCmdOptions) getPropertiesDir() string {
	if o.IntegrationDirectory == "" {
		return getLocalPropertiesDir()
	}

	return getCustomPropertiesDir(o.IntegrationDirectory)
}

func (o *localRunCmdOptions) deinit() error {
	if o.Containerize {
		if err := deleteDockerBaseWorkingDirectory(); err != nil {
			return err
		}

		if err := deleteDockerWorkingDirectory(); err != nil {
			return err
		}
	}

	if o.IntegrationDirectory != "" {
		if err := deleteLocalIntegrationDirs(o.IntegrationDirectory); err != nil {
			return err
		}
	}

	return deleteMavenWorkingDirectory()
}

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

	"github.com/apache/camel-k/pkg/cmd/local"
	"github.com/apache/camel-k/pkg/util"
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
	cmd.Flags().String("image", "", usageImage)
	cmd.Flags().String("network", "", "Custom network name to be used by the underlying Docker command.")
	cmd.Flags().String("integration-directory", "", usageIntegrationDirectory)
	cmd.Flags().StringArrayP("env", "e", nil, "Flag to specify an environment variable [--env VARIABLE=value].")
	cmd.Flags().StringArray("property-file", nil, usagePropertyFile)
	cmd.Flags().StringArrayP("property", "p", nil, usageProperty)

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
}

func (o *localRunCmdOptions) validate(args []string) error {
	if len(args) == 0 && o.IntegrationDirectory == "" && o.Image == "" {
		return errors.New("either integration files, --image, or --integration-directory must be provided")
	}

	// If containerize is set then docker image name must be set.
	if o.Containerize && o.Image == "" {
		return errors.New("--containerize requires --image")
	}

	// Validate integration files.
	if err := local.ValidateFiles(args); err != nil {
		return err
	}

	// Validate additional dependencies specified by the user.
	if err := local.ValidateDependencies(o.Dependencies); err != nil {
		return err
	}

	// Validate properties file.
	if err := local.ValidatePropertyFiles(o.PropertyFiles); err != nil {
		return err
	}

	if o.IntegrationDirectory != "" {
		if ok, err := util.DirectoryExists(o.IntegrationDirectory); err != nil {
			return err
		} else if !ok {
			return errors.Errorf("integration directory %q does not exist", o.IntegrationDirectory)
		}
	}

	return nil
}

func (o *localRunCmdOptions) init() error {
	if o.Containerize {
		if err := local.CreateDockerBaseWorkingDirectory(); err != nil {
			return err
		}
		if err := local.CreateDockerWorkingDirectory(); err != nil {
			return err
		}
	}
	local.SetDockerNetworkName(o.Network)
	local.SetDockerEnvVars(o.EnvironmentVariables)

	return local.CreateMavenWorkingDirectory()
}

func (o *localRunCmdOptions) run(cmd *cobra.Command, args []string) error {
	// If local run is provided with an image name, it will just run the image locally and exit.
	if o.Image != "" && !o.Containerize {
		// Run image locally.
		return local.RunIntegrationImage(o.Context, o.Image, cmd.OutOrStdout(), cmd.ErrOrStderr())
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
		// Create, build, and run the container image.
		if err := local.CreateAndBuildIntegrationImage(o.Context, "", false, o.Image,
			propertyFiles, dependencies, routes, o.IntegrationDirectory != "",
			cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
			return err
		}

		return local.RunIntegrationImage(o.Context, o.Image, cmd.OutOrStdout(), cmd.ErrOrStderr())
	}

	// Run integration locally.
	return local.RunLocalIntegration(o.Context, propertyFiles, dependencies, routes, o.getPropertiesDir(),
		cmd.OutOrStdout(), cmd.ErrOrStderr())
}

func (o *localRunCmdOptions) processDependencies(args []string) ([]string, error) {
	if o.IntegrationDirectory == "" {
		return local.GetDependencies(o.Context, args, o.Dependencies, o.MavenRepositories, true)
	}

	// Set up on the integration directory

	// Fetch local dependencies
	dependencies, err := local.GetBuildDependencies(o.IntegrationDirectory)
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
	localDependenciesDir := local.GetCustomDependenciesDir(o.IntegrationDirectory)

	// The quarkus application files need to be at a specific location i.e.:
	// <integration_directory>/../quarkus/quarkus-application.dat
	// <integration_directory>/../quarkus/generated-bytecode.jar
	localQuarkusDir := local.GetCustomQuarkusDir(o.IntegrationDirectory)
	if err := local.CopyQuarkusAppFiles(localDependenciesDir, localQuarkusDir); err != nil {
		return err
	}

	// The dependency jar files need to be at a specific location i.e.:
	// <integration_directory>/../lib/main/*.jar
	localLibDir := local.GetCustomLibDir(o.IntegrationDirectory)
	if err := local.CopyLibFiles(localDependenciesDir, localLibDir); err != nil {
		return err
	}

	// The Camel K jar file needs to be at a specific location i.e.:
	// <integration_directory>/../app/camel-k-integration-X.X.X{-SNAPSHOT}.jar
	localAppDir := local.GetCustomAppDir(o.IntegrationDirectory)
	if err := local.CopyAppFile(localDependenciesDir, localAppDir); err != nil {
		return err
	}

	return nil
}

// processPropertyFiles processes integration properties which may come from files or CLI.
func (o *localRunCmdOptions) processPropertyFiles() ([]string, error) {
	propertyFiles := o.PropertyFiles
	hasIntegrationDir := o.IntegrationDirectory != ""
	if hasIntegrationDir {
		localPropertyFiles, err := local.GetBuildProperties(o.IntegrationDirectory)
		if err != nil {
			return nil, err
		}
		propertyFiles = localPropertyFiles
	}

	return local.UpdateIntegrationProperties(o.Properties, propertyFiles, hasIntegrationDir)
}

func (o *localRunCmdOptions) processRoutes(args []string) ([]string, error) {
	if o.IntegrationDirectory == "" {
		return args, nil
	}

	return local.GetBuildRoutes(o.IntegrationDirectory)
}

func (o *localRunCmdOptions) getPropertiesDir() string {
	if o.IntegrationDirectory == "" {
		return local.GetLocalPropertiesDir()
	}

	return local.GetCustomPropertiesDir(o.IntegrationDirectory)
}

func (o *localRunCmdOptions) deinit() error {
	if o.Containerize {
		if err := local.DeleteDockerBaseWorkingDirectory(); err != nil {
			return err
		}

		if err := local.DeleteDockerWorkingDirectory(); err != nil {
			return err
		}
	}

	if o.IntegrationDirectory != "" {
		if err := local.DeleteLocalIntegrationDirs(o.IntegrationDirectory); err != nil {
			return err
		}
	}

	return local.DeleteMavenWorkingDirectory()
}

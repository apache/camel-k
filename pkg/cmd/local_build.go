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

func newCmdLocalBuild(localCmdOptions *LocalCmdOptions) (*cobra.Command, *localBuildCmdOptions) {
	options := localBuildCmdOptions{
		LocalCmdOptions: localCmdOptions,
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

	cmd.Flags().Bool("base-image", false, "Build base image used as a starting point for any integration.")
	cmd.Flags().Bool("dependencies-only", false,
		"Only output the integration dependencies. The integration-directory flag must be set.")
	cmd.Flags().String("container-registry", "",
		"Registry that holds intermediate images. This flag should only be used in conjunction with the base-image flag.")
	cmd.Flags().String("image", "", "Full path to integration image including registry.")
	cmd.Flags().String("integration-directory", "", "Directory to hold local integration files.")
	cmd.Flags().StringArray("property-file", nil, "Add a property file to the integration.")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a Camel property to the integration.")
	cmd.Flags().StringArray("maven-repository", nil, "Use a maven repository")

	return &cmd, &options
}

type localBuildCmdOptions struct {
	*LocalCmdOptions
	BaseImage            bool     `mapstructure:"base-image"`
	DependenciesOnly     bool     `mapstructure:"dependencies-only"`
	ContainerRegistry    string   `mapstructure:"container-registry"`
	Image                string   `mapstructure:"image"`
	IntegrationDirectory string   `mapstructure:"integration-directory"`
	Properties           []string `mapstructure:"properties"`
	PropertyFiles        []string `mapstructure:"property-files"`
	MavenRepositories    []string `mapstructure:"maven-repositories"`
}

func (o *localBuildCmdOptions) validate(args []string) error {
	// Validate integration files.
	if len(args) > 0 {
		if err := validateFiles(args); err != nil {
			return err
		}
	}

	// Validate additional dependencies specified by the user.
	if err := validateDependencies(o.Dependencies); err != nil {
		return err
	}

	// Validate properties file.
	if err := validateFiles(o.PropertyFiles); err != nil {
		return err
	}

	if o.BaseImage {
		// Cannot have both integration files and the base image construction enabled.
		if len(args) > 0 {
			return errors.New("integration files have been provided and the base image construction is enabled")
		}

		// Docker registry must be set.
		if o.ContainerRegistry == "" {
			return errors.New("base image cannot be built because container registry has not been provided")
		}

		// If an integration directory is provided then no base image containerization can be enabled.
		if o.IntegrationDirectory != "" {
			return errors.New("base image construction does not use integration files")
		}
	} else if o.ContainerRegistry != "" {
		// ContainerRegistry should only be specified when building the base image.
		return errors.New("cannot specify container registry unless a base integration image is being built")
	}

	// The integration directory must be set when only outputting dependencies.
	if o.DependenciesOnly && o.IntegrationDirectory == "" {
		return errors.New("to output dependencies the integration directory flag must be set")
	}

	return nil
}

func (o *localBuildCmdOptions) init(args []string) error {
	// Create integration directory if one is provided.
	if o.IntegrationDirectory != "" {
		if err := util.CreateDirectory(o.IntegrationDirectory); err != nil {
			return err
		}
	}

	if o.BaseImage || o.Image != "" {
		// If base image construction is enabled create a directory for it.
		if err := createDockerBaseWorkingDirectory(); err != nil {
			return err
		}

		// If integration image construction is enabled, an integration image will be built.
		if o.Image != "" {
			if err := createDockerWorkingDirectory(); err != nil {
				return err
			}
		}
	}

	return createMavenWorkingDirectory()
}

func (o *localBuildCmdOptions) run(cmd *cobra.Command, args []string) error {
	var dependenciesList, propertyFilesList []string
	routeFiles := args

	if !o.BaseImage {
		dependencies, err := getDependencies(o.Context, args, o.Dependencies, o.MavenRepositories, true)
		if err != nil {
			return err
		}

		var propertyFiles []string
		if !o.DependenciesOnly {
			// Manage integration properties which may come from files or CLI
			propertyFiles, err = updateIntegrationProperties(o.Properties, o.PropertyFiles, false)
			if err != nil {
				return err
			}
		}

		dependenciesList = dependencies
		propertyFilesList = propertyFiles

		// Integration directory can only be used when building an integration image or when we just
		// build the integration without also building the image. A local build of the integration is
		// represented by all the files that define the integration: dependencies, properties, and routes.
		if o.IntegrationDirectory != "" {
			localDependenciesDir := getCustomDependenciesDir(o.IntegrationDirectory)
			dependenciesList, err = copyIntegrationFilesToDirectory(dependencies, localDependenciesDir)
			if err != nil {
				return err
			}

			if o.DependenciesOnly {
				// Once dependencies have been copied to local folder, we can exit.
				return nil
			}

			localPropertiesDir := getCustomPropertiesDir(o.IntegrationDirectory)
			propertyFilesList, err = copyIntegrationFilesToDirectory(propertyFiles, localPropertiesDir)
			if err != nil {
				return err
			}

			localRoutesDir := getCustomRoutesDir(o.IntegrationDirectory)
			routeFiles, err = copyIntegrationFilesToDirectory(args, localRoutesDir)
			if err != nil {
				return err
			}

			// The only case in which we should not execute the integration image creation is when we want to
			// just output the files that comprise the integration locally.
			if o.Image == "" {
				return nil
			}
		}
	}

	if err := createAndBuildIntegrationImage(o.Context, o.ContainerRegistry, o.BaseImage, o.Image,
		propertyFilesList, dependenciesList, routeFiles, false,
		cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
		return err
	}

	return nil
}

func (o *localBuildCmdOptions) deinit() error {
	// If base image construction is enabled delete the directory for it.
	if err := deleteDockerBaseWorkingDirectory(); err != nil {
		return err
	}

	// If integration files are provided delete the maven project folder.
	if !o.BaseImage {
		if err := deleteDockerWorkingDirectory(); err != nil {
			return err
		}
		if err := deleteMavenWorkingDirectory(); err != nil {
			return err
		}
	}

	return nil
}

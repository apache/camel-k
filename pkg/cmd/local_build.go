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

	"github.com/apache/camel-k/pkg/cmd/local"
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
			if err := options.validate(cmd, args); err != nil {
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

	cmd.Flags().Bool("base-image", false, "Build base image used as a starting point for any integration")
	cmd.Flags().Bool("dependencies-only", false,
		"Only output the integration dependencies. The integration-directory flag must be set.")
	cmd.Flags().String("container-registry", "",
		"Registry that holds intermediate images. This flag should only be used in conjunction with the base-image flag.")
	cmd.Flags().String("image", "", usageImage)
	cmd.Flags().String("integration-directory", "", usageIntegrationDirectory)
	cmd.Flags().StringArray("property-file", nil, usagePropertyFile)
	cmd.Flags().StringArrayP("property", "p", nil, usageProperty)

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
}

func (o *localBuildCmdOptions) validate(cmd *cobra.Command, args []string) error {
	if o.BaseImage {
		return o.validateBaseImageMode(cmd, args)
	}

	return o.validateIntegrationMode(args)
}

func (o *localBuildCmdOptions) validateBaseImageMode(cmd *cobra.Command, args []string) error {
	// Cannot have both integration files and the base image construction enabled.
	if len(args) > 0 {
		return errors.New("cannot use --base-image with integration files")
	}

	// Docker registry must be set.
	if o.ContainerRegistry == "" {
		return errors.New("--base-image requires --container-registry")
	}

	// If an integration directory is provided then no base image containerization can be enabled.
	if o.IntegrationDirectory != "" {
		return errors.New("cannot use --integration-directory with --base-image")
	}

	if o.DependenciesOnly {
		return errors.New("cannot use --dependencies-only with --base-image")
	}

	if len(o.Dependencies) > 0 || len(o.PropertyFiles) > 0 || len(o.Properties) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(),
			"Warning: --dependency, --property, and --property-file are ignored in --base-image mode")
	}

	return nil
}

func (o *localBuildCmdOptions) validateIntegrationMode(args []string) error {
	if len(args) == 0 {
		if o.IntegrationDirectory == "" {
			return errors.New("either integration files, --integration-directory, or --base-image must be provided")
		}
	} else {
		if o.IntegrationDirectory == "" && o.Image == "" {
			return errors.New("either --integration-directory or --image must be provided with integration files")
		}
	}

	if o.ContainerRegistry != "" {
		// ContainerRegistry should only be specified when building the base image.
		return errors.New("--container-registry must be used with --base-image")
	}

	// The integration directory must be set when only outputting dependencies.
	if o.DependenciesOnly && o.IntegrationDirectory == "" {
		return errors.New("--dependencies-only requires --integration-directory")
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
	if err := local.ValidateFiles(o.PropertyFiles); err != nil {
		return err
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
		if err := local.CreateDockerBaseWorkingDirectory(); err != nil {
			return err
		}

		// If integration image construction is enabled, an integration image will be built.
		if o.Image != "" {
			if err := local.CreateDockerWorkingDirectory(); err != nil {
				return err
			}
		}
	}

	return local.CreateMavenWorkingDirectory()
}

func (o *localBuildCmdOptions) run(cmd *cobra.Command, args []string) error {
	var dependenciesList, propertyFilesList []string
	routeFiles := args

	if !o.BaseImage {
		dependencies, err := local.GetDependencies(o.Context, args, o.Dependencies, o.MavenRepositories, true)
		if err != nil {
			return err
		}

		var propertyFiles []string
		if !o.DependenciesOnly {
			// Manage integration properties which may come from files or CLI
			propertyFiles, err = local.UpdateIntegrationProperties(o.Properties, o.PropertyFiles, false)
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
			localDependenciesDir := local.GetCustomDependenciesDir(o.IntegrationDirectory)
			dependenciesList, err = local.CopyIntegrationFilesToDirectory(dependencies, localDependenciesDir)
			if err != nil {
				return err
			}

			if o.DependenciesOnly {
				// Once dependencies have been copied to local folder, we can exit.
				return nil
			}

			localPropertiesDir := local.GetCustomPropertiesDir(o.IntegrationDirectory)
			propertyFilesList, err = local.CopyIntegrationFilesToDirectory(propertyFiles, localPropertiesDir)
			if err != nil {
				return err
			}

			localRoutesDir := local.GetCustomRoutesDir(o.IntegrationDirectory)
			routeFiles, err = local.CopyIntegrationFilesToDirectory(args, localRoutesDir)
			if err != nil {
				return err
			}

			// The only case in which we should not execute the integration image creation is when we want to
			// just output the files that comprise the integration locally.
			if o.Image == "" {
				fmt.Fprintf(cmd.OutOrStdout(),
					"Integration directory generated: %s\n", o.IntegrationDirectory)
				return nil
			}
		}
	}

	if err := local.CreateAndBuildIntegrationImage(o.Context, o.ContainerRegistry, o.BaseImage, o.Image,
		propertyFilesList, dependenciesList, routeFiles, false,
		cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
		return err
	}

	return nil
}

func (o *localBuildCmdOptions) deinit() error {
	// If base image construction is enabled delete the directory for it.
	if err := local.DeleteDockerBaseWorkingDirectory(); err != nil {
		return err
	}

	// If integration files are provided delete the maven project folder.
	if !o.BaseImage {
		if err := local.DeleteDockerWorkingDirectory(); err != nil {
			return err
		}
		if err := local.DeleteMavenWorkingDirectory(); err != nil {
			return err
		}
	}

	return nil
}

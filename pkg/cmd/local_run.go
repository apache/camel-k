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
	cmd.Flags().String("image-name", "", "Integration image name.")
	cmd.Flags().String("docker-registry", "", "Docker registry to store intermediate images.")
	cmd.Flags().StringArray("property-file", nil, "Add a property file to the integration.")
	cmd.Flags().StringArrayP("property", "p", nil, "Add a Camel property to the integration.")
	cmd.Flags().StringArrayP("dependency", "d", nil, additionalDependencyUsageMessage)

	return &cmd, &options
}

type localRunCmdOptions struct {
	*RootCmdOptions
	Containerize           bool     `mapstructure:"containerize"`
	ImageName              string   `mapstructure:"image-name"`
	DockerRegistry         string   `mapstructure:"docker-registry"`
	PropertyFiles          []string `mapstructure:"property-files"`
	Properties             []string `mapstructure:"properties"`
	AdditionalDependencies []string `mapstructure:"dependencies"`
}

func (command *localRunCmdOptions) validate(args []string) error {
	// Validate integration files.
	if command.ImageName == "" || command.Containerize {
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

	// If containerize is set then docker registry must be set.
	if command.Containerize && command.DockerRegistry == "" {
		return errors.New("containerization is active but no registry has been provided")
	}

	// If containerize is set then docker image name must be set.
	if command.Containerize && command.ImageName == "" {
		return errors.New("containerization is active but no image name has been provided")
	}

	// If ImageName is provided then docker registry must be set.
	if command.ImageName != "" && command.DockerRegistry == "" {
		return errors.New("cannot get image as no registry has been provided")
	}

	return nil
}

func (command *localRunCmdOptions) init() error {
	return createMavenWorkingDirectory()
}

func (command *localRunCmdOptions) run(args []string) error {
	// If local run is provided with an image name, it will just run the image locally and exit.
	if command.ImageName != "" && !command.Containerize {
		// Run image locally.
		err := runIntegrationImage(command.DockerRegistry, command.ImageName)
		if err != nil {
			return err
		}

		return nil
	}

	// Fetch dependencies.
	dependencies, err := getDependencies(args, command.AdditionalDependencies, true)
	if err != nil {
		return err
	}

	// Manage integration properties which may come from files or CLI.
	propertyFiles, err := updateIntegrationProperties(command.Properties, command.PropertyFiles)
	if err != nil {
		return err
	}

	// If this is a containerized local run, create, build and run the container image.
	if command.Containerize {
		// Create and build integration image.
		err = createAndBuildIntegrationImage(command.DockerRegistry, false, command.ImageName,
			propertyFiles, dependencies, args)
		if err != nil {
			return err
		}

		// Run integratgion image.
		err = runIntegrationImage(command.DockerRegistry, command.ImageName)
		if err != nil {
			return err
		}
	} else {
		// Run integration locally.
		err = RunLocalIntegrationRunCommand(propertyFiles, dependencies, args)
		if err != nil {
			return err
		}
	}

	return nil
}

func (command *localRunCmdOptions) deinit() error {
	return deleteMavenWorkingDirectory()
}

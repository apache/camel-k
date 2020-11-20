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

	cmd.Flags().Bool("base-image", false, "Create base image used as a starting point for any integration.")
	cmd.Flags().String("docker-registry", "", "Docker registry to store intermediate images.")

	return &cmd, &options
}

type localCreateCmdOptions struct {
	*RootCmdOptions
	BaseImage      bool   `mapstructure:"base-image"`
	DockerRegistry string `mapstructure:"docker-registry"`
}

func (command *localCreateCmdOptions) validate(args []string) error {
	// If containerize is set then docker registry must be set.
	if command.BaseImage && command.DockerRegistry == "" {
		return errors.New("base image cannot be created as no registry has been provided")
	}

	return nil
}

func (command *localCreateCmdOptions) init() error {
	return createDockerBaseWorkingDirectory()
}

func (command *localCreateCmdOptions) run(args []string) error {
	// Create the Dockerfile and build the base image.
	if command.BaseImage {
		err := createAndBuildBaseImage(command.DockerRegistry)
		if err != nil {
			return err
		}
	}

	return nil
}

func (command *localCreateCmdOptions) deinit() error {
	return deleteDockerBaseWorkingDirectory()
}

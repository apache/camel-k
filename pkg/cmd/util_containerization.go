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
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/apache/camel-k/pkg/util/docker"
	"github.com/pkg/errors"
)

/// Local Docker file system management functions.

func createDockerBaseWorkingDirectory() error {
	// Create local docker base directory.
	temporaryDirectory, err := ioutil.TempDir(os.TempDir(), "docker-base-")
	if err != nil {
		return err
	}

	// Set the Docker base directory to the default value.
	docker.BaseWorkingDirectory = temporaryDirectory

	return nil
}

func deleteDockerBaseWorkingDirectory() error {
	// Remove directory used for computing the dependencies.
	defer os.RemoveAll(docker.BaseWorkingDirectory)

	return nil
}

func createDockerWorkingDirectory() error {
	// Create local docker base directory.
	temporaryDirectory, err := ioutil.TempDir(os.TempDir(), "docker-")
	if err != nil {
		return err
	}

	// Set the Docker base directory to the default value.
	docker.IntegrationWorkingDirectory = temporaryDirectory

	return nil
}

func deleteDockerWorkingDirectory() error {
	// Remove directory used for computing the dependencies.
	defer os.RemoveAll(docker.IntegrationWorkingDirectory)

	return nil
}

func createAndBuildBaseImage(containerRegistry string) error {
	// Create the base image Docker file.
	err := docker.CreateBaseImageDockerFile()
	if err != nil {
		return err
	}

	// Get the Docker command arguments for building the base image and create the command.
	args := docker.BuildBaseImageArgs()
	cmd := exec.CommandContext(ctx, "docker", args...)

	// Output executed command.
	fmt.Printf("Executing: " + strings.Join(cmd.Args, " ") + "\n")

	// Run the command.
	if err := cmd.Run(); err != nil {
		return errors.Errorf("base image containerization did not run successfully: %v", err)
	}

	return nil
}

func createAndBuildIntegrationImage(containerRegistry string, justBaseImage bool, image string,
	propertyFiles []string, dependencies []string, routes []string) error {
	// This ensures the Dockerfile for the base image will not end up in an undesired location.
	if docker.BaseWorkingDirectory == "" {
		return errors.New("base directory that holds the base image Dockerfile has not been set correctly")
	}

	docker.RegistryName = containerRegistry
	if !justBaseImage {
		registryName, err := docker.ExtractRegistryName(image)
		if err != nil {
			return err
		}

		docker.RegistryName = registryName
	}

	// Create the Dockerfile and build the base image.
	err := createAndBuildBaseImage(containerRegistry)
	if err != nil {
		return err
	}

	if justBaseImage {
		return nil
	}

	if docker.IntegrationWorkingDirectory == "" {
		return errors.New("integration directory that holds the image Dockerfile has not been set correctly")
	}

	// Create integration image if integration files were provided.
	// Copy dependencies to a dependencies folder under a local directory.
	err = updateIntegrationDependencies(dependencies)
	if err != nil {
		return err
	}

	// Copy routes to a routes folder under a local directory.
	err = updateIntegrationRoutes(routes)
	if err != nil {
		return err
	}

	// Get integration run command to be run inside the container. This means the command
	// has to be created with the paths which will be valid inside the container.
	containerCmd := GetContainerIntegrationRunCommand(propertyFiles, dependencies, routes)

	// Create the integration image Docker file.
	err = docker.CreateIntegrationImageDockerFile(containerCmd)
	if err != nil {
		return err
	}

	// Get the Docker command arguments for building the base image and create the command.
	args := docker.BuildIntegrationImageArgs(image)
	cmd := exec.CommandContext(ctx, "docker", args...)

	// Set stdout and stderr.
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// Output executed command.
	fmt.Printf("Executing: " + strings.Join(cmd.Args, " ") + "\n")

	// Run the command.
	if err := cmd.Run(); err != nil {
		return errors.Errorf("integration image containerization did not run successfully: %v", err)
	}

	return nil
}

func runIntegrationImage(image string) error {
	// Get the docker command line argument for running an image.
	args := docker.RunIntegrationImageArgs(image)
	cmd := exec.CommandContext(ctx, "docker", args...)

	// Set stdout and stderr.
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// Output executed command.
	fmt.Printf("Executing: " + strings.Join(cmd.Args, " ") + "\n")

	// Run the command.
	if err := cmd.Run(); err != nil {
		return errors.Errorf("integration image did not run successfully: %v", err)
	}

	return nil
}

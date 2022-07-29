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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/docker"
	"github.com/pkg/errors"
)

// Local Docker file system management functions.

// createDockerBaseWorkingDirectory creates local docker base directory.
func createDockerBaseWorkingDirectory() error {
	temporaryDirectory, err := ioutil.TempDir(os.TempDir(), "docker-base-")
	if err != nil {
		return err
	}

	// Set the Docker base directory to the default value.
	docker.BaseWorkingDirectory = temporaryDirectory

	return nil
}

// deleteDockerBaseWorkingDirectory removes directory used for computing the base dependencies.
func deleteDockerBaseWorkingDirectory() error {
	return os.RemoveAll(docker.BaseWorkingDirectory)
}

// createDockerWorkingDirectory creates local docker directory.
func createDockerWorkingDirectory() error {
	temporaryDirectory, err := ioutil.TempDir(os.TempDir(), "docker-")
	if err != nil {
		return err
	}

	// Set the Docker base directory to the default value.
	docker.IntegrationWorkingDirectory = temporaryDirectory

	return nil
}

// deleteDockerWorkingDirectory removes directory used for computing the integration dependencies.
func deleteDockerWorkingDirectory() error {
	return os.RemoveAll(docker.IntegrationWorkingDirectory)
}

func setDockerNetworkName(networkName string) {
	if networkName != "" {
		docker.NetworkName = networkName
	}
}

func setDockerEnvVars(envVars []string) {
	if len(envVars) > 0 {
		util.CLIEnvVars = envVars
	}
}

func createAndBuildBaseImage(ctx context.Context, stdout, stderr io.Writer) error {
	// This ensures the Dockerfile for the base image will not end up in an undesired location.
	if docker.BaseWorkingDirectory == "" {
		return errors.New("base directory that holds the base image Dockerfile has not been set correctly")
	}

	// Create the base image Docker file.
	if err := docker.CreateBaseImageDockerFile(); err != nil {
		return err
	}

	return buildBaseImage(ctx, stdout, stderr)
}

func buildBaseImage(ctx context.Context, stdout, stderr io.Writer) error {
	// Get the Docker command arguments for building the base image and create the command.
	args := docker.BuildBaseImageArgs()
	cmd := exec.CommandContext(ctx, "docker", args...)

	// Set stdout and stderr.
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Output executed command.
	fmt.Fprintln(cmd.Stdout, "Executing:", strings.Join(cmd.Args, " "))

	// Run the command.
	if err := cmd.Run(); err != nil {
		return errors.Errorf("base image containerization did not run successfully: %v", err)
	}

	return nil
}

func setupDockerRegistry(containerRegistry string, image string, justBaseImage bool) error {
	docker.RegistryName = containerRegistry
	// If we build a normal image, i.e. not the base image, we need to parse
	// the location where images will be pushed.
	if !justBaseImage {
		registryName, err := docker.ExtractRegistryName(image)
		if err != nil {
			return err
		}
		docker.RegistryName = registryName
	}

	return nil
}

func createAndBuildIntegrationImage(ctx context.Context, containerRegistry string, justBaseImage bool, image string,
	propertyFiles []string, dependencies []string, routes []string, startsFromLocalFolder bool,
	stdout, stderr io.Writer) error {
	if err := setupDockerRegistry(containerRegistry, image, justBaseImage); err != nil {
		return err
	}

	// Create the Dockerfile and build the base image.
	if err := createAndBuildBaseImage(ctx, stdout, stderr); err != nil {
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
	if err := updateIntegrationDependencies(dependencies); err != nil {
		return err
	}

	// Copy routes to a routes folder under a local directory.
	if err := updateIntegrationRoutes(routes); err != nil {
		return err
	}

	// Copy quarkus files in maven subdirectory
	if err := updateQuarkusDirectory(); err != nil {
		return err
	}

	// Copy app files in maven subdirectory
	if err := updateAppDirectory(); err != nil {
		return err
	}

	// Copy lib files in maven subdirectory
	if err := updateLibDirectory(); err != nil {
		return err
	}

	// Get integration run command to be run inside the container. This means the command
	// has to be created with the paths which will be valid inside the container.
	containerCmd, err := getContainerIntegrationRunCommand(ctx, propertyFiles, dependencies, routes, stdout, stderr)
	if err != nil {
		return err
	}

	// Create the integration image Docker file.
	if err := docker.CreateIntegrationImageDockerFile(containerCmd, startsFromLocalFolder); err != nil {
		return err
	}

	return buildIntegrationImage(ctx, image, stdout, stderr)
}

func buildIntegrationImage(ctx context.Context, image string, stdout, stderr io.Writer) error {
	// Get the Docker command arguments for building the base image and create the command.
	args := docker.BuildIntegrationImageArgs(image, MavenWorkingDirectory)
	cmd := exec.CommandContext(ctx, "docker", args...)

	// Set stdout and stderr.
	cmd.Stderr = stderr
	cmd.Stdout = stdout

	// Output executed command.
	fmt.Fprintln(cmd.Stdout, "Executing:", strings.Join(cmd.Args, " "))

	// Run the command.
	if err := cmd.Run(); err != nil {
		return errors.Errorf("integration image containerization did not run successfully: %v", err)
	}

	return nil
}

func runIntegrationImage(ctx context.Context, image string, stdout, stderr io.Writer) error {
	// Stop the child process before exiting
	dockerCtx, cancel := context.WithCancel(ctx)
	cs := make(chan os.Signal, 1)
	signal.Notify(cs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-cs
		cancel()
	}()

	// Get the docker command line argument for running an image.
	args, err := docker.RunIntegrationImageArgs(image)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(dockerCtx, "docker", args...)

	// Set stdout and stderr.
	cmd.Stderr = stderr
	cmd.Stdout = stdout

	// Output executed command.
	fmt.Fprintln(cmd.Stdout, "Executing:", strings.Join(cmd.Args, " "))

	// Run the command.
	if err := cmd.Run(); err != nil {
		return errors.Errorf("integration image did not run successfully: %v", err)
	}

	return nil
}

// getContainerIntegrationRunCommand returns the integration command which will be run inside the container.
func getContainerIntegrationRunCommand(ctx context.Context, properties []string, dependencies []string, routes []string,
	stdout, stderr io.Writer) (*exec.Cmd, error) {
	// All paths need to be valid container paths.
	// Update property file paths.
	containerProperties := docker.ContainerizeFilePaths(properties, docker.GetContainerPropertiesDir())
	containerDependencies := docker.ContainerizeDependencyPaths(dependencies, docker.GetContainerDependenciesDir())
	containerRoutes := docker.ContainerizeFilePaths(routes, docker.GetContainerRoutesDir())

	return assembleIntegrationRunCommand(ctx, containerProperties, containerDependencies, containerRoutes,
		docker.GetContainerPropertiesDir(), stdout, stderr, false)
}

func runLocalIntegration(ctx context.Context, properties []string, dependencies []string, routes []string,
	propertiesDir string, stdout, stderr io.Writer) error {
	cmd, err := assembleIntegrationRunCommand(ctx, properties, dependencies, routes,
		propertiesDir, stdout, stderr, true)
	if err != nil {
		return err
	}

	// Output command we are about to run.
	fmt.Fprintln(cmd.Stdout, "Executing:", strings.Join(cmd.Args, " "))

	// Run integration locally.
	return cmd.Run()
}

func assembleIntegrationRunCommand(ctx context.Context, properties []string, dependencies []string, routes []string,
	propertiesDir string, stdout, stderr io.Writer, local bool) (*exec.Cmd, error) {
	// Create classpath value.
	classpathValue := assembleClasspathArgValue(properties, dependencies, routes)

	// Create java command that runs the integration.
	javaCmd := "java"

	// Create java command arguments.
	args := make([]string, 0)
	args = append(args, "-cp")
	args = append(args, classpathValue)
	args = append(args, "io.quarkus.bootstrap.runner.QuarkusEntryPoint")

	cmd := exec.CommandContext(ctx, javaCmd, args...)

	// Add directory where the properties files reside. The directory is the local properties directory
	// or the properties directory inside the container.
	cmd.Env = append(cmd.Env, "CAMEL_K_CONF_D="+propertiesDir)

	// Add files to the command line under the CAMEL_K_ROUTES flag.
	cmd.Env = append(cmd.Env, "CAMEL_K_ROUTES="+strings.Join(formatRoutes(routes), ","))

	// Add any lazily evaluated environment variables.
	if local {
		// If we are running locally then this is as late as we can evaluate the
		// lazy environment variables since we are going to run the command
		// immediately after the generation of these arguments.
		setEnvVars, err := util.EvaluateCLIAndLazyEnvVars()
		if err != nil {
			return nil, err
		}
		cmd.Env = append(cmd.Env, setEnvVars...)
	} else {
		// If we are running in containerized or just building an image, we should
		// not evaluate the variables at this point since we are only generating the
		// run command and not actually running it.
		for _, lazyEnvVar := range util.ListOfLazyEvaluatedEnvVars {
			cmd.Env = append(cmd.Env, lazyEnvVar+"={{env:"+lazyEnvVar+"}}")
		}
	}

	// Set stdout and stderr.
	cmd.Stderr = stderr
	cmd.Stdout = stdout

	return cmd, nil
}

func assembleClasspathArgValue(properties []string, dependencies []string, routes []string) string {
	classpathContents := []string{}
	classpathContents = append(classpathContents, properties...)
	classpathContents = append(classpathContents, routes...)
	classpathContents = append(classpathContents, dependencies...)
	return strings.Join(classpathContents, string(os.PathListSeparator))
}

func formatRoutes(files []string) []string {
	routes := []string{}
	for _, route := range files {
		// Split route path.
		a := strings.Split(route, ".")

		// Extract extension.
		extension := a[len(a)-1]

		// Add file with extension.
		routes = append(routes, "file:"+route+"?language="+extension)
	}

	return routes
}

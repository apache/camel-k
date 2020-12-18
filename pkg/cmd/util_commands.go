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
	"os/exec"
	"strings"

	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/docker"
)

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

func assembleClasspathArgValue(properties []string, dependencies []string, routes []string) string {
	classpathContents := []string{}
	classpathContents = append(classpathContents, properties...)
	classpathContents = append(classpathContents, routes...)
	classpathContents = append(classpathContents, dependencies...)
	return strings.Join(classpathContents, ":")
}

func assembleIntegrationRunCommand(ctx context.Context, properties []string, dependencies []string, routes []string, propertiesDir string, stdout, stderr io.Writer) *exec.Cmd {
	// Create classpath value.
	classpathValue := assembleClasspathArgValue(properties, dependencies, routes)

	// Create java command that runs the integration.
	javaCmd := "java"

	// Create java command arguments.
	args := make([]string, 0)
	args = append(args, "-cp")
	args = append(args, classpathValue)
	args = append(args, "io.quarkus.runner.GeneratedMain")

	cmd := exec.CommandContext(ctx, javaCmd, args...)

	// Add directory where the properties files reside. The directory is the local properties directory
	// or the properties directory inside the container.
	cmd.Env = append(cmd.Env, "CAMEL_K_CONF_D="+propertiesDir)

	// Add files to the command line under the CAMEL_K_ROUTES flag.
	cmd.Env = append(cmd.Env, "CAMEL_K_ROUTES="+strings.Join(formatRoutes(routes), ","))

	// Set stdout and stderr.
	cmd.Stderr = stderr
	cmd.Stdout = stdout

	return cmd
}

// RunLocalIntegrationRunCommand --
func RunLocalIntegrationRunCommand(ctx context.Context, properties []string, dependencies []string, routes []string, stdout, stderr io.Writer) error {
	cmd := assembleIntegrationRunCommand(ctx, properties, dependencies, routes, util.GetLocalPropertiesDir(), stdout, stderr)

	// Output command we are about to run.
	fmt.Printf("Executing: %s", strings.Join(cmd.Args, " "))

	// Run integration locally.
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// GetContainerIntegrationRunCommand --
func GetContainerIntegrationRunCommand(ctx context.Context, properties []string, dependencies []string, routes []string, stdout, stderr io.Writer) *exec.Cmd {
	// This is the integration command which will be run inside the container. Therefore all paths need to
	// be valid container paths.

	// Update property file paths.
	containerProperties := docker.ContainerizeFilePaths(properties, docker.GetContainerPropertiesDir())
	containerDependencies := docker.ContainerizeFilePaths(dependencies, docker.GetContainerDependenciesDir())
	containerRoutes := docker.ContainerizeFilePaths(routes, docker.GetContainerRoutesDir())

	return assembleIntegrationRunCommand(ctx, containerProperties, containerDependencies, containerRoutes, docker.GetContainerPropertiesDir(), stdout, stderr)
}

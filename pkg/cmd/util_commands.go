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
	"os"
	"os/exec"
	"path"
	"strings"
)

var (
	ctx9, cancel9 = context.WithCancel(context.Background()) // preemptive: kill subprocess
	ctx, cancel   = context.WithCancel(ctx9)                 // cooperative: wait for subprocess
)

func formatRoutes(files []string) []string {
	routes := []string{}
	for _, route := range files {
		// Split route path.
		a := strings.Split(route, ".")

		// Extract extension.
		extension := a[len(a)-1]

		// Add file if extension is supported.
		routes = append(routes, "file:"+route+"?language="+extension)
	}

	return routes
}

func confDirectories(properties []string) []string {
	confDirs := []string{}

	for _, propertiesPath := range properties {
		confDirs = append(confDirs, path.Dir(propertiesPath))
	}

	return confDirs
}

func assembleClasspatchArgValue(properties []string, dependencies []string, routes []string) string {
	classpathContents := []string{}
	classpathContents = append(classpathContents, properties...)
	classpathContents = append(classpathContents, routes...)
	classpathContents = append(classpathContents, dependencies...)
	return strings.Join(classpathContents, ":")
}

// GetIntegrationRunCommand --
func GetIntegrationRunCommand(properties []string, dependencies []string, routes []string) *exec.Cmd {
	// Create classpath value.
	classpathValue := assembleClasspatchArgValue(properties, dependencies, routes)

	// Create java command that runs the integration.
	javaCmd := "java"

	// Create java command arguments.
	args := make([]string, 0)
	args = append(args, "-cp")
	args = append(args, classpathValue)
	args = append(args, "io.quarkus.runner.GeneratedMain")

	cmd := exec.CommandContext(ctx, javaCmd, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// Add directory where the properties file resides.
	cmd.Env = append(cmd.Env, "CAMEL_K_CONF_D="+getPropertiesDir())

	// Add files to the command line under the CAMEL_K_ROUTES flag.
	cmd.Env = append(cmd.Env, "CAMEL_K_ROUTES="+strings.Join(formatRoutes(routes), ","))

	fmt.Printf("executing: %s", strings.Join(cmd.Args, " "))

	return cmd
}

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

package docker

import (
	"path"
	"strings"

	"github.com/apache/camel-k/pkg/util"
)

// CreateBaseImageDockerFile --
func CreateBaseImageDockerFile() error {
	dockerFile := []string{}

	// Base image is a java-only image since the integration command is just a java command.
	dockerFile = append(dockerFile, FROM("adoptopenjdk/openjdk11:alpine"))

	// Ensure Maven is already installed.
	dockerFile = append(dockerFile, RUNMavenInstall())

	// Write <base-work-dir>/Dockerfile
	baseDockerFilePath := path.Join(BaseWorkingDirectory, "Dockerfile")
	err := util.WriteToFile(baseDockerFilePath, strings.Join(dockerFile, "\n"))
	if err != nil {
		return err
	}

	return nil
}

// BuildBaseImageArgs --
func BuildBaseImageArgs() []string {
	// Construct the docker command:
	//
	// docker build -f <BaseWorkingDirectory>/Dockerfile -t <dockerRegistry>/<BaseImageName>
	//
	return BuildImageArgs(BaseWorkingDirectory, BaseImageName, BaseWorkingDirectory)
}

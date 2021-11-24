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
	"os/exec"
	"path"
	"strings"

	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/pkg/errors"
)

// CreateBaseImageDockerFile --.
func CreateBaseImageDockerFile() error {
	dockerFile := []string{}

	// Base image is a java-only image since the integration command is just a java command.
	dockerFile = append(dockerFile, FROM(defaults.BaseImage()))

	// Ensure Maven is already installed.
	dockerFile = append(dockerFile, RUNMavenInstall())

	// Write <BaseWorkingDirectory>/Dockerfile
	baseDockerFilePath := path.Join(BaseWorkingDirectory, "Dockerfile")
	err := util.WriteToFile(baseDockerFilePath, strings.Join(dockerFile, "\n"))
	if err != nil {
		return err
	}

	return nil
}

// CreateIntegrationImageDockerFile --.
func CreateIntegrationImageDockerFile(integrationRunCmd *exec.Cmd, startsFromLocalFolder bool) error {
	dockerFile := []string{}

	// Start from the base image that contains the maven install: <RegistryName>/<BaseImageName>
	dockerFile = append(dockerFile, FROM(GetFullDockerImage(GetBaseImagePath(), latestTag)))

	// Create container workspace directory.
	dockerFile = append(dockerFile, RUNMakeDir(GetContainerWorkspaceDir()))

	// Copy files from local directory to container directories.
	dockerFile = append(dockerFile, COPY(util.DefaultRoutesDirectoryName, GetContainerRoutesDir()))
	dockerFile = append(dockerFile, COPY(util.DefaultPropertiesDirectoryName, GetContainerPropertiesDir()))
	dockerFile = append(dockerFile, COPY(util.DefaultDependenciesDirectoryName, GetContainerDependenciesDir()))

	// Copy additional folder structure used for integrations built from a separately built local folder.
	if startsFromLocalFolder {
		dockerFile = append(dockerFile, RUNMakeDir(util.ContainerQuarkusDirectoryName))
		dockerFile = append(dockerFile, RUNMakeDir(util.ContainerLibDirectoryName))
		dockerFile = append(dockerFile, RUNMakeDir(util.ContainerAppDirectoryName))

		dockerFile = append(dockerFile, COPY(util.CustomQuarkusDirectoryName, util.ContainerQuarkusDirectoryName))
		dockerFile = append(dockerFile, COPY(util.CustomLibDirectoryName, util.ContainerLibDirectoryName))
		dockerFile = append(dockerFile, COPY(util.CustomAppDirectoryName, util.ContainerAppDirectoryName))
	}

	// All Env variables the command requires need to be set in the container.
	for _, keyValue := range integrationRunCmd.Env {
		values := strings.Split(keyValue, "=")
		dockerFile = append(dockerFile, ENV(values[0], strings.Join(values[1:], "=")))
	}

	// Compose command line.
	dockerFile = append(dockerFile, CMDShellWrap(strings.Join(integrationRunCmd.Args, " ")))

	// Write <IntegrationWorkingDirectory>/Dockerfile
	integrationDockerFilePath := path.Join(IntegrationWorkingDirectory, "Dockerfile")
	err := util.WriteToFile(integrationDockerFilePath, strings.Join(dockerFile, "\n"))
	if err != nil {
		return err
	}

	return nil
}

// BuildBaseImageArgs --.
func BuildBaseImageArgs() []string {
	// Construct the docker command:
	//
	// docker build -f <BaseWorkingDirectory>/Dockerfile -t <dockerRegistry>/<BaseImageName> <BaseWorkingDirectory>
	//
	// Add register
	return BuildImageArgs(BaseWorkingDirectory, GetBaseImagePath(), BaseWorkingDirectory)
}

// BuildIntegrationImageArgs --.
func BuildIntegrationImageArgs(imagePath string) []string {
	// Construct the docker command:
	//
	// docker build -f <BaseWorkingDirectory>/Dockerfile -t <imagePath> <MavenWorkingDirectory>
	//
	return BuildImageArgs(IntegrationWorkingDirectory, imagePath, util.MavenWorkingDirectory)
}

// RunIntegrationImageArgs --.
func RunIntegrationImageArgs(imagePath string) ([]string, error) {
	// Construct the docker command:
	//
	// docker run --network=<network-name> --env LAZY_ENV_VAR=value <dockerRegistry>/<ImageName>
	//
	return RunImageArgs(imagePath, latestTag)
}

// GetContainerWorkspaceDir -- directory inside the container where all the integration files are copied.
func GetContainerWorkspaceDir() string {
	return containerFileSeparator + util.DefaultWorkingDirectoryName
}

// GetContainerPropertiesDir -- directory inside the container where all the integration property files are copied.
func GetContainerPropertiesDir() string {
	return util.ContainerPropertiesDirectory
}

// GetContainerDependenciesDir -- directory inside the container where all the integration dependencies are copied.
func GetContainerDependenciesDir() string {
	return util.ContainerDependenciesDirectory
}

// GetContainerRoutesDir -- directory inside the container where all the integration routes are copied.
func GetContainerRoutesDir() string {
	return util.ContainerRoutesDirectory
}

// ContainerizeFilePaths -- make paths valid container paths given a valid container directory in newDir.
func ContainerizeFilePaths(currentFilePaths []string, newDir string) []string {
	newFilePaths := []string{}
	for _, currentFilePath := range currentFilePaths {
		newFilePaths = append(newFilePaths, newDir+containerFileSeparator+path.Base(currentFilePath))
	}

	return newFilePaths
}

// ContainerizeDependencyPaths -- make dependency paths valid given a valid container directory in newDir.
func ContainerizeDependencyPaths(dependencyPaths []string, newDir string) []string {
	newDependencyPaths := []string{}
	for _, currentDependencyPath := range dependencyPaths {
		newDependencyPath := util.SubstringFrom(currentDependencyPath, util.QuarkusDependenciesBaseDirectory)
		if newDependencyPath != "" {
			newDependencyPaths = append(newDependencyPaths, newDir+newDependencyPath)
		} else {
			newDependencyPaths = append(newDependencyPaths, newDir+containerFileSeparator+path.Base(currentDependencyPath))
		}
	}
	return newDependencyPaths
}

// ExtractRegistryName -- Extract registry name from image path.
func ExtractRegistryName(image string) (string, error) {
	pathComponents := strings.Split(image, containerFileSeparator)

	// There must be at least two components in the path:
	//  - docker.io/registry/imageName
	//  - localhost:5000/<dir>/imageName
	//  - localhost:5000/<dir>/.../<dir>/imageName
	//  - localhost:5000/imageName
	if len(pathComponents) < 2 {
		return "", errors.New("image path is too short, usage: registry/imageName or registry/*/imageName")
	}

	// Registry name is given by the path to the image.
	return strings.Join(pathComponents[0:len(pathComponents)-1], containerFileSeparator), nil
}

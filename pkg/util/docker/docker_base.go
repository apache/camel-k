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
)

// RegistryName -- the docker registry name.
var RegistryName = ""

// BaseImageName -- base image name.
var BaseImageName string = "integration-base-image"

// BaseWorkingDirectory -- directory used by Docker to construct the base image.
var BaseWorkingDirectory string = ""

// Internal variables.
var (
	dockerEndpointSeparator = "/"
	containerFileSeparator  = "/"
	latestTag               = "latest"
)

// BuildImageArgs - standard docker build arguments.
func BuildImageArgs(dockerFileDir string, imageName string, sourceDir string) []string {
	// Construct the docker command:
	//
	// docker build -f <docker-file> -t <image-name> <source-directory>
	//
	args := make([]string, 0)
	args = append(args, "build")

	// Add path to Dockerfile:
	dockerFile := path.Join(dockerFileDir, "Dockerfile")
	args = append(args, DockerfilePathArg(dockerFile)...)

	// Image name:
	args = append(args, ImageArg(imageName, "")...)

	// Root of source directory.
	if sourceDir != "" {
		args = append(args, sourceDir)
	}

	return args
}

// RunImageArgs -- standard docker run arguments.
func RunImageArgs(imageName string, imageTag string) []string {
	// Construct the docker command:
	//
	// docker run --network="host" <image-name>:<tag>
	//
	// TODO: support other types of network connections.
	args := make([]string, 0)
	args = append(args, "run")

	// TODO: support other networks.
	args = append(args, "--network=host")

	// Path to Docker image:
	args = append(args, ImageArg(imageName, imageTag)...)

	return args
}

//
// Arguments to docker command line.
//

// DockerfilePathArg --
func DockerfilePathArg(dockerfilePath string) []string {
	args := make([]string, 0)
	args = append(args, "-f")
	args = append(args, dockerfilePath)
	return args
}

// ImageArg --
func ImageArg(dockerImageName string, tag string) []string {
	args := make([]string, 0)
	args = append(args, "-t")
	args = append(args, GetFullDockerImage(dockerImageName, tag))
	return args
}

// LatestImageArg --
func LatestImageArg(dockerImageName string) []string {
	args := make([]string, 0)
	args = append(args, "-t")
	args = append(args, GetFullDockerImage(dockerImageName, latestTag))
	return args
}

//
// Docker-spcific helper functions.
//

// GetImage - <image-name>:<tag>
func GetImage(dockerImageName string, tag string) string {
	image := make([]string, 0)
	image = append(image, dockerImageName)
	image = append(image, tag)
	return strings.Join(image, ":")
}

// GetLatestImage - <image-name>:latest
func GetLatestImage(dockerImageName string) string {
	return GetImage(dockerImageName, latestTag)
}

// GetFullDockerImage - <docker-registry>/<image-name>:<tag>
func GetFullDockerImage(dockerImageName string, tag string) string {
	fullImagePath := make([]string, 0)
	fullImagePath = append(fullImagePath, RegistryName)
	if tag == "" {
		fullImagePath = append(fullImagePath, dockerImageName)
	} else {
		fullImagePath = append(fullImagePath, GetImage(dockerImageName, tag))
	}

	return strings.Join(fullImagePath, dockerEndpointSeparator)
}

//
// Container file management.
//

// JoinPath -- for container paths.
func JoinPath(lhsPath string, rhsPath string) string {
	p := []string{lhsPath, rhsPath}
	return strings.Join(p, containerFileSeparator)
}

//
// Docker syntax functions.
//

// Generic commands.

// COPY --
func COPY(from string, to string) string {
	c := []string{"COPY", from, to}
	return strings.Join(c, " ")
}

// RUN --
func RUN(command string) string {
	c := []string{"RUN", command}
	return strings.Join(c, " ")
}

// FROM --
func FROM(imageName string) string {
	c := []string{"FROM", imageName}
	return strings.Join(c, " ")
}

// WORKDIR --
func WORKDIR(workDir string) string {
	c := []string{"WORKDIR", workDir}
	return strings.Join(c, " ")
}

// ENV --
func ENV(envVar string, value string) string {
	p := []string{envVar, value}
	c := []string{"ENV", strings.Join(p, "=")}
	return strings.Join(c, " ")
}

// AS --
func AS(image string, alias string) string {
	c := []string{image, "as", alias}
	return strings.Join(c, " ")
}

// CMD --
func CMD(command string) string {
	c := []string{"CMD", command}
	return strings.Join(c, " ")
}

// // COPYFromBuilder --
// func COPYFromBuilder(from string, to string) string {
// 	flag := []string{"--from", internalBuilderImageName}
// 	newFrom := []string{strings.Join(flag, "="), from}
// 	return COPY(strings.Join(newFrom, " "), to)
// }

// RUNMavenInstall --
func RUNMavenInstall() string {
	return RUN("apk add --update maven && apk update && apk upgrade")
}

// RUNMakeDir --
func RUNMakeDir(dirName string) string {
	c := []string{"mkdir", "-p", dirName}
	return RUN(strings.Join(c, " "))
}

// ENVAppend --
func ENVAppend(envVar string, value string) string {
	tail := []string{value, "$" + envVar}
	return ENV(envVar, strings.Join(tail, ":"))
}

// CMDShellWrap --
func CMDShellWrap(command string) string {
	return CMD("/bin/sh -c \"" + command + "\"")
}

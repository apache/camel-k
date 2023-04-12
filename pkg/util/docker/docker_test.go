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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/stretchr/testify/assert"
)

func TestCreateBaseImageDockerFile(t *testing.T) {

	dockerfileExists, _ := util.FileExists("/tmp/Dockerfile")
	if dockerfileExists {
		os.Remove("/tmp/Dockerfile")
	}

	dockerFile := []string{}
	dockerFile = append(dockerFile, FROM(defaults.BaseImage()))
	dockerFile = append(dockerFile, RUNMavenInstall())
	expected := strings.Join(dockerFile, "\n")

	BaseWorkingDirectory = "/tmp/"

	err := CreateBaseImageDockerFile()
	assert.Nil(t, err)
	baseDockerFilePath := filepath.Join(BaseWorkingDirectory, "Dockerfile")

	c, err := util.ReadFile(baseDockerFilePath)

	assert.Nil(t, err)
	assert.Equal(t, expected, string(c))

	os.Remove("/tmp/Dockerfile")
}

func TestCreateIntegrationImageDockerFile(t *testing.T) {
	dockerfileExists, _ := util.FileExists("/tmp/Dockerfile")
	if dockerfileExists {
		os.Remove("/tmp/Dockerfile")
	}
	IntegrationWorkingDirectory = "/tmp/"
	var cmd exec.Cmd
	cmd.Path = "test"
	cmd.Env = []string{"key=value"}
	cmd.Args = []string{"-t"}
	dockerFile1 := []string{}
	dockerFile1 = append(dockerFile1, FROM(GetFullDockerImage(GetBaseImagePath(), latestTag)))
	dockerFile1 = append(dockerFile1, RUNMakeDir(GetContainerWorkspaceDir()))
	dockerFile1 = append(dockerFile1, COPY(util.DefaultRoutesDirectoryName, GetContainerRoutesDir()))
	dockerFile1 = append(dockerFile1, COPY(util.DefaultPropertiesDirectoryName, GetContainerPropertiesDir()))
	dockerFile1 = append(dockerFile1, COPY(util.DefaultDependenciesDirectoryName, GetContainerDependenciesDir()))

	dockerFile2 := dockerFile1
	dockerFile2 = append(dockerFile2, RUNMakeDir(util.ContainerQuarkusDirectoryName))
	dockerFile2 = append(dockerFile2, RUNMakeDir(util.ContainerLibDirectoryName))
	dockerFile2 = append(dockerFile2, RUNMakeDir(util.ContainerAppDirectoryName))
	dockerFile2 = append(dockerFile2, COPY(util.CustomQuarkusDirectoryName, util.ContainerQuarkusDirectoryName))
	dockerFile2 = append(dockerFile2, COPY(util.CustomLibDirectoryName, util.ContainerLibDirectoryName))
	dockerFile2 = append(dockerFile2, COPY(util.CustomAppDirectoryName, util.ContainerAppDirectoryName))

	for _, keyValue := range cmd.Env {
		values := strings.Split(keyValue, "=")
		dockerFile1 = append(dockerFile1, ENV(values[0], strings.Join(values[1:], "=")))
		dockerFile2 = append(dockerFile2, ENV(values[0], strings.Join(values[1:], "=")))
	}
	dockerFile1 = append(dockerFile1, CMDShellWrap(strings.Join(cmd.Args, " ")))
	dockerFile2 = append(dockerFile2, CMDShellWrap(strings.Join(cmd.Args, " ")))

	expected := strings.Join(dockerFile1, "\n")

	err := CreateIntegrationImageDockerFile(&cmd, false)
	assert.Nil(t, err)

	c, err := util.ReadFile("/tmp/Dockerfile")
	assert.Nil(t, err)
	assert.Equal(t, expected, string(c))

	os.Remove("/tmp/Dockerfile")

	expected = strings.Join(dockerFile2, "\n")

	err = CreateIntegrationImageDockerFile(&cmd, true)
	assert.Nil(t, err)

	c, err = util.ReadFile("/tmp/Dockerfile")
	assert.Nil(t, err)
	assert.Equal(t, expected, string(c))

	os.Remove("/tmp/Dockerfile")

}

func TestContainerizeFilePaths(t *testing.T) {
	expected := []string{"home/quarkus-app"}

	dependencyPaths := []string{"test1/quarkus-app"}
	newDir := "home"
	result := ContainerizeFilePaths(dependencyPaths, newDir)

	assert.Equal(t, expected, result)
}

func TestContainerizeDependencyPaths(t *testing.T) {
	expected := []string{"home/quarkus-app", "home/not-quarkus-app"}

	dependencyPaths := []string{"test1/quarkus-app", "not-quarkus-app"}
	newDir := "home"
	result := ContainerizeDependencyPaths(dependencyPaths, newDir)

	assert.Equal(t, expected, result)
}

func TestExtractRegistryName(t *testing.T) {
	expected := "localhost:5000"

	result, err := ExtractRegistryName("localhost:5000/imageName")
	assert.Nil(t, err)
	assert.Equal(t, expected, result)

	_, err = ExtractRegistryName("imageName")
	assert.NotNil(t, err)
}

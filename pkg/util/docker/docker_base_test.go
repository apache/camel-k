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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildImageArgs(t *testing.T) {

	args := BuildImageArgs("dockerfileDirectory", "imageName", "sourceDirectory")
	assert.Equal(t, "build", args[0])
	assert.Equal(t, "-f", args[1])
	assert.Equal(t, "dockerfileDirectory/Dockerfile", args[2])
	assert.Equal(t, "-t", args[3])
	assert.Equal(t, "imageName", args[4])
	assert.Equal(t, "sourceDirectory", args[5])
}

func TestRunImageArgs(t *testing.T) {

	args, err := RunImageArgs("imagePath", "tag")
	assert.Nil(t, err)
	assert.Equal(t, "run", args[0])
	assert.Equal(t, "--network="+NetworkName, args[1])
	assert.Equal(t, "-t", args[2])
	assert.Equal(t, "imagePath:tag", args[3])
}

func TestDockerfilePathArg(t *testing.T) {
	args := DockerfilePathArg("path/docker")
	assert.Equal(t, "-f", args[0])
	assert.Equal(t, "path/docker", args[1])
}

func TestImageArg(t *testing.T) {
	args := ImageArg("imageName", "tag")
	assert.Equal(t, "-t", args[0])
	assert.Equal(t, "imageName:tag", args[1])
}

func TestLatestImageArg(t *testing.T) {
	args := LatestImageArg("imageName")
	assert.Equal(t, "-t", args[0])
	assert.Equal(t, "imageName:latest", args[1])
}

func TestFullImageArg(t *testing.T) {
	args := FullImageArg("dir/imageName:tag")
	assert.Equal(t, "-t", args[0])
	assert.Equal(t, "dir/imageName:tag", args[1])

	args = FullImageArg("imageName:tag")
	assert.Equal(t, "-t", args[0])
	assert.Equal(t, "imageName:tag", args[1])

	args = FullImageArg("imageName")
	assert.Equal(t, "-t", args[0])
	assert.Equal(t, "imageName:latest", args[1])
}

func TestGetImage(t *testing.T) {
	image := GetImage("imageName", "tag")
	assert.Equal(t, "imageName:tag", image)
	assert.Equal(t, GetImage("imageName", "latest"), GetLatestImage("imageName"))
}

func TestGetFullDockerImage(t *testing.T) {
	image := GetFullDockerImage("dir/dir/imageName", "tag")
	assert.Equal(t, "dir/dir/imageName:tag", image)
	image = GetFullDockerImage("imageName", "tag")
	assert.Equal(t, "imageName:tag", image)
}

func TestJoinPath(t *testing.T) {
	path := JoinPath("path1", "dir/path2")
	assert.Equal(t, "path1/dir/path2", path)
}

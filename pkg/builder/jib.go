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

package builder

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/jib"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/maven"
)

type jibTask struct {
	c     client.Client
	build *v1.Build
	task  *v1.JibTask
}

var _ Task = &jibTask{}

func (t *jibTask) Do(ctx context.Context) v1.BuildStatus {
	status := v1.BuildStatus{}

	baseImage := t.build.Status.BaseImage
	if baseImage == "" {
		baseImage = t.task.BaseImage
		status.BaseImage = baseImage
	}

	contextDir := t.task.ContextDir
	if contextDir == "" {
		// Use the working directory.
		// This is useful when the task is executed in-container,
		// so that its WorkingDir can be used to share state and
		// coordinate with other tasks.
		pwd, err := os.Getwd()
		if err != nil {
			return status.Failed(err)
		}
		contextDir = filepath.Join(pwd, ContextDir)
	}

	exists, err := util.DirectoryExists(contextDir)
	if err != nil {
		return status.Failed(err)
	}
	empty, err := util.DirectoryEmpty(contextDir)
	if err != nil {
		return status.Failed(err)
	}
	if !exists || empty {
		// this can only indicate that there are no more resources to add to the base image,
		// because transitive resolution is the same even if spec differs.
		log.Infof("No new image to build, reusing existing image %s", baseImage)
		status.Image = baseImage
		return status
	}
	mavenDir := strings.ReplaceAll(contextDir, ContextDir, "maven")

	log.Debugf("Registry address: %s", t.task.Registry.Address)
	log.Debugf("Base image: %s", baseImage)

	registryConfigDir := ""
	if t.task.Registry.Secret != "" {
		registryConfigDir, err = MountSecret(ctx, t.c, t.build.Namespace, t.task.Registry.Secret)
		if err != nil {
			return status.Failed(err)
		}
	}

	if registryConfigDir != "" {
		if err := os.RemoveAll(registryConfigDir); err != nil {
			return status.Failed(err)
		}
	}

	// TODO refactor maven code to avoid creating a file to pass command args
	mavenCommand, err := util.ReadFile(filepath.Join(mavenDir, "MAVEN_CONTEXT"))
	if err != nil {
		return status.Failed(err)
	}

	mavenArgs := make([]string, 0)
	mavenArgs = append(mavenArgs, jib.JibMavenGoal)
	mavenArgs = append(mavenArgs, strings.Split(string(mavenCommand), " ")...)
	mavenArgs = append(mavenArgs, "-P", "jib")
	mavenArgs = append(mavenArgs, jib.JibMavenToImageParam+t.task.Image)
	mavenArgs = append(mavenArgs, jib.JibMavenFromImageParam+baseImage)
	if t.task.Registry.Insecure {
		mavenArgs = append(mavenArgs, jib.JibMavenInsecureRegistries+"true")
	}

	mvnCmd := "./mvnw"
	if c, ok := os.LookupEnv("MAVEN_CMD"); ok {
		mvnCmd = c
	}
	cmd := exec.CommandContext(ctx, mvnCmd, mavenArgs...)
	cmd.Dir = mavenDir

	myerror := util.RunAndLog(ctx, cmd, maven.MavenLogHandler, maven.MavenLogHandler)
	if myerror != nil {
		log.Errorf(myerror, "jib integration image containerization did not run successfully")
		return status.Failed(myerror)
	} else {
		log.Debug("jib integration image containerization did run successfully")
		status.Image = t.task.Image

		// retrieve image digest
		mavenDigest, errDigest := util.ReadFile(filepath.Join(mavenDir, jib.JibDigestFile))
		if errDigest != nil {
			return status.Failed(errDigest)
		}
		status.Digest = string(mavenDigest)
	}

	return status
}

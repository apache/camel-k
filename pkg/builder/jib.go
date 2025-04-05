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
	"fmt"
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
	"github.com/apache/camel-k/v2/pkg/util/registry"
)

type jibTask struct {
	c     client.Client
	build *v1.Build
	task  *v1.JibTask
}

var _ Task = &jibTask{}

func (t *jibTask) Do(ctx context.Context) v1.BuildStatus {
	status := initializeStatusFrom(t.build.Status, t.task.BaseImage)

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
		status.Image = status.BaseImage
		log.Infof("No new image to build, reusing existing image %s", status.Image)
		return *status
	}
	mavenDir := strings.ReplaceAll(contextDir, ContextDir, "maven")

	log.Debugf("Registry address: %s", t.task.Registry.Address)
	log.Debugf("Base image: %s", status.BaseImage)

	registryConfigDir := ""
	if t.task.Registry.Secret != "" {
		registryConfigDir, err = registry.MountSecretRegistryConfig(ctx, t.c, t.build.Namespace, "jib-secret-", t.task.Registry.Secret)
		os.Setenv(jib.JibRegistryConfigEnvVar, registryConfigDir)
		if err != nil {
			return status.Failed(err)
		}
	}

	mavenArgs := buildJibMavenArgs(mavenDir, t.task.Image, status.BaseImage, t.task.Registry.Insecure, t.task.Configuration.ImagePlatforms)
	mvnCmd := "./mvnw"
	if c, ok := os.LookupEnv("MAVEN_CMD"); ok {
		mvnCmd = c
	}
	cmd := exec.CommandContext(ctx, mvnCmd, mavenArgs...)
	cmd.Env = os.Environ()
	// Set Jib config directory to a writable directory within the image, Jib will create a default config file
	cmd.Env = append(cmd.Env, fmt.Sprintf("XDG_CONFIG_HOME=%s/jib", mavenDir))
	cmd.Dir = mavenDir

	myerror := util.RunAndLog(ctx, cmd, maven.LogHandler, maven.LogHandler)

	if myerror != nil {
		log.Errorf(myerror, "jib integration image containerization did not run successfully")
		_ = cleanRegistryConfig(registryConfigDir)
		return status.Failed(myerror)
	} else {
		log.Debug("jib integration image containerization did run successfully")
		status.Image = t.task.Image

		// retrieve image digest
		mavenDigest, errDigest := util.ReadFile(filepath.Join(mavenDir, jib.JibDigestFile))
		if errDigest != nil {
			_ = cleanRegistryConfig(registryConfigDir)
			return status.Failed(errDigest)
		}
		status.Digest = string(mavenDigest)
	}

	if registryConfigDir != "" {
		if err := cleanRegistryConfig(registryConfigDir); err != nil {
			return status.Failed(err)
		}
	}

	return *status
}

func cleanRegistryConfig(registryConfigDir string) error {
	if err := os.Unsetenv(jib.JibRegistryConfigEnvVar); err != nil {
		return err
	}
	if err := os.RemoveAll(registryConfigDir); err != nil {
		return err
	}
	return nil
}

// buildJibMavenArgs build the jib execution expected parameters.
func buildJibMavenArgs(mavenDir, image, baseImage string, insecureRegistry bool, imagePlatforms []string) []string {
	mavenArgs := make([]string, 0)
	mavenArgs = append(mavenArgs, jib.JibMavenGoal)
	mavenArgs = append(mavenArgs, "-Djib.disableUpdateChecks=true")
	mavenArgs = append(mavenArgs, "-P", "jib")
	mavenArgs = append(mavenArgs, jib.JibMavenToImageParam+image)
	mavenArgs = append(mavenArgs, jib.JibMavenFromImageParam+baseImage)
	mavenArgs = append(mavenArgs, jib.JibMavenBaseImageCache+mavenDir+"/jib")
	mavenArgs = append(mavenArgs, "-Djib.container.user=1000")

	if imagePlatforms != nil {
		platforms := strings.Join(imagePlatforms, ",")
		mavenArgs = append(mavenArgs, jib.JibMavenFromPlatforms+platforms)
	}

	if insecureRegistry {
		mavenArgs = append(mavenArgs, jib.JibMavenInsecureRegistries+"true")
	}

	return mavenArgs
}

// injectJibProfile is in charge to load the pom.xml and inject the profile required eventually
// for Jib publishing.
func injectJibProfile(ctx *builderContext) error {
	pomFile := filepath.Join(ctx.Path, "maven", "pom.xml")
	originalPom, err := util.ReadFile(pomFile)
	if err != nil {
		return err
	}
	updatedPom := injectProfilePom(jib.XMLJibProfile, string(originalPom))

	return util.WriteFileWithContent(pomFile, []byte(updatedPom))
}

// injectedProfilePom will inject the profile into the pom.
// NOTE: this may fail if the profiles section is commented as it would
// add the profile in a commented block, hence, won't be taken in account.
func injectProfilePom(profile, pom string) string {
	if strings.Contains(pom, "</profiles>") {
		return strings.ReplaceAll(
			pom,
			"</profiles>",
			fmt.Sprintf("\n\n<!-- Added by Camel K -->\n%s\n<!-- -->\n\n</profiles>", profile),
		)
	}

	return strings.ReplaceAll(
		pom,
		"</project>",
		fmt.Sprintf("\n\n<!-- Added by Camel K -->\n<profiles>\n%s\n</profiles>\n<!-- -->\n\n</project>", profile),
	)
}

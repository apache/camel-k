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
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.uber.org/multierr"

	spectrum "github.com/container-tools/spectrum/pkg/builder"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/log"
)

type spectrumTask struct {
	c     client.Client
	build *v1.Build
	task  *v1.SpectrumTask
}

var _ Task = &spectrumTask{}

func (t *spectrumTask) Do(ctx context.Context) v1.BuildStatus {
	status := v1.BuildStatus{}

	baseImage := t.build.Status.BaseImage
	if baseImage == "" {
		baseImage = t.task.BaseImage
	}
	status.BaseImage = baseImage
	rootImage := t.build.Status.RootImage
	if rootImage == "" {
		rootImage = t.task.BaseImage
	}
	status.RootImage = rootImage

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

	log.Infof("Running spectrum task in context directory: %s", contextDir)

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

	pullInsecure := t.task.Registry.Insecure // incremental build case

	log.Debugf("Registry address: %s", t.task.Registry.Address)
	log.Debugf("Base image: %s", baseImage)

	if !strings.HasPrefix(baseImage, t.task.Registry.Address) {
		if pullInsecure {
			log.Info("Assuming secure pull because the registry for the base image and the main registry are different")
			pullInsecure = false
		}
	}

	registryConfigDir := ""
	if t.task.Registry.Secret != "" {
		registryConfigDir, err = MountSecret(ctx, t.c, t.build.Namespace, t.task.Registry.Secret)
		if err != nil {
			return status.Failed(err)
		}
	}

	newStdR, newStdW, pipeErr := os.Pipe()
	defer util.CloseQuietly(newStdW)

	if pipeErr != nil {
		// In the unlikely case of an error, use stdout instead of aborting
		log.Errorf(pipeErr, "Unable to remap I/O. Spectrum messages will be displayed on the stdout")
		newStdW = os.Stdout
	}

	options := spectrum.Options{
		PullInsecure:  pullInsecure,
		PushInsecure:  t.task.Registry.Insecure,
		PullConfigDir: registryConfigDir,
		PushConfigDir: registryConfigDir,
		Base:          baseImage,
		Target:        t.task.Image,
		Stdout:        newStdW,
		Stderr:        newStdW,
		Recursive:     true,
	}

	if jobs := runtime.GOMAXPROCS(0); jobs > 1 {
		options.Jobs = jobs
	}

	go readSpectrumLogs(newStdR)
	digest, err := spectrum.Build(options, contextDir+":"+filepath.Join(DeploymentDir)) //nolint
	if err != nil {
		_ = os.RemoveAll(registryConfigDir)
		return status.Failed(err)
	}

	status.Image = t.task.Image
	status.Digest = digest

	if registryConfigDir != "" {
		if err := os.RemoveAll(registryConfigDir); err != nil {
			return status.Failed(err)
		}
	}

	return status
}

func readSpectrumLogs(newStdOut io.Reader) {
	scanner := bufio.NewScanner(newStdOut)

	for scanner.Scan() {
		line := scanner.Text()
		log.Infof(line)
	}
}

func MountSecret(ctx context.Context, c client.Client, namespace, name string) (string, error) {
	dir, err := os.MkdirTemp("", "spectrum-secret-")
	if err != nil {
		return "", err
	}

	secret, err := c.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if removeErr := os.RemoveAll(dir); removeErr != nil {
			err = multierr.Append(err, removeErr)
		}
		return "", err
	}

	for file, content := range secret.Data {
		if err := os.WriteFile(filepath.Join(dir, remap(file)), content, 0o600); err != nil {
			if removeErr := os.RemoveAll(dir); removeErr != nil {
				err = multierr.Append(err, removeErr)
			}
			return "", err
		}
	}
	return dir, nil
}

func remap(name string) string {
	if name == ".dockerconfigjson" {
		return "config.json"
	}
	return name
}

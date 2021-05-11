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
	spectrum "github.com/container-tools/spectrum/pkg/builder"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path"
	"path/filepath"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/log"
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
		status.BaseImage = baseImage
	}

	libraryPath := path.Join(t.task.ContextDir, DependenciesDir)
	_, err := os.Stat(libraryPath)
	if err != nil && os.IsNotExist(err) {
		// this can only indicate that there are no more libraries to add to the base image,
		// because transitive resolution is the same even if spec differs
		log.Infof("No new image to build, reusing existing image %s", baseImage)
		status.Image = baseImage
		return status
	} else if err != nil {
		return status.Failed(err)
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
		registryConfigDir, err = mountSecret(ctx, t.c, t.build.Namespace, t.task.Registry.Secret)
		if err != nil {
			return status.Failed(err)
		}
		defer os.RemoveAll(registryConfigDir)
	}

	newStdR, newStdW, pipeErr := os.Pipe()
	defer newStdW.Close()

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

	go readSpectrumLogs(newStdR)
	digest, err := spectrum.Build(options, libraryPath+":"+path.Join(DeploymentDir, DependenciesDir))
	
	if err != nil {
		return status.Failed(err)
	}

	status.Image = t.task.Image
	status.Digest = digest

	return status
}

func readSpectrumLogs(newStdOut *os.File) {
	scanner := bufio.NewScanner(newStdOut)

	for scanner.Scan() {
		line := scanner.Text()
		log.Infof(line)
	}
}

func mountSecret(ctx context.Context, c client.Client, namespace, name string) (string, error) {
	dir, err := ioutil.TempDir("", "spectrum-secret-")
	if err != nil {
		return "", err
	}

	secret, err := c.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}

	for file, content := range secret.Data {
		if err := ioutil.WriteFile(filepath.Join(dir, remap(file)), content, 0600); err != nil {
			os.RemoveAll(dir)
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

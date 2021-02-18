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

package spectrum

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/log"
	spectrum "github.com/container-tools/spectrum/pkg/builder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func publisher(ctx *builder.Context) error {
	libraryPath := path.Join(ctx.Path, "context", "dependencies")
	_, err := os.Stat(libraryPath)
	if err != nil && os.IsNotExist(err) {
		// this can only indicate that there are no more libraries to add to the base image,
		// because transitive resolution is the same even if spec differs
		log.Infof("No new image to build, reusing existing image %s", ctx.BaseImage)
		ctx.Image = ctx.BaseImage
		return nil
	} else if err != nil {
		return err
	}

	pl, err := platform.GetCurrent(ctx.C, ctx, ctx.Namespace)
	if err != nil {
		return err
	}

	target := "camel-k-" + ctx.Build.Meta.Name + ":" + ctx.Build.Meta.ResourceVersion
	repo := pl.Status.Build.Registry.Organization
	if repo != "" {
		target = fmt.Sprintf("%s/%s", repo, target)
	} else {
		target = fmt.Sprintf("%s/%s", ctx.Namespace, target)
	}
	registry := pl.Status.Build.Registry.Address
	if registry != "" {
		target = fmt.Sprintf("%s/%s", registry, target)
	}

	pullInsecure := pl.Status.Build.Registry.Insecure // incremental build case
	if ctx.BaseImage == pl.Status.Build.BaseImage {
		// Assuming the base image is always secure (we should add a flag)
		pullInsecure = false
	}

	registryConfigDir := ""
	if pl.Spec.Build.Registry.Secret != "" {
		registryConfigDir, err = mountSecret(ctx, pl.Spec.Build.Registry.Secret)
		if err != nil {
			return err
		}
		defer os.RemoveAll(registryConfigDir)
	}

	options := spectrum.Options{
		PullInsecure:  pullInsecure,
		PushInsecure:  pl.Status.Build.Registry.Insecure,
		PullConfigDir: registryConfigDir,
		PushConfigDir: registryConfigDir,
		Base:          ctx.BaseImage,
		Target:        target,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		Recursive:     true,
	}

	digest, err := spectrum.Build(options, libraryPath+":/deployments/dependencies")
	if err != nil {
		return err
	}

	ctx.Image = target
	ctx.Digest = digest
	return nil
}

func mountSecret(ctx *builder.Context, name string) (string, error) {
	dir, err := ioutil.TempDir("", "spectrum-secret-")
	if err != nil {
		return "", err
	}

	secret, err := ctx.CoreV1().Secrets(ctx.Namespace).Get(ctx.C, name, metav1.GetOptions{})
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

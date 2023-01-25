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

package catalog

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	platformutil "github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/log"
	spectrum "github.com/container-tools/spectrum/pkg/builder"
	corev1 "k8s.io/api/core/v1"
)

// NewInitializeAction returns a action that initializes the catalog configuration when not provided by the user.
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(catalog *v1.CamelCatalog) bool {
	return catalog.Status.Phase == v1.CamelCatalogPhaseNone
}

func (action *initializeAction) Handle(ctx context.Context, catalog *v1.CamelCatalog) (*v1.CamelCatalog, error) {
	platform, err := platformutil.GetOrFindLocal(ctx, action.client, catalog.Namespace)

	if err != nil {
		return catalog, err
	}

	if platform.Status.Phase != v1.IntegrationPlatformPhaseReady {
		// Wait the platform to be ready
		return catalog, nil
	}

	return initialize(platform, catalog)
}

func initialize(platform *v1.IntegrationPlatform, catalog *v1.CamelCatalog) (*v1.CamelCatalog, error) {
	target := catalog.DeepCopy()

	imageName := fmt.Sprintf("camel-k-runtime-%s-builder-%s", catalog.Spec.Runtime.Provider, strings.ToLower(catalog.Spec.Runtime.Version))

	// TODO: verify if the image already exists in the registry

	err := buildRuntimeBuilderImage(
		catalog.Spec.GetQuarkusToolingImage(),
		imageName,
		platform.Spec.Build.Registry.Address,
	)

	if err != nil {
		target.Status.Phase = v1.CamelCatalogPhaseError
		target.Status.SetErrorCondition(
			v1.CamelCatalogConditionReady,
			"Builder Image",
			err,
		)
	} else {
		target.Status.Phase = v1.CamelCatalogPhaseReady
		target.Status.SetCondition(
			v1.CamelCatalogConditionReady,
			corev1.ConditionTrue,
			"Builder Image",
			"Container image successfully built",
		)
		target.Status.Image = imageName
	}

	return target, nil
}

// This func will take care to dynamically build an image that will contain the tools required
// by the catalog build plus kamel binary and a maven wrapper required for the build.
func buildRuntimeBuilderImage(baseImage, targetImage, registryAddress string) error {
	if baseImage == "" {
		return fmt.Errorf("Missing base image, likely catalog is not compatible with this Camel K version")
	}
	log.Infof("Making up Camel K builder container %s", targetImage)

	newStdR, newStdW, pipeErr := os.Pipe()
	defer util.CloseQuietly(newStdW)

	if pipeErr != nil {
		// In the unlikely case of an error, use stdout instead of aborting
		log.Errorf(pipeErr, "Unable to remap I/O. Spectrum messages will be displayed on the stdout")
		newStdW = os.Stdout
	}

	// TODO provide proper configuration as in pkg/builder/spectrum.Do()
	remoteTarget := fmt.Sprintf("%s/%s", registryAddress, targetImage)

	options := spectrum.Options{
		PullInsecure:    true,
		PushInsecure:    true,
		PullConfigDir:   "",
		PushConfigDir:   "",
		Base:            baseImage,
		Target:          remoteTarget,
		Stdout:          newStdW,
		Stderr:          newStdW,
		Recursive:       true,
		ClearEntrypoint: true,
		RunAs:           "0",
	}

	if jobs := runtime.GOMAXPROCS(0); jobs > 1 {
		options.Jobs = jobs
	}

	go readSpectrumLogs(newStdR)
	_, err := spectrum.Build(options,
		"/usr/local/bin/kamel:/usr/local/bin/",
		"/usr/share/maven/mvnw/:/usr/share/maven/mvnw/",
		"/tmp/artifacts/m2/org/apache/camel/:/tmp/artifacts/m2/org/apache/camel/") //nolint
	if err != nil {
		return err
	}

	return nil
}

func readSpectrumLogs(newStdOut io.Reader) {
	scanner := bufio.NewScanner(newStdOut)

	for scanner.Scan() {
		line := scanner.Text()
		log.Infof(line)
	}
}

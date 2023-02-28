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
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/client"
	platformutil "github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util"
	spectrum "github.com/container-tools/spectrum/pkg/builder"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
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

	// Make basic options for building image in the registry
	options, err := makeSpectrumOptions(ctx, action.client, platform.Namespace, platform.Status.Build.Registry)
	if err != nil {
		return catalog, err
	}

	return initialize(options, platform.Spec.Build.Registry.Address, catalog)
}

func initialize(options spectrum.Options, registryAddress string, catalog *v1.CamelCatalog) (*v1.CamelCatalog, error) {
	target := catalog.DeepCopy()
	imageName := fmt.Sprintf(
		"%s/camel-k-runtime-%s-builder:%s",
		registryAddress,
		catalog.Spec.Runtime.Provider,
		strings.ToLower(catalog.Spec.Runtime.Version),
	)

	newStdR, newStdW, pipeErr := os.Pipe()
	defer util.CloseQuietly(newStdW)

	if pipeErr != nil {
		// In the unlikely case of an error, use stdout instead of aborting
		Log.Errorf(pipeErr, "Unable to remap I/O. Spectrum messages will be displayed on the stdout")
		newStdW = os.Stdout
	}
	go readSpectrumLogs(newStdR)

	// We use the future target image as a base just for the sake of pulling and verify it exists
	options.Base = imageName
	options.Stderr = newStdW
	options.Stdout = newStdW

	if !imageSnapshot(options) && imageExists(options) {
		target.Status.Phase = v1.CamelCatalogPhaseReady
		target.Status.SetCondition(
			v1.CamelCatalogConditionReady,
			corev1.ConditionTrue,
			"Builder Image",
			"Container image exists on registry",
		)
		target.Status.Image = imageName

		return target, nil
	}

	// Now we properly set the base and the target image
	options.Base = catalog.Spec.GetQuarkusToolingImage()
	options.Target = imageName

	err := buildRuntimeBuilderImage(options)

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

func imageExists(options spectrum.Options) bool {
	Log.Infof("Checking if Camel K builder container %s already exists...", options.Base)
	ctrImg, err := spectrum.Pull(options)
	if ctrImg != nil && err == nil {
		var hash gcrv1.Hash
		if hash, err = ctrImg.Digest(); err != nil {
			Log.Errorf(err, "Cannot calculate digest")
			return false
		}
		Log.Infof("Found Camel K builder container with digest %s", hash.String())
		return true
	}

	Log.Infof("Couldn't pull image due to %s", err.Error())
	return false
}

func imageSnapshot(options spectrum.Options) bool {
	return strings.HasSuffix(options.Base, "snapshot")
}

// This func will take care to dynamically build an image that will contain the tools required
// by the catalog build plus kamel binary and a maven wrapper required for the build.
func buildRuntimeBuilderImage(options spectrum.Options) error {
	if options.Base == "" {
		return fmt.Errorf("missing base image, likely catalog is not compatible with this Camel K version")
	}
	Log.Infof("Making up Camel K builder container %s", options.Target)

	if jobs := runtime.GOMAXPROCS(0); jobs > 1 {
		options.Jobs = jobs
	}

	// TODO support also S2I
	_, err := spectrum.Build(options,
		"/usr/local/bin/kamel:/usr/local/bin/",
		"/usr/share/maven/mvnw/:/usr/share/maven/mvnw/")
	if err != nil {
		return err
	}

	return nil
}

func readSpectrumLogs(newStdOut io.Reader) {
	scanner := bufio.NewScanner(newStdOut)

	for scanner.Scan() {
		line := scanner.Text()
		Log.Infof(line)
	}
}

func makeSpectrumOptions(ctx context.Context, c client.Client, platformNamespace string, registry v1.RegistrySpec) (spectrum.Options, error) {
	options := spectrum.Options{}
	var err error
	registryConfigDir := ""
	if registry.Secret != "" {
		registryConfigDir, err = builder.MountSecret(ctx, c, platformNamespace, registry.Secret)
		if err != nil {
			return options, err
		}
	}
	options.PullInsecure = registry.Insecure
	options.PushInsecure = registry.Insecure
	options.PullConfigDir = registryConfigDir
	options.PushConfigDir = registryConfigDir
	options.Recursive = true

	return options, nil
}

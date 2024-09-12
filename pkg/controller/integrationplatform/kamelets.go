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

package integrationplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/install"
	"github.com/apache/camel-k/v2/pkg/platform"
	"knative.dev/pkg/ptr"

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/maven"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	kameletDirEnv          = "KAMELET_CATALOG_DIR"
	defaultKameletDir      = "/tmp/kamelets/"
	kamelVersionAnnotation = "camel.apache.org/version"
)

// installKameletCatalog installs the version Apache Kamelet Catalog into the specified namespace. It returns the number of Kamelets installed and errored
// if successful.
func installKameletCatalog(ctx context.Context, c client.Client, platform *v1.IntegrationPlatform, version string) (int, int, error) {
	// Prepare proper privileges for Kamelets installed globally
	if err := prepareKameletsPermissions(ctx, c, platform.Namespace); err != nil {
		return -1, -1, err
	}
	// Prepare directory to contains kamelets
	kameletDir := prepareKameletDirectory()
	// Download Kamelet dependency
	if err := downloadKameletDependency(ctx, version, kameletDir); err != nil {
		return -1, -1, err
	}
	// Extract Kamelets files
	if err := extractKameletsFromDependency(ctx, version, kameletDir); err != nil {
		return -1, -1, err
	}

	// Store Kamelets as Kubernetes resources
	return applyKamelets(ctx, c, platform, kameletDir)
}

func prepareKameletsPermissions(ctx context.Context, c client.Client, installingNamespace string) error {
	watchOperatorNamespace := platform.GetOperatorWatchNamespace()
	operatorNamespace := platform.GetOperatorNamespace()
	if watchOperatorNamespace == "" && operatorNamespace == installingNamespace {
		// Kamelets installed into the global operator namespace
		// They need to be visible publicly
		if err := kameletViewerRole(ctx, c, installingNamespace); err != nil {
			return err
		}
	}

	return nil
}

func prepareKameletDirectory() string {
	kameletDir := os.Getenv(kameletDirEnv)
	if kameletDir == "" {
		kameletDir = defaultKameletDir
	}

	return kameletDir
}

func downloadKameletDependency(ctx context.Context, version, kameletsDir string) error {
	// TODO: we may want to add the maven settings coming from the platform
	// in order to cover any user security setting in place
	p := maven.NewProjectWithGAV("org.apache.camel.k.kamelets", "kamelets-catalog", defaults.Version)
	mc := maven.NewContext(kameletsDir)
	mc.AddArgument("-q")
	mc.AddArgument("dependency:copy")
	mc.AddArgument(fmt.Sprintf("-Dartifact=org.apache.camel.kamelets:camel-kamelets:%s:jar", version))
	mc.AddArgument("-Dmdep.useBaseVersion=true")
	mc.AddArgument(fmt.Sprintf("-DoutputDirectory=%s", kameletsDir))

	return p.Command(mc).Do(ctx)
}

func extractKameletsFromDependency(ctx context.Context, version, kameletsDir string) error {
	args := strings.Split(
		fmt.Sprintf("-xf camel-kamelets-%s.jar kamelets/", version), " ")
	cmd := exec.CommandContext(ctx, "jar", args...)
	cmd.Dir = kameletsDir
	return util.RunAndLog(ctx, cmd, maven.LogHandler, maven.LogHandler)
}

func applyKamelets(ctx context.Context, c client.Client, platform *v1.IntegrationPlatform, kameletDir string) (int, int, error) {
	appliedKam := 0
	erroredKam := 0
	applier := c.ServerOrClientSideApplier()
	dir := path.Join(kameletDir, "kamelets")

	err := filepath.WalkDir(dir, func(p string, f fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !(strings.HasSuffix(f.Name(), ".yaml") || strings.HasSuffix(f.Name(), ".yml")) {
			return nil
		}
		kamelet, err := loadKamelet(filepath.Join(dir, f.Name()), platform)
		// We cannot return if an error happen, otherwise the creation of the IntegrationPlatform would result
		// in a failure. We better report in conditions.
		if err != nil {
			erroredKam++
			log.Errorf(err, "Error occurred whilst loading a bundled kamelet named %s", f.Name())
			return nil
		}
		err = applier.Apply(ctx, kamelet)
		if err != nil {
			erroredKam++
			log.Error(err, "Error occurred whilst applying a bundled kamelet named %s", kamelet.GetName())
			return nil
		}
		appliedKam++

		return nil
	})
	if err != nil {
		return appliedKam, erroredKam, err
	}

	return appliedKam, erroredKam, nil
}

func loadKamelet(path string, platform *v1.IntegrationPlatform) (*v1.Kamelet, error) {
	yamlContent, err := util.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Kamelet spec contains raw object spec, for which we need to convert to json
	// for a proper serde
	jsonContent, err := k8syaml.ToJSON(yamlContent)
	if err != nil {
		return nil, err
	}
	var kamelet *v1.Kamelet
	if err = json.Unmarshal(jsonContent, &kamelet); err != nil {
		return nil, err
	}
	kamelet.SetNamespace(platform.Namespace)
	annotations := kamelet.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[kamelVersionAnnotation] = defaults.Version
	kamelet.SetAnnotations(annotations)
	labels := kamelet.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[v1.KameletBundledLabel] = "true"
	labels[v1.KameletReadOnlyLabel] = "true"

	// The Kamelet will be owned by the IntegrationPlatform
	references := []metav1.OwnerReference{
		{
			APIVersion:         platform.APIVersion,
			Kind:               platform.Kind,
			Name:               platform.Name,
			UID:                platform.UID,
			Controller:         ptr.Bool(true),
			BlockOwnerDeletion: ptr.Bool(true),
		},
	}
	kamelet.SetOwnerReferences(references)

	return kamelet, nil
}

// kameletViewerRole installs the role that allows any user ro access kamelets in the global namespace.
func kameletViewerRole(ctx context.Context, c client.Client, namespace string) error {
	if err := install.Resource(ctx, c, namespace, true, install.IdentityResourceCustomizer,
		"/resources/viewer/user-global-kamelet-viewer-role.yaml"); err != nil {
		return err
	}
	return install.Resource(ctx, c, namespace, true, install.IdentityResourceCustomizer,
		"/resources/viewer/user-global-kamelet-viewer-role-binding.yaml")
}

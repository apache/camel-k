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
	"knative.dev/pkg/ptr"

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/jvm"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
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

// installKameletCatalog installs the version Apache Kamelet Catalog into the specified namespace.
// It returns the number of Kamelets installed and errored if successful.
func installKameletCatalog(ctx context.Context, c client.Client, platform *v1.IntegrationPlatform, version string) (int, int, error) {
	// Prepare directory to contains kamelets
	kameletDir, err := prepareKameletDirectory()
	if err != nil {
		return -1, -1, err
	}
	// Download Kamelet dependency
	if err := downloadKameletDependency(ctx, c, platform, version, kameletDir); err != nil {
		return -1, -1, err
	}
	// Extract Kamelets files
	if err := extractKameletsFromDependency(ctx, version, kameletDir); err != nil {
		return -1, -1, err
	}
	// Store Kamelets as Kubernetes resources
	return applyKamelets(ctx, c, platform, kameletDir)
}

func prepareKameletDirectory() (string, error) {
	kameletDir := os.Getenv(kameletDirEnv)
	if kameletDir == "" {
		kameletDir = defaultKameletDir
	}
	// If the directory exists, it is likely a leftover from any previous Kamelet
	// catalog installation. We should remove to be able to proceed
	if err := os.RemoveAll(kameletDir); err != nil {
		return kameletDirEnv, err
	}
	err := os.MkdirAll(kameletDir, os.ModePerm)

	return kameletDir, err
}

func downloadKameletDependency(ctx context.Context, c client.Client, platform *v1.IntegrationPlatform, version, kameletsDir string) error {
	p := maven.NewProjectWithGAV("org.apache.camel.k.kamelets", "kamelets-catalog", defaults.Version)
	mc := maven.NewContext(kameletsDir)
	mc.SkipMavenConfigGeneration = true
	mc.LocalRepository = platform.Status.Build.Maven.LocalRepository
	mc.AdditionalArguments = platform.Status.Build.Maven.CLIOptions
	mc.AddArgument("-q")
	mc.AddArgument("dependency:copy")
	mc.AddArgument(fmt.Sprintf("-Dartifact=org.apache.camel.kamelets:camel-kamelets:%s:jar", version))
	mc.AddArgument("-Dmdep.useBaseVersion=true")
	// TODO: this one should be already managed during the command execution
	// This workaround is fixing temporarily the problem
	mc.AddArgument("-Dmaven.repo.local=" + mc.LocalRepository)
	mc.AddArgument("-DoutputDirectory=" + kameletsDir)

	if settings, err := kubernetes.ResolveValueSource(ctx, c, platform.Namespace, &platform.Status.Build.Maven.Settings); err != nil {
		return err
	} else if settings != "" {
		mc.UserSettings = []byte(settings)
	}

	settings, err := maven.NewSettings(maven.DefaultRepositories, maven.ProxyFromEnvironment)
	if err != nil {
		return err
	}
	data, err := settings.MarshalBytes()
	if err != nil {
		return err
	}
	mc.GlobalSettings = data
	secrets := platform.Status.Build.Maven.CASecrets

	if secrets != nil {
		certsData, err := kubernetes.GetSecretsRefData(ctx, c, platform.Namespace, secrets)
		if err != nil {
			return err
		}
		trustStoreName := "trust.jks"
		trustStorePass := jvm.NewKeystorePassword()
		err = jvm.GenerateKeystore(ctx, kameletsDir, trustStoreName, trustStorePass, certsData)
		if err != nil {
			return err
		}
		mc.ExtraMavenOpts = append(mc.ExtraMavenOpts,
			"-Djavax.net.ssl.trustStore="+trustStoreName,
			"-Djavax.net.ssl.trustStorePassword="+trustStorePass,
		)
		// TODO: this one should be already managed during the command execution
		// This workaround is fixing temporarily the problem
		mc.AddArgument("-Djavax.net.ssl.trustStore=" + trustStoreName)
		mc.AddArgument("-Djavax.net.ssl.trustStorePassword=" + trustStorePass)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, platform.Status.Build.GetTimeout().Duration)
	defer cancel()

	if err := p.Command(mc).DoSettings(ctx); err != nil {
		return err
	}
	if err := p.Command(mc).DoPom(ctx); err != nil {
		return err
	}

	return p.Command(mc).Do(timeoutCtx)
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
		if !strings.HasSuffix(f.Name(), ".yaml") && !strings.HasSuffix(f.Name(), ".yml") {
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
			log.Errorf(err, "Error occurred whilst applying a bundled kamelet named %s", kamelet.GetName())

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

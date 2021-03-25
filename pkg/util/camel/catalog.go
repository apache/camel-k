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

package camel

import (
	"context"
	"io/ioutil"
	"os"
	"path"

	yaml2 "gopkg.in/yaml.v2"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/resources"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/jvm"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/maven"
)

func DefaultCatalog() (*RuntimeCatalog, error) {
	return QuarkusCatalog()
}

func QuarkusCatalog() (*RuntimeCatalog, error) {
	return catalogForRuntimeProvider(v1.RuntimeProviderQuarkus)
}

func catalogForRuntimeProvider(provider v1.RuntimeProvider) (*RuntimeCatalog, error) {
	catalogs := make([]v1.CamelCatalog, 0)

	for _, name := range resources.ResourcesWithPrefix("/camel-catalog-") {
		var c v1.CamelCatalog
		if err := yaml2.Unmarshal(resources.Resource(name), &c); err != nil {
			return nil, err
		}

		catalogs = append(catalogs, c)
	}

	return findBestMatch(catalogs, v1.RuntimeSpec{
		Version:  defaults.DefaultRuntimeVersion,
		Provider: provider,
		Metadata: make(map[string]string),
	})
}

func GenerateCatalog(
	ctx context.Context,
	client k8sclient.Reader,
	namespace string,
	mvn v1.MavenSpec,
	runtime v1.RuntimeSpec,
	providerDependencies []maven.Dependency) (*RuntimeCatalog, error) {

	settings, err := kubernetes.ResolveValueSource(ctx, client, namespace, &mvn.Settings)
	if err != nil {
		return nil, err
	}

	var caCert []byte
	if mvn.CaCert != nil {
		caCert, err = kubernetes.GetSecretRefData(ctx, client, namespace, mvn.CaCert)
		if err != nil {
			return nil, err
		}
	}

	return GenerateCatalogCommon(settings, caCert, mvn, runtime, providerDependencies)
}

func GenerateCatalogCommon(
	settings string,
	caCert []byte,
	mvn v1.MavenSpec,
	runtime v1.RuntimeSpec,
	providerDependencies []maven.Dependency) (*RuntimeCatalog, error) {

	root := os.TempDir()
	tmpDir, err := ioutil.TempDir(root, "camel-catalog")
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return nil, err
	}

	project := generateMavenProject(runtime.Version, providerDependencies)

	mc := maven.NewContext(tmpDir, project)
	mc.LocalRepository = mvn.LocalRepository
	mc.Timeout = mvn.GetTimeout().Duration
	mc.AddSystemProperty("catalog.path", tmpDir)
	mc.AddSystemProperty("catalog.file", "catalog.yaml")
	mc.AddSystemProperty("catalog.runtime", string(runtime.Provider))

	mc.SettingsContent = nil
	if settings != "" {
		mc.SettingsContent = []byte(settings)
	}

	if caCert != nil {
		trustStoreName := "trust.jks"
		err := jvm.GenerateJavaKeystore(context.Background(), tmpDir, trustStoreName, caCert)
		if err != nil {
			return nil, err
		}
		mc.ExtraMavenOpts = append(mc.ExtraMavenOpts, "-Djavax.net.ssl.trustStore="+trustStoreName)
	}

	err = maven.Run(mc)
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadFile(path.Join(tmpDir, "catalog.yaml"))
	if err != nil {
		return nil, err
	}

	catalog := v1.CamelCatalog{}
	if err := yaml2.Unmarshal(content, &catalog); err != nil {
		return nil, err
	}

	return NewRuntimeCatalog(catalog.Spec), nil
}

func generateMavenProject(runtimeVersion string, providerDependencies []maven.Dependency) maven.Project {
	p := maven.NewProjectWithGAV("org.apache.camel.k.integration", "camel-k-catalog-generator", defaults.Version)

	plugin := maven.Plugin{
		GroupID:    "org.apache.camel.k",
		ArtifactID: "camel-k-maven-plugin",
		Version:    runtimeVersion,
		Executions: []maven.Execution{
			{
				ID: "generate-catalog",
				Goals: []string{
					"generate-catalog",
				},
			},
		},
	}

	plugin.Dependencies = append(plugin.Dependencies, providerDependencies...)

	p.Build = &maven.Build{
		DefaultGoal: "generate-resources",
		Plugins:     []maven.Plugin{plugin},
	}

	return p
}

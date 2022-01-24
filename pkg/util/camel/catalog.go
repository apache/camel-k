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
	"path"

	yaml2 "gopkg.in/yaml.v2"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/resources"
	"github.com/apache/camel-k/pkg/util"
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

	names, err := resources.WithPrefix("/camel-catalog-")
	if err != nil {
		return nil, err
	}

	for _, name := range names {

		content, err := resources.Resource(name)
		if err != nil {
			return nil, err
		}

		var c v1.CamelCatalog
		if err := yaml2.Unmarshal(content, &c); err != nil {
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
	client ctrl.Reader,
	namespace string,
	mvn v1.MavenSpec,
	runtime v1.RuntimeSpec,
	providerDependencies []maven.Dependency) (*RuntimeCatalog, error) {

	userSettings, err := kubernetes.ResolveValueSource(ctx, client, namespace, &mvn.Settings)
	if err != nil {
		return nil, err
	}
	settings, err := maven.NewSettings(maven.DefaultRepositories, maven.ProxyFromEnvironment)
	if err != nil {
		return nil, err
	}
	globalSettings, err := settings.MarshalBytes()
	if err != nil {
		return nil, err
	}

	var caCert []byte
	if mvn.CASecret != nil {
		caCert, err = kubernetes.GetSecretRefData(ctx, client, namespace, mvn.CASecret)
		if err != nil {
			return nil, err
		}
	}

	return GenerateCatalogCommon(ctx, globalSettings, []byte(userSettings), caCert, mvn, runtime, providerDependencies)
}

func GenerateCatalogCommon(
	ctx context.Context,
	globalSettings []byte,
	userSettings []byte,
	caCert []byte,
	mvn v1.MavenSpec,
	runtime v1.RuntimeSpec,
	providerDependencies []maven.Dependency) (*RuntimeCatalog, error) {

	catalog := v1.CamelCatalog{}

	err := util.WithTempDir("camel-catalog", func(tmpDir string) error {
		project := generateMavenProject(runtime.Version, providerDependencies)

		mc := maven.NewContext(tmpDir)
		mc.LocalRepository = mvn.LocalRepository
		mc.AdditionalArguments = mvn.CLIOptions
		mc.AddSystemProperty("catalog.path", tmpDir)
		mc.AddSystemProperty("catalog.file", "catalog.yaml")
		mc.AddSystemProperty("catalog.runtime", string(runtime.Provider))

		if len(globalSettings) > 0 {
			mc.GlobalSettings = globalSettings
		}
		if len(userSettings) > 0 {
			mc.UserSettings = userSettings
		}

		if caCert != nil {
			trustStoreName := "trust.jks"
			trustStorePass := jvm.NewKeystorePassword()
			if err := jvm.GenerateKeystore(ctx, tmpDir, trustStoreName, trustStorePass, caCert); err != nil {
				return err
			}
			mc.ExtraMavenOpts = append(mc.ExtraMavenOpts,
				"-Djavax.net.ssl.trustStore="+trustStoreName,
				"-Djavax.net.ssl.trustStorePassword="+trustStorePass,
			)
		}

		if err := project.Command(mc).Do(ctx); err != nil {
			return err
		}

		content, err := util.ReadFile(path.Join(tmpDir, "catalog.yaml"))
		if err != nil {
			return err
		}

		return yaml2.Unmarshal(content, &catalog)
	})

	return NewRuntimeCatalog(catalog.Spec), err
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

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

package trait

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder/runtime"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/envvar"
	"github.com/apache/camel-k/pkg/util/maven"
)

// The Quarkus trait activates the Quarkus runtime.
//
// It's disabled by default.
//
// +camel-k:trait=quarkus
type quarkusTrait struct {
	BaseTrait `property:",squash"`
	// The Quarkus version to use for the integration
	QuarkusVersion string `property:"quarkus-version"`
	// The Camel-Quarkus version to use for the integration
	CamelQuarkusVersion string `property:"camel-quarkus-version"`
}

func newQuarkusTrait() *quarkusTrait {
	return &quarkusTrait{
		BaseTrait: newBaseTrait("quarkus"),
	}
}

func (t *quarkusTrait) isEnabled() bool {
	return t.Enabled != nil && *t.Enabled
}

func (t *quarkusTrait) Configure(e *Environment) (bool, error) {
	return t.isEnabled(), nil
}

func (t *quarkusTrait) Apply(e *Environment) error {
	return nil
}

// InfluencesKit overrides base class method
func (t *quarkusTrait) InfluencesKit() bool {
	return true
}

func (t *quarkusTrait) loadOrCreateCatalog(e *Environment, camelVersion string, runtimeVersion string) error {
	ns := e.DetermineNamespace()
	if ns == "" {
		return errors.New("unable to determine namespace")
	}

	camelQuarkusVersion := t.determineCamelQuarkusVersion(e)
	quarkusVersion := t.determineQuarkusVersion(e)

	catalog, err := camel.LoadCatalog(e.C, e.Client, ns, camelVersion, runtimeVersion, v1alpha1.QuarkusRuntimeProvider{
		CamelQuarkusVersion: camelQuarkusVersion,
		QuarkusVersion:      quarkusVersion,
	})
	if err != nil {
		return err
	}

	if catalog == nil {
		// if the catalog is not found in the cluster, try to create it if
		// the required versions (camel, runtime and provider) are not expressed as
		// semver constraints
		if exactVersionRegexp.MatchString(camelVersion) && exactVersionRegexp.MatchString(runtimeVersion) &&
			exactVersionRegexp.MatchString(camelQuarkusVersion) && exactVersionRegexp.MatchString(quarkusVersion) {
			catalog, err = camel.GenerateCatalogWithProvider(e.C, e.Client, ns, e.Platform.Status.FullConfig.Build.Maven, camelVersion, runtimeVersion,
				"quarkus",
				[]maven.Dependency{
					{
						GroupID:    "org.apache.camel.quarkus",
						ArtifactID: "camel-catalog-quarkus",
						Version:    camelQuarkusVersion,
					},
					// This is required to retrieve the Quarkus dependency version
					{
						GroupID:    "org.apache.camel.quarkus",
						ArtifactID: "camel-quarkus-core",
						Version:    camelQuarkusVersion,
					},
				})
			if err != nil {
				return err
			}

			// sanitize catalog name
			catalogName := "camel-catalog-quarkus-" + strings.ToLower(camelVersion+"-"+runtimeVersion)

			cx := v1alpha1.NewCamelCatalogWithSpecs(ns, catalogName, catalog.CamelCatalogSpec)
			cx.Labels = make(map[string]string)
			cx.Labels["app"] = "camel-k"
			cx.Labels["camel.apache.org/catalog.version"] = camelVersion
			cx.Labels["camel.apache.org/catalog.loader.version"] = camelVersion
			cx.Labels["camel.apache.org/runtime.version"] = runtimeVersion
			cx.Labels["camel.apache.org/runtime.provider"] = "quarkus"
			cx.Labels["camel.apache.org/catalog.generated"] = True

			err = e.Client.Create(e.C, &cx)
			if err != nil && !k8serrors.IsAlreadyExists(err) {
				return err
			}
		}
	}

	if catalog == nil {
		return fmt.Errorf("unable to find catalog matching version requirement: camel=%s, runtime=%s, camel-quarkus=%s, quarkus=%s",
			camelVersion, runtimeVersion, camelQuarkusVersion, quarkusVersion)
	}

	e.CamelCatalog = catalog

	return nil
}

func (t *quarkusTrait) addBuildSteps(e *Environment) {
	e.Steps = append(e.Steps, runtime.QuarkusSteps...)
}

func (t *quarkusTrait) addClasspath(e *Environment) {
	// No-op as we rely on the Quarkus runner
}

func (t *quarkusTrait) addRuntimeDependencies(e *Environment) error {
	dependencies := &e.Integration.Status.Dependencies

	for _, s := range e.Integration.Sources() {
		meta := metadata.Extract(e.CamelCatalog, s)

		switch language := s.InferLanguage(); language {
		case v1alpha1.LanguageYaml:
			addRuntimeDependency("camel-k-quarkus-loader-yaml", dependencies)
		case v1alpha1.LanguageXML:
			addRuntimeDependency("camel-k-quarkus-loader-xml", dependencies)
		case v1alpha1.LanguageJavaScript:
			addRuntimeDependency("camel-k-quarkus-loader-js", dependencies)
		default:
			return fmt.Errorf("unsupported language for Quarkus runtime: %s", language)
		}

		if strings.HasPrefix(s.Loader, "knative-source") {
			addRuntimeDependency("camel-k-quarkus-loader-knative", dependencies)
			addRuntimeDependency("camel-k-quarkus-knative", dependencies)
		}

		addRuntimeDependency("camel-k-runtime-quarkus", dependencies)

		for _, d := range meta.Dependencies.List() {
			util.StringSliceUniqueAdd(dependencies, d)
		}
	}
	return nil
}

func (t *quarkusTrait) addContainerEnvironment(e *Environment) {
	envvar.SetVal(&e.EnvVars, envVarAppJAR, "camel-k-integration-"+defaults.Version+"-runner.jar")
}

func addRuntimeDependency(dependency string, dependencies *[]string) {
	util.StringSliceUniqueAdd(dependencies, fmt.Sprintf("mvn:org.apache.camel.k/%s", dependency))
}

func (t *quarkusTrait) determineQuarkusVersion(e *Environment) string {
	if t.QuarkusVersion != "" {
		return t.QuarkusVersion
	}
	if e.Integration != nil && e.Integration.Status.RuntimeProvider != nil && e.Integration.Status.RuntimeProvider.Quarkus != nil &&
		e.Integration.Status.RuntimeProvider.Quarkus.QuarkusVersion != "" {
		return e.Integration.Status.RuntimeProvider.Quarkus.QuarkusVersion
	}
	if e.IntegrationKit != nil && e.IntegrationKit.Status.RuntimeProvider != nil && e.IntegrationKit.Status.RuntimeProvider.Quarkus != nil &&
		e.IntegrationKit.Status.RuntimeProvider.Quarkus.QuarkusVersion != "" {
		return e.IntegrationKit.Status.RuntimeProvider.Quarkus.QuarkusVersion
	}
	if e.Platform.Status.FullConfig.Build.RuntimeProvider != nil && e.Platform.Status.FullConfig.Build.RuntimeProvider.Quarkus != nil &&
		e.Platform.Status.FullConfig.Build.RuntimeProvider.Quarkus.QuarkusVersion != "" {
		return e.Platform.Status.FullConfig.Build.RuntimeProvider.Quarkus.QuarkusVersion
	}
	return defaults.QuarkusVersionConstraint
}

func (t *quarkusTrait) determineCamelQuarkusVersion(e *Environment) string {
	if t.CamelQuarkusVersion != "" {
		return t.CamelQuarkusVersion
	}
	if e.Integration != nil && e.Integration.Status.RuntimeProvider != nil && e.Integration.Status.RuntimeProvider.Quarkus != nil &&
		e.Integration.Status.RuntimeProvider.Quarkus.CamelQuarkusVersion != "" {
		return e.Integration.Status.RuntimeProvider.Quarkus.CamelQuarkusVersion
	}
	if e.IntegrationKit != nil && e.IntegrationKit.Status.RuntimeProvider != nil && e.IntegrationKit.Status.RuntimeProvider.Quarkus != nil &&
		e.IntegrationKit.Status.RuntimeProvider.Quarkus.CamelQuarkusVersion != "" {
		return e.IntegrationKit.Status.RuntimeProvider.Quarkus.CamelQuarkusVersion
	}
	if e.Platform.Status.FullConfig.Build.RuntimeProvider != nil && e.Platform.Status.FullConfig.Build.RuntimeProvider.Quarkus != nil &&
		e.Platform.Status.FullConfig.Build.RuntimeProvider.Quarkus.CamelQuarkusVersion != "" {
		return e.Platform.Status.FullConfig.Build.RuntimeProvider.Quarkus.CamelQuarkusVersion
	}
	return defaults.CamelQuarkusVersionConstraint
}

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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder/runtime"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/envvar"
)

type quarkusTrait struct {
	BaseTrait `property:",squash"`
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

func (t *quarkusTrait) loadOrCreateCatalog(e *Environment, camelVersion string, runtimeVersion string) error {
	ns := e.DetermineNamespace()
	if ns == "" {
		return errors.New("unable to determine namespace")
	}

	c, err := camel.LoadCatalog(e.C, e.Client, ns, camelVersion, runtimeVersion, v1alpha1.QuarkusRuntimeProvider{
		// FIXME
		CamelQuarkusVersion: "0.2.0",
		QuarkusVersion:      "0.21.2",
	})
	if err != nil {
		return err
	}

	e.CamelCatalog = c

	// TODO: generate a catalog if nil

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

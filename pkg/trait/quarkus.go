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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder/runtime"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
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

func (t *quarkusTrait) addBuildSteps(e *Environment) {
	e.Steps = append(e.Steps, runtime.QuarkusSteps...)
}

func (t *quarkusTrait) addRuntimeDependencies(e *Environment, dependencies *[]string) error {
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

		// main required by default
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

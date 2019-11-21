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
	"sort"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
)

// The Dependencies trait is internally used to automatically add runtime dependencies based on the
// integration that the user wants to run.
//
// +camel-k:trait=dependencies
type dependenciesTrait struct {
	BaseTrait `property:",squash"`
}

func newDependenciesTrait() *dependenciesTrait {
	return &dependenciesTrait{
		BaseTrait: newBaseTrait("dependencies"),
	}
}

func (t *dependenciesTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return e.IntegrationInPhase(v1alpha1.IntegrationPhaseInitialization), nil
}

func (t *dependenciesTrait) Apply(e *Environment) error {
	if e.Integration.Spec.Dependencies != nil {
		dependencies := make([]string, 0)
		for _, dep := range e.Integration.Spec.Dependencies {
			util.StringSliceUniqueAdd(&dependencies, dep)
			e.Integration.Status.Dependencies = dependencies
		}
	} else {
		e.Integration.Status.Dependencies = make([]string, 0)
	}

	quarkus := e.Catalog.GetTrait("quarkus").(*quarkusTrait)
	if quarkus.isEnabled() {
		err := quarkus.addRuntimeDependencies(e)
		if err != nil {
			return err
		}
	} else {
		addDefaultRuntimeDependencies(e)
	}

	// sort the dependencies to get always the same list if they don't change
	sort.Strings(e.Integration.Status.Dependencies)
	return nil
}

// IsPlatformTrait overrides base class method
func (t *dependenciesTrait) IsPlatformTrait() bool {
	return true
}

func addDefaultRuntimeDependencies(e *Environment) {
	dependencies := &e.Integration.Status.Dependencies

	for _, s := range e.Integration.Sources() {
		meta := metadata.Extract(e.CamelCatalog, s)

		switch s.InferLanguage() {
		case v1alpha1.LanguageGroovy:
			util.StringSliceUniqueAdd(dependencies, "mvn:org.apache.camel.k/camel-k-loader-groovy")
		case v1alpha1.LanguageKotlin:
			util.StringSliceUniqueAdd(dependencies, "mvn:org.apache.camel.k/camel-k-loader-kotlin")
		case v1alpha1.LanguageYaml:
			util.StringSliceUniqueAdd(dependencies, "mvn:org.apache.camel.k/camel-k-loader-yaml")
		case v1alpha1.LanguageXML:
			util.StringSliceUniqueAdd(dependencies, "mvn:org.apache.camel.k/camel-k-loader-xml")
		case v1alpha1.LanguageJavaScript:
			util.StringSliceUniqueAdd(dependencies, "mvn:org.apache.camel.k/camel-k-loader-js")
		case v1alpha1.LanguageJavaSource:
			util.StringSliceUniqueAdd(dependencies, "mvn:org.apache.camel.k/camel-k-loader-java")
		}

		if strings.HasPrefix(s.Loader, "knative-source") {
			util.StringSliceUniqueAdd(dependencies, "mvn:org.apache.camel.k/camel-k-loader-knative")
			util.StringSliceUniqueAdd(dependencies, "mvn:org.apache.camel.k/camel-k-runtime-knative")
		}

		// main required by default
		util.StringSliceUniqueAdd(dependencies, "mvn:org.apache.camel.k/camel-k-runtime-main")

		for _, d := range meta.Dependencies.List() {
			util.StringSliceUniqueAdd(dependencies, d)
		}
	}
}

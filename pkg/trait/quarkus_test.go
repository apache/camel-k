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
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"

	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/envvar"
	"github.com/stretchr/testify/assert"
)

func TestConfigureQuarkusTraitShouldSucceed(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()

	configured, err := quarkusTrait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
}

func TestConfigureDisabledQuarkusTraitShouldFail(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	quarkusTrait.Enabled = new(bool)

	configured, err := quarkusTrait.Configure(environment)

	assert.False(t, configured)
	assert.Nil(t, err)
}

func TestApplyQuarkusTraitDoesNothing(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()

	err := quarkusTrait.Apply(environment)

	assert.Nil(t, err)
}

func TestQuarkusTraitAddBuildStepsShouldSucceed(t *testing.T) {
	quarkusTrait, _ := createNominalQuarkusTest()

	task := &v1.BuilderTask{
		Steps: builder.StepIDsFor(builder.DefaultSteps...),
	}

	quarkusTrait.addBuildSteps(task)

	assert.Len(t, task.Steps, 8)
}

func TestQuarkusTraitAddRuntimeDependenciesShouldSucceed(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()

	err := quarkusTrait.addRuntimeDependencies(environment)

	assert.Nil(t, err)
	assert.Len(t, environment.Integration.Status.Dependencies, 2)
}

func TestQuarkusTraitAddRuntimeDependenciesWithKnativeLoaderShouldSucceed(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.Integration.Spec.Sources[0].Loader = "knative-source"

	err := quarkusTrait.addRuntimeDependencies(environment)

	assert.Nil(t, err)
	assert.Len(t, environment.Integration.Status.Dependencies, 4)
}

func TestQuarkusTraitAddRuntimeDependenciesWithUnsupportedLanguageShouldFail(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.Integration.Spec.Sources[0].Language = "unsupported-language"

	err := quarkusTrait.addRuntimeDependencies(environment)

	assert.NotNil(t, err)
	assert.Equal(t, "unsupported language for Quarkus runtime: unsupported-language (native=false)", err.Error())
	assert.Len(t, environment.Integration.Status.Dependencies, 0)
}

func TestQuarkusTraitAddRuntimeDependenciesWithouSourceDoesNothing(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.Integration.Spec.Sources = nil

	err := quarkusTrait.addRuntimeDependencies(environment)

	assert.Nil(t, err)
	assert.Len(t, environment.Integration.Status.Dependencies, 0)
}

func TestQuarkusTraitAddContainerEnvironmentShouldSucceed(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()

	quarkusTrait.addContainerEnvironment(environment)

	assert.NotNil(t, envvar.Get(environment.EnvVars, envVarAppJAR))
}

func TestQuarkusTraitAddRuntimeDependencyShouldSucceed(t *testing.T) {
	dependency := "depA"
	dependencies := make([]string, 0)
	addRuntimeDependency(dependency, &dependencies)
	assert.Len(t, dependencies, 1)
	assert.Equal(t, "mvn:org.apache.camel.k/depA", dependencies[0])
}

func TestQuarkusTraitDetermineQuarkusVersionShouldReturnDefaultVersion(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()

	version := quarkusTrait.determineQuarkusVersion(environment)

	assert.NotNil(t, version)
}

func TestQuarkusTraitDetermineQuarkusVersionShouldReturnTraitVersion(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	quarkusTrait.QuarkusVersion = "0.0.5-from-trait"

	version := quarkusTrait.determineQuarkusVersion(environment)

	assert.Equal(t, "0.0.5-from-trait", version)
}

func TestQuarkusTraitDetermineQuarkusVersionShouldReturnIntegrationVersion(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.Integration.Status.RuntimeProvider = &v1.RuntimeProvider{
		Quarkus: &v1.QuarkusRuntimeProvider{
			QuarkusVersion: "0.0.5-from-integration",
		},
	}

	version := quarkusTrait.determineQuarkusVersion(environment)

	assert.Equal(t, "0.0.5-from-integration", version)
}

func TestQuarkusTraitDetermineQuarkusVersionShouldReturnIntegrationKitVersion(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.IntegrationKit = &v1.IntegrationKit{
		Status: v1.IntegrationKitStatus{
			RuntimeProvider: &v1.RuntimeProvider{
				Quarkus: &v1.QuarkusRuntimeProvider{
					QuarkusVersion: "0.0.5-from-integration-kit",
				},
			},
		},
	}

	version := quarkusTrait.determineQuarkusVersion(environment)

	assert.Equal(t, "0.0.5-from-integration-kit", version)
}

func TestQuarkusTraitDetermineQuarkusVersionShouldReturnPlatformVersion(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.Platform.Status.Build.RuntimeProvider = &v1.RuntimeProvider{
		Quarkus: &v1.QuarkusRuntimeProvider{
			QuarkusVersion: "0.0.5-from-platform",
		},
	}

	version := quarkusTrait.determineQuarkusVersion(environment)

	assert.Equal(t, "0.0.5-from-platform", version)
}

func TestQuarkusTraitDetermineCamelQuarkusVersionShouldReturnDefaultVersion(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()

	version := quarkusTrait.determineCamelQuarkusVersion(environment)

	assert.NotNil(t, version)
}

func TestQuarkusTraitDetermineCamelQuarkusVersionShouldReturnIntegrationVersion(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.Integration.Status.RuntimeProvider = &v1.RuntimeProvider{
		Quarkus: &v1.QuarkusRuntimeProvider{
			CamelQuarkusVersion: "1.0.5-from-integration",
		},
	}

	version := quarkusTrait.determineCamelQuarkusVersion(environment)

	assert.Equal(t, "1.0.5-from-integration", version)
}

func TestQuarkusTraitDetermineCamelQuarkusVersionShouldReturnIntegrationKitVersion(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.IntegrationKit = &v1.IntegrationKit{
		Status: v1.IntegrationKitStatus{
			RuntimeProvider: &v1.RuntimeProvider{
				Quarkus: &v1.QuarkusRuntimeProvider{
					CamelQuarkusVersion: "1.0.5-from-integration-kit",
				},
			},
		},
	}

	version := quarkusTrait.determineCamelQuarkusVersion(environment)

	assert.Equal(t, "1.0.5-from-integration-kit", version)
}

func TestQuarkusTraitDetermineCamelQuarkusVersionShouldReturnPlatformVersion(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.Platform.Status.Build.RuntimeProvider = &v1.RuntimeProvider{
		Quarkus: &v1.QuarkusRuntimeProvider{
			CamelQuarkusVersion: "1.0.5-from-platform",
		},
	}

	version := quarkusTrait.determineCamelQuarkusVersion(environment)

	assert.Equal(t, "1.0.5-from-platform", version)
}

func createNominalQuarkusTest() (*quarkusTrait, *Environment) {

	trait := newQuarkusTrait()
	enabled := true
	trait.Enabled = &enabled

	environment := &Environment{
		CamelCatalog: &camel.RuntimeCatalog{},
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						Language: v1.LanguageJavaSource,
					},
				},
			},
		},
		Platform: &v1.IntegrationPlatform{},
	}

	return trait, environment
}

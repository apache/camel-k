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

	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/builder"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/test"
)

func TestConfigureQuarkusTraitBuildSubmitted(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseBuildSubmitted

	configured, err := quarkusTrait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)

	err = quarkusTrait.Apply(environment)
	assert.Nil(t, err)

	build := getBuilderTask(environment.Pipeline)
	assert.NotNil(t, t, build)
	assert.Len(t, build.Steps, len(builder.Quarkus.CommonSteps))

	packageTask := getPackageTask(environment.Pipeline)
	assert.NotNil(t, t, packageTask)
	assert.Len(t, packageTask.Steps, 4)
}

func TestApplyQuarkusTraitDefaultKitLayout(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit

	configured, err := quarkusTrait.Configure(environment)
	assert.True(t, configured)
	assert.Nil(t, err)

	err = quarkusTrait.Apply(environment)
	assert.Nil(t, err)
	assert.Len(t, environment.IntegrationKits, 1)
	assert.Equal(t, environment.IntegrationKits[0].Labels[v1.IntegrationKitLayoutLabel], v1.IntegrationKitLayoutFastJar)
}

func TestApplyQuarkusTraitAnnotationKitConfiguration(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit

	v1.SetAnnotation(&environment.Integration.ObjectMeta, v1.TraitAnnotationPrefix+"quarkus.foo", "camel-k")

	configured, err := quarkusTrait.Configure(environment)
	assert.True(t, configured)
	assert.Nil(t, err)

	err = quarkusTrait.Apply(environment)
	assert.Nil(t, err)
	assert.Len(t, environment.IntegrationKits, 1)
	assert.Equal(t, v1.IntegrationKitLayoutFastJar, environment.IntegrationKits[0].Labels[v1.IntegrationKitLayoutLabel])
	assert.Equal(t, "camel-k", environment.IntegrationKits[0].Annotations[v1.TraitAnnotationPrefix+"quarkus.foo"])

}

func createNominalQuarkusTest() (*quarkusTrait, *Environment) {
	trait, _ := newQuarkusTrait().(*quarkusTrait)
	client, _ := test.NewFakeClient()

	environment := &Environment{
		Catalog:      NewCatalog(client),
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
		IntegrationKit: &v1.IntegrationKit{},
		Pipeline: []v1.Task{
			{
				Builder: &v1.BuilderTask{},
			},
			{
				Package: &v1.BuilderTask{},
			},
		},
		Platform: &v1.IntegrationPlatform{},
	}

	return trait, environment
}

func TestGetLanguageSettingsWithoutLoaders(t *testing.T) {
	environment := &Environment{
		CamelCatalog: &camel.RuntimeCatalog{
			CamelCatalogSpec: v1.CamelCatalogSpec{
				Loaders: map[string]v1.CamelLoader{},
			},
		},
	}
	assert.Equal(t, languageSettings{native: false, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageJavaSource))
	assert.Equal(t, languageSettings{native: false, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageGroovy))
	assert.Equal(t, languageSettings{native: false, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageJavaScript))
	assert.Equal(t, languageSettings{native: false, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageKotlin))
	assert.Equal(t, languageSettings{native: false, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageJavaShell))
	assert.Equal(t, languageSettings{native: true, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageKamelet))
	assert.Equal(t, languageSettings{native: true, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageXML))
	assert.Equal(t, languageSettings{native: true, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageYaml))
}

func TestGetLanguageSettingsWithoutMetadata(t *testing.T) {
	environment := &Environment{
		CamelCatalog: &camel.RuntimeCatalog{
			CamelCatalogSpec: v1.CamelCatalogSpec{
				Loaders: map[string]v1.CamelLoader{
					"java":    {},
					"groovy":  {},
					"js":      {},
					"kts":     {},
					"jsh":     {},
					"kamelet": {},
					"xml":     {},
					"yaml":    {},
				},
			},
		},
	}
	assert.Equal(t, languageSettings{native: false, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageJavaSource))
	assert.Equal(t, languageSettings{native: false, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageGroovy))
	assert.Equal(t, languageSettings{native: false, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageJavaScript))
	assert.Equal(t, languageSettings{native: false, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageKotlin))
	assert.Equal(t, languageSettings{native: false, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageJavaShell))
	assert.Equal(t, languageSettings{native: true, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageKamelet))
	assert.Equal(t, languageSettings{native: true, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageXML))
	assert.Equal(t, languageSettings{native: true, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageYaml))
}

func TestGetLanguageSettingsWithLoaders(t *testing.T) {
	environment := &Environment{
		CamelCatalog: &camel.RuntimeCatalog{
			CamelCatalogSpec: v1.CamelCatalogSpec{
				Loaders: map[string]v1.CamelLoader{
					"java": {
						Metadata: map[string]string{
							"native":                         "true",
							"sources-required-at-build-time": "true",
						},
					},
					"groovy": {
						Metadata: map[string]string{
							"native":                         "false",
							"sources-required-at-build-time": "false",
						},
					},
					"js": {
						Metadata: map[string]string{
							"native":                         "true",
							"sources-required-at-build-time": "false",
						},
					},
					"kts": {
						Metadata: map[string]string{
							"native":                         "false",
							"sources-required-at-build-time": "true",
						},
					},
					"jsh": {
						Metadata: map[string]string{
							"native": "true",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, languageSettings{native: true, sourcesRequiredAtBuildTime: true}, getLanguageSettings(environment, v1.LanguageJavaSource))
	assert.Equal(t, languageSettings{native: false, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageGroovy))
	assert.Equal(t, languageSettings{native: true, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageJavaScript))
	assert.Equal(t, languageSettings{native: false, sourcesRequiredAtBuildTime: true}, getLanguageSettings(environment, v1.LanguageKotlin))
	assert.Equal(t, languageSettings{native: true, sourcesRequiredAtBuildTime: false}, getLanguageSettings(environment, v1.LanguageJavaShell))
}

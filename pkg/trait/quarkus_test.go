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

	"github.com/apache/camel-k/v2/pkg/util/boolean"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/builder"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/test"
)

func TestConfigureQuarkusTraitBuildSubmitted(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseBuildSubmitted

	configured, condition, err := quarkusTrait.Configure(environment)

	assert.True(t, configured)
	require.NoError(t, err)
	assert.Nil(t, condition)

	err = quarkusTrait.Apply(environment)
	require.NoError(t, err)

	build := getBuilderTask(environment.Pipeline)
	assert.NotNil(t, t, build)
	assert.Len(t, build.Steps, len(builder.Quarkus.CommonSteps))

	packageTask := getPackageTask(environment.Pipeline)
	assert.NotNil(t, t, packageTask)
	assert.Len(t, packageTask.Steps, 4)
}

func TestConfigureQuarkusTraitNativeNotSupported(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	// Set a source not supporting Quarkus native
	environment.Integration.Spec.Sources[0].Language = v1.LanguageJavaScript
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit
	quarkusTrait.Modes = []traitv1.QuarkusMode{traitv1.NativeQuarkusMode}

	configured, condition, err := quarkusTrait.Configure(environment)

	assert.False(t, configured)
	require.Error(t, err)
	assert.Equal(t, "invalid native support: Integration default/my-it contains a js source that cannot be compiled to native executable", err.Error())
	assert.Nil(t, condition)
}

func TestApplyQuarkusTraitDefaultKitLayout(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit

	configured, condition, err := quarkusTrait.Configure(environment)
	assert.True(t, configured)
	require.NoError(t, err)
	assert.Nil(t, condition)

	err = quarkusTrait.Apply(environment)
	require.NoError(t, err)
	assert.Len(t, environment.IntegrationKits, 1)
	assert.Equal(t, environment.IntegrationKits[0].Labels[v1.IntegrationKitLayoutLabel], v1.IntegrationKitLayoutFastJar)
}

func TestQuarkusTraitBuildModeOrder(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	quarkusTrait.Modes = []traitv1.QuarkusMode{traitv1.NativeQuarkusMode, traitv1.JvmQuarkusMode}
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit
	environment.Integration.Spec.Sources = []v1.SourceSpec{
		{
			Language: v1.LanguageYaml,
		},
	}

	err := quarkusTrait.Apply(environment)
	require.NoError(t, err)
	assert.Len(t, environment.IntegrationKits, 2)
	// assure jvm mode is executed before native mode
	assert.Equal(t, environment.IntegrationKits[0].Labels[v1.IntegrationKitLayoutLabel], v1.IntegrationKitLayoutFastJar)
	assert.Equal(t, environment.IntegrationKits[1].Labels[v1.IntegrationKitLayoutLabel], v1.IntegrationKitLayoutNativeSources)
}

func createNominalQuarkusTest() (*quarkusTrait, *Environment) {
	trait, _ := newQuarkusTrait().(*quarkusTrait)
	client, _ := test.NewFakeClient()

	environment := &Environment{
		Catalog:      NewCatalog(client),
		CamelCatalog: &camel.RuntimeCatalog{},
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "my-it",
			},
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
							"native":                         boolean.TrueString,
							"sources-required-at-build-time": boolean.TrueString,
						},
					},
					"groovy": {
						Metadata: map[string]string{
							"native":                         boolean.FalseString,
							"sources-required-at-build-time": boolean.FalseString,
						},
					},
					"js": {
						Metadata: map[string]string{
							"native":                         boolean.TrueString,
							"sources-required-at-build-time": boolean.FalseString,
						},
					},
					"kts": {
						Metadata: map[string]string{
							"native":                         boolean.FalseString,
							"sources-required-at-build-time": boolean.TrueString,
						},
					},
					"jsh": {
						Metadata: map[string]string{
							"native": boolean.TrueString,
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

func TestQuarkusMatches(t *testing.T) {
	qt := quarkusTrait{
		BasePlatformTrait: NewBasePlatformTrait("quarkus", 600),
		QuarkusTrait: traitv1.QuarkusTrait{
			Modes: []traitv1.QuarkusMode{traitv1.JvmQuarkusMode},
		},
	}
	qt2 := quarkusTrait{
		BasePlatformTrait: NewBasePlatformTrait("quarkus", 600),
		QuarkusTrait: traitv1.QuarkusTrait{
			Modes:           []traitv1.QuarkusMode{traitv1.JvmQuarkusMode},
			NativeBaseImage: QuarkusNativeDefaultBaseImageName,
		},
	}

	assert.True(t, qt.Matches(&qt2))
	qt2.Modes = append(qt2.Modes, traitv1.NativeQuarkusMode)
	assert.True(t, qt.Matches(&qt2))
	qt2.Modes = []traitv1.QuarkusMode{traitv1.NativeQuarkusMode}
	assert.False(t, qt.Matches(&qt2))
	qt2.Modes = nil
	assert.True(t, qt.Matches(&qt2))
	qt2.Modes = []traitv1.QuarkusMode{}
	assert.True(t, qt.Matches(&qt2))
	qt2.NativeBaseImage = "docker.io/my-new-native-base"
	assert.False(t, qt.Matches(&qt2))
}

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

package builder

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/cancellable"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

type testSteps struct {
	TestStep Step
}

func TestRegisterDuplicatedSteps(t *testing.T) {
	steps := testSteps{
		TestStep: NewStep(
			ApplicationPublishPhase,
			func(context *Context) error {
				return nil
			},
		),
	}
	RegisterSteps(steps)
	assert.Panics(t, func() {
		RegisterSteps(steps)
	})
}

func TestGenerateJvmProject(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	ctx := Context{
		Catalog: catalog,
		Build: v1alpha1.BuildSpec{
			RuntimeVersion: defaults.RuntimeVersion,
			Platform: v1alpha1.IntegrationPlatformSpec{
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					CamelVersion: catalog.Version,
				},
			},
			Dependencies: []string{
				"camel-k:runtime-main",
			},
		},
	}

	err = generateProject(&ctx)
	assert.Nil(t, err)
	err = injectDependencies(&ctx)
	assert.Nil(t, err)

	assert.ElementsMatch(
		t,
		ctx.Maven.Project.DependencyManagement.Dependencies,
		[]maven.Dependency{
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "camel-bom",
				Version:    catalog.Version,
				Type:       "pom",
				Scope:      "import",
			},
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-runtime-bom",
				Version:    defaults.RuntimeVersion,
				Type:       "pom",
				Scope:      "import",
			},
		},
	)

	assert.ElementsMatch(
		t,
		ctx.Maven.Project.Dependencies,
		[]maven.Dependency{
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-runtime-main",
			},
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "camel-core-engine",
			},
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "camel-main",
			},
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "camel-properties",
			},
		},
	)
}

func TestMavenSettingsFromConfigMap(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	c, err := test.NewFakeClient(
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "maven-settings",
			},
			Data: map[string]string{
				"settings.xml": "setting-data",
			},
		},
	)

	assert.Nil(t, err)

	ctx := Context{
		Catalog:   catalog,
		Client:    c,
		Namespace: "ns",
		Build: v1alpha1.BuildSpec{
			RuntimeVersion: defaults.RuntimeVersion,
			Platform: v1alpha1.IntegrationPlatformSpec{
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					CamelVersion: catalog.Version,
					Maven: v1alpha1.MavenSpec{
						Settings: v1alpha1.ValueSource{
							ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "maven-settings",
								},
								Key: "settings.xml",
							},
						},
					},
				},
			},
		},
	}

	err = generateProjectSettings(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, []byte("setting-data"), ctx.Maven.SettingsData)
}

func TestMavenSettingsFromSecret(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	c, err := test.NewFakeClient(
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "maven-settings",
			},
			Data: map[string][]byte{
				"settings.xml": []byte("setting-data"),
			},
		},
	)

	assert.Nil(t, err)

	ctx := Context{
		Catalog:   catalog,
		Client:    c,
		Namespace: "ns",
		Build: v1alpha1.BuildSpec{
			RuntimeVersion: defaults.RuntimeVersion,
			Platform: v1alpha1.IntegrationPlatformSpec{
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					CamelVersion: catalog.Version,
					Maven: v1alpha1.MavenSpec{
						Settings: v1alpha1.ValueSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "maven-settings",
								},
								Key: "settings.xml",
							},
						},
					},
				},
			},
		},
	}

	err = generateProjectSettings(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, []byte("setting-data"), ctx.Maven.SettingsData)
}

func TestGenerateGroovyProject(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	ctx := Context{
		Catalog: catalog,
		Build: v1alpha1.BuildSpec{
			RuntimeVersion: defaults.RuntimeVersion,
			Platform: v1alpha1.IntegrationPlatformSpec{
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					CamelVersion: catalog.Version,
				},
			},
			Dependencies: []string{
				"camel-k:runtime-main",
				"camel-k:loader-groovy",
			},
		},
	}

	err = generateProject(&ctx)
	assert.Nil(t, err)
	err = injectDependencies(&ctx)
	assert.Nil(t, err)

	assert.ElementsMatch(
		t,
		ctx.Maven.Project.DependencyManagement.Dependencies,
		[]maven.Dependency{
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "camel-bom",
				Version:    catalog.Version,
				Type:       "pom",
				Scope:      "import",
			},
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-runtime-bom",
				Version:    defaults.RuntimeVersion,
				Type:       "pom",
				Scope:      "import",
			},
		},
	)

	assert.ElementsMatch(
		t,
		ctx.Maven.Project.Dependencies,
		[]maven.Dependency{
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-runtime-main",
			},
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-loader-groovy",
			},
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "camel-core-engine",
			},
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "camel-main",
			},
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "camel-properties",
			},
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "camel-endpointdsl",
			},
			{
				GroupID:    "org.apache.camel",
				ArtifactID: "camel-groovy",
			},
		},
	)
}

func TestSanitizeDependencies(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	ctx := Context{
		Catalog: catalog,
		Build: v1alpha1.BuildSpec{
			RuntimeVersion: defaults.RuntimeVersion,
			Platform: v1alpha1.IntegrationPlatformSpec{
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					CamelVersion: catalog.Version,
				},
			},
			Dependencies: []string{
				"camel:undertow",
				"mvn:org.apache.camel/camel-core/2.18.0",
				"mvn:org.apache.camel.k/camel-k-runtime-main/1.0.0",
				"mvn:com.mycompany/my-dep/1.2.3",
			},
		},
	}

	err = generateProject(&ctx)
	assert.Nil(t, err)
	err = injectDependencies(&ctx)
	assert.Nil(t, err)
	err = sanitizeDependencies(&ctx)
	assert.Nil(t, err)

	assert.Contains(t, ctx.Maven.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel.k",
		ArtifactID: "camel-k-runtime-main",
	})
	assert.Contains(t, ctx.Maven.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel",
		ArtifactID: "camel-core",
	})
	assert.Contains(t, ctx.Maven.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel",
		ArtifactID: "camel-undertow",
	})
	assert.Contains(t, ctx.Maven.Project.Dependencies, maven.Dependency{
		GroupID:    "com.mycompany",
		ArtifactID: "my-dep",
		Version:    "1.2.3",
	})
}

func TestListPublishedImages(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	c, err := test.NewFakeClient(
		&v1alpha1.IntegrationKit{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.IntegrationKindKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-kit-1",
				Labels: map[string]string{
					"camel.apache.org/kit.type": v1alpha1.IntegrationKitTypePlatform,
				},
			},
			Status: v1alpha1.IntegrationKitStatus{
				Phase:        v1alpha1.IntegrationKitPhaseError,
				Image:        "image-1",
				CamelVersion: catalog.Version,
			},
		},
		&v1alpha1.IntegrationKit{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.IntegrationKindKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-kit-2",
				Labels: map[string]string{
					"camel.apache.org/kit.type": v1alpha1.IntegrationKitTypePlatform,
				},
			},
			Status: v1alpha1.IntegrationKitStatus{
				Phase:        v1alpha1.IntegrationKitPhaseReady,
				Image:        "image-2",
				CamelVersion: catalog.Version,
			},
		},
	)

	assert.Nil(t, err)
	assert.NotNil(t, c)

	i, err := listPublishedImages(&Context{
		Client:  c,
		Catalog: catalog,
		C:       cancellable.NewContext(),
	})

	assert.Nil(t, err)
	assert.Len(t, i, 1)
	assert.Equal(t, "image-2", i[0].Image)
}

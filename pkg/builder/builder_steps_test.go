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

func TestMavenRepositories(t *testing.T) {
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
				"runtime:jvm",
			},
			Repositories: []string{
				"https://repository.apache.org/content/repositories/snapshots@id=apache.snapshots@snapshots@noreleases",
				"https://oss.sonatype.org/content/repositories/snapshots/@id=sonatype.snapshots@snapshots",
			},
		},
	}

	err = generateProject(&ctx)
	assert.Nil(t, err)
	err = injectDependencies(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(ctx.Project.Repositories))
	assert.Equal(t, 2, len(ctx.Project.PluginRepositories))

	assert.Contains(t, ctx.Project.Repositories, maven.Repository{
		ID:        "apache.snapshots",
		URL:       "https://repository.apache.org/content/repositories/snapshots",
		Snapshots: maven.RepositoryPolicy{Enabled: true},
		Releases:  maven.RepositoryPolicy{Enabled: false},
	})

	assert.Contains(t, ctx.Project.Repositories, maven.Repository{
		ID:        "sonatype.snapshots",
		URL:       "https://oss.sonatype.org/content/repositories/snapshots/",
		Snapshots: maven.RepositoryPolicy{Enabled: true},
		Releases:  maven.RepositoryPolicy{Enabled: true},
	})

	assert.Contains(t, ctx.Project.PluginRepositories, maven.Repository{
		ID:        "apache.snapshots",
		URL:       "https://repository.apache.org/content/repositories/snapshots",
		Snapshots: maven.RepositoryPolicy{Enabled: true},
		Releases:  maven.RepositoryPolicy{Enabled: false},
	})

	assert.Contains(t, ctx.Project.PluginRepositories, maven.Repository{
		ID:        "sonatype.snapshots",
		URL:       "https://oss.sonatype.org/content/repositories/snapshots/",
		Snapshots: maven.RepositoryPolicy{Enabled: true},
		Releases:  maven.RepositoryPolicy{Enabled: true},
	})
}

func TestMavenProxies(t *testing.T) {
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
				Name:      "maven-proxy",
			},
			Data: map[string]string{
				"proxy-user": "user-from-cm",
			},
		},
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "maven-proxy-secret",
			},
			Data: map[string][]byte{
				"proxy-user": []byte("user-from-secret"),
			},
		},
	)

	ctx := Context{
		Catalog:   catalog,
		Client:    c,
		Namespace: "ns",
		Build: v1alpha1.BuildSpec{
			RuntimeVersion: defaults.RuntimeVersion,
			Platform: v1alpha1.IntegrationPlatformSpec{
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					CamelVersion: catalog.Version,
					Proxies: []v1alpha1.Proxy{
						{
							ID:   "my-poxy-1",
							Host: "my-host",
							Port: 8999,
						},
						{
							ID:   "my-poxy-2",
							Host: "my-cm-host",
							Port: 9998,
							UsernameFrom: &v1alpha1.ValueSource{
								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "maven-proxy",
									},
									Key: "proxy-user",
								},
							},
						},
						{
							ID:   "my-poxy-3",
							Host: "my-secret-host",
							Port: 9988,
							UsernameFrom: &v1alpha1.ValueSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "maven-proxy-secret",
									},
									Key: "proxy-user",
								},
							},
						},
					},
				},
			},
		},
	}

	err = generateProjectSettings(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, 3, len(ctx.Settings.Proxies))

	assert.Contains(t, ctx.Settings.Proxies, maven.Proxy{
		ID:       "my-poxy-1",
		Active:   true,
		Protocol: "http",
		Host:     "my-host",
		Port:     8999,
	})
	assert.Contains(t, ctx.Settings.Proxies, maven.Proxy{
		ID:       "my-poxy-2",
		Active:   true,
		Protocol: "http",
		Host:     "my-cm-host",
		Port:     9998,
		Username: "user-from-cm",
	})
	assert.Contains(t, ctx.Settings.Proxies, maven.Proxy{
		ID:       "my-poxy-3",
		Active:   true,
		Protocol: "http",
		Host:     "my-secret-host",
		Port:     9988,
		Username: "user-from-secret",
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
				"runtime:jvm",
			},
		},
	}

	err = generateProject(&ctx)
	assert.Nil(t, err)
	err = injectDependencies(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(ctx.Project.DependencyManagement.Dependencies))
	assert.Equal(t, "org.apache.camel", ctx.Project.DependencyManagement.Dependencies[0].GroupID)
	assert.Equal(t, "camel-bom", ctx.Project.DependencyManagement.Dependencies[0].ArtifactID)
	assert.Equal(t, catalog.Version, ctx.Project.DependencyManagement.Dependencies[0].Version)
	assert.Equal(t, "pom", ctx.Project.DependencyManagement.Dependencies[0].Type)
	assert.Equal(t, "import", ctx.Project.DependencyManagement.Dependencies[0].Scope)

	assert.Equal(t, 4, len(ctx.Project.Dependencies))
	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel.k",
		ArtifactID: "camel-k-runtime-jvm",
		Version:    defaults.RuntimeVersion,
		Type:       "jar",
	})
	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel",
		ArtifactID: "camel-core",
	})
	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel.k",
		ArtifactID: "camel-k-adapter-camel-2",
	})
	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.logging.log4j",
		ArtifactID: "log4j-slf4j-impl",
		Version:    "2.11.2",
		Scope:      "runtime",
	})
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
				"runtime:groovy",
			},
		},
	}

	err = generateProject(&ctx)
	assert.Nil(t, err)
	err = injectDependencies(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(ctx.Project.DependencyManagement.Dependencies))
	assert.Equal(t, "org.apache.camel", ctx.Project.DependencyManagement.Dependencies[0].GroupID)
	assert.Equal(t, "camel-bom", ctx.Project.DependencyManagement.Dependencies[0].ArtifactID)
	assert.Equal(t, catalog.Version, ctx.Project.DependencyManagement.Dependencies[0].Version)
	assert.Equal(t, "pom", ctx.Project.DependencyManagement.Dependencies[0].Type)
	assert.Equal(t, "import", ctx.Project.DependencyManagement.Dependencies[0].Scope)

	assert.Equal(t, 6, len(ctx.Project.Dependencies))

	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel.k",
		ArtifactID: "camel-k-runtime-jvm",
		Version:    defaults.RuntimeVersion,
		Type:       "jar",
	})
	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel.k",
		ArtifactID: "camel-k-runtime-groovy",
		Version:    defaults.RuntimeVersion,
		Type:       "jar",
	})
	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel",
		ArtifactID: "camel-core",
	})
	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel.k",
		ArtifactID: "camel-k-adapter-camel-2",
	})
	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel",
		ArtifactID: "camel-groovy",
	})
	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.logging.log4j",
		ArtifactID: "log4j-slf4j-impl",
		Version:    "2.11.2",
		Scope:      "runtime",
	})
}

func TestGenerateProjectWithRepositories(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	ctx := Context{
		Catalog: catalog,
		Build: v1alpha1.BuildSpec{
			Platform: v1alpha1.IntegrationPlatformSpec{
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					CamelVersion: catalog.Version,
				},
			},
			Repositories: []string{
				"https://repository.apache.org/content/groups/snapshots-group@id=apache-snapshots@snapshots@noreleases",
				"https://oss.sonatype.org/content/repositories/ops4j-snapshots@id=ops4j-snapshots@snapshots@noreleases",
			},
		},
	}

	err = generateProject(&ctx)
	assert.Nil(t, err)
	err = injectDependencies(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(ctx.Project.DependencyManagement.Dependencies))
	assert.Equal(t, "org.apache.camel", ctx.Project.DependencyManagement.Dependencies[0].GroupID)
	assert.Equal(t, "camel-bom", ctx.Project.DependencyManagement.Dependencies[0].ArtifactID)
	assert.Equal(t, catalog.Version, ctx.Project.DependencyManagement.Dependencies[0].Version)
	assert.Equal(t, "pom", ctx.Project.DependencyManagement.Dependencies[0].Type)
	assert.Equal(t, "import", ctx.Project.DependencyManagement.Dependencies[0].Scope)

	assert.Equal(t, 2, len(ctx.Project.Repositories))
	assert.Equal(t, "apache-snapshots", ctx.Project.Repositories[0].ID)
	assert.False(t, ctx.Project.Repositories[0].Releases.Enabled)
	assert.True(t, ctx.Project.Repositories[0].Snapshots.Enabled)
	assert.Equal(t, "ops4j-snapshots", ctx.Project.Repositories[1].ID)
	assert.False(t, ctx.Project.Repositories[1].Releases.Enabled)
	assert.True(t, ctx.Project.Repositories[1].Snapshots.Enabled)
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
				"mvn:org.apache.camel.k/camel-k-runtime-jvm/1.0.0",
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

	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel.k",
		ArtifactID: "camel-k-runtime-jvm",
		Version:    defaults.RuntimeVersion,
		Type:       "jar",
	})
	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel",
		ArtifactID: "camel-core",
		Type:       "jar",
	})
	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "org.apache.camel",
		ArtifactID: "camel-undertow",
		Type:       "jar",
	})
	assert.Contains(t, ctx.Project.Dependencies, maven.Dependency{
		GroupID:    "com.mycompany",
		ArtifactID: "my-dep",
		Version:    "1.2.3",
		Type:       "jar",
	})
}

func TestListPublishedImages(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	c, err := test.NewFakeClient(
		&v1alpha1.IntegrationContext{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.IntegrationContextKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-context-1",
				Labels: map[string]string{
					"camel.apache.org/context.type": v1alpha1.IntegrationContextTypePlatform,
				},
			},
			Status: v1alpha1.IntegrationContextStatus{
				Phase:        v1alpha1.IntegrationContextPhaseError,
				Image:        "image-1",
				CamelVersion: catalog.Version,
			},
		},
		&v1alpha1.IntegrationContext{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.IntegrationContextKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-context-2",
				Labels: map[string]string{
					"camel.apache.org/context.type": v1alpha1.IntegrationContextTypePlatform,
				},
			},
			Status: v1alpha1.IntegrationContextStatus{
				Phase:        v1alpha1.IntegrationContextPhaseReady,
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

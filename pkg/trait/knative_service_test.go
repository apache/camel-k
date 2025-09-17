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
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/envvar"
	"github.com/apache/camel-k/v2/pkg/util/gzip"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

const (
	KnativeServiceTestNamespace = "ns"
	KnativeServiceTestName      = "test"
)

func TestKnativeService(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("platform-http:test").log("hello");`))
	require.NoError(t, err)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      KnativeServiceTestName,
				Namespace: KnativeServiceTestNamespace,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:        "routes.java",
							Content:     string(compressedRoute),
							Compression: true,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					KnativeService: &traitv1.KnativeServiceTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
					},
					Camel: &traitv1.CamelTrait{
						Properties: []string{
							"my-property=my-property-value",
						},
					},
					Mount: &traitv1.MountTrait{
						Configs: []string{
							"configmap:my-cm", "secret:my-secret",
						},
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative-service"))
	assert.Equal(t, 4, environment.Resources.Size())

	s := environment.Resources.GetKnativeService(func(service *serving.Service) bool {
		return service.Name == KnativeServiceTestName
	})

	assert.NotNil(t, s)

	spec := s.Spec.ConfigurationSpec.Template.Spec

	assert.Len(t, spec.Containers[0].VolumeMounts, 5)
	assert.Len(t, spec.Volumes, 5)

	assert.Condition(t, func() bool {
		for _, v := range spec.Containers[0].VolumeMounts {
			if v.Name == "user-properties" {
				return true
			}
		}
		return false
	})
	assert.Condition(t, func() bool {
		for _, v := range spec.Containers[0].VolumeMounts {
			if v.Name == "my-cm" {
				return true
			}
		}
		return false
	})
	assert.Condition(t, func() bool {
		for _, v := range spec.Volumes {
			if v.Name == "my-secret" {
				return true
			}
		}
		return false
	})

	names := make([]string, 0)
	environment.Resources.VisitConfigMap(func(cm *corev1.ConfigMap) {
		names = append(names, cm.Name)
	})

	assert.Contains(t, names, "test-user-properties")
	assert.Contains(t, names, "test-source-000")

	environment.Resources.VisitConfigMap(func(cm *corev1.ConfigMap) {
		if cm.Name == "test-properties" {
			_, ok := cm.Data["application.properties"]
			assert.True(t, ok)
		}
	})

	assert.Equal(t, "file:/etc/camel/sources/routes.java", environment.ApplicationProperties["camel.k.sources[0].location"])
	assert.Equal(t, "java", environment.ApplicationProperties["camel.k.sources[0].language"])
	assert.Equal(t, boolean.TrueString, environment.ApplicationProperties["camel.k.sources[0].compressed"])
	envVarHasValue(t, spec.Containers[0].Env, "CAMEL_K_CONF", filepath.FromSlash("/etc/camel/application.properties"))
	envVarHasValue(t, spec.Containers[0].Env, "CAMEL_K_CONF_D", filepath.FromSlash("/etc/camel/conf.d"))
}

// envVarHasValue --.
func envVarHasValue(t *testing.T, env []corev1.EnvVar, name string, val string) {
	t.Helper()

	ev := envvar.Get(env, name)
	assert.NotNil(t, ev)
	assert.Equal(t, val, ev.Value)
}

func TestKnativeServiceWithCustomContainerName(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      KnativeServiceTestName,
				Namespace: KnativeServiceTestNamespace,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},

			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Traits: v1.Traits{
					Deployer: &traitv1.DeployerTrait{
						Kind: "knative-service",
					},
					KnativeService: &traitv1.KnativeServiceTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
						Auto: ptr.To(false),
					},
					Container: &traitv1.ContainerTrait{
						Name: "my-container-name",
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative-service"))
	assert.NotNil(t, environment.GetTrait("container"))

	s := environment.Resources.GetKnativeService(func(service *serving.Service) bool {
		return service.Name == KnativeServiceTestName
	})

	assert.NotNil(t, s)

	trait := environment.Integration.Spec.Traits.Container
	assert.Equal(
		t,
		trait.Name,
		s.Spec.ConfigurationSpec.Template.Spec.Containers[0].Name,
	)
}

func TestKnativeServiceWithRest(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      KnativeServiceTestName,
				Namespace: KnativeServiceTestNamespace,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "routes.java",
							Content: `from("direct:test").log("hello");`,
						},
						Language: v1.LanguageJavaSource,
					},
					{
						DataSpec: v1.DataSpec{
							Name:    "rests.xml",
							Content: `<rest path="/test"></rest>`,
						},
						Language: v1.LanguageXML,
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative-service"))

	assert.NotNil(t, environment.Resources.GetKnativeService(func(service *serving.Service) bool {
		return service.Name == KnativeServiceTestName
	}))
}

func TestKnativeServiceNotApplicable(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      KnativeServiceTestName,
				Namespace: KnativeServiceTestNamespace,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "routes.java",
							Content: `from("timer:tick").log("hello");`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.Nil(t, environment.GetTrait("knative-service"))

	assert.Nil(t, environment.Resources.GetKnativeService(func(service *serving.Service) bool {
		return service.Name == KnativeServiceTestName
	}))

	assert.NotNil(t, environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == KnativeServiceTestName
	}))
}

func TestKnativeServiceNoServingAvailable(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	fakeClient := client.(*internal.FakeClient) //nolint
	fakeClient.DisableKnativeServing()

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      KnativeServiceTestName,
				Namespace: KnativeServiceTestNamespace,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "routes.java",
							Content: `from("platform-http:test").log("hello");`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.Nil(t, environment.GetTrait("knative-service"))

	assert.Nil(t, environment.Resources.GetKnativeService(func(service *serving.Service) bool {
		return service.Name == KnativeServiceTestName
	}))

	assert.NotNil(t, environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == KnativeServiceTestName
	}))
}

func TestKnativeServiceWithRollout(t *testing.T) {
	environment := createKnativeServiceTestEnvironment(t, &traitv1.KnativeServiceTrait{RolloutDuration: "60s"})
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative-service"))

	ksvc := environment.Resources.GetKnativeService(func(service *serving.Service) bool {
		return service.Name == KnativeServiceTestName
	})
	assert.NotNil(t, ksvc)

	assert.Equal(t, ksvc.Annotations[knativeServingRolloutDurationAnnotation], "60s")
}

func TestKnativeServiceWithVisibility(t *testing.T) {
	environment := createKnativeServiceTestEnvironment(t, &traitv1.KnativeServiceTrait{
		Visibility: "cluster-local",
	})
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative-service"))

	ksvc := environment.Resources.GetKnativeService(func(service *serving.Service) bool {
		return service.Name == KnativeServiceTestName
	})
	assert.NotNil(t, ksvc)

	assert.Equal(t, ksvc.Labels[knativeServingVisibilityLabel], "cluster-local")
}

func TestKnativeServiceWithTimeoutSeconds(t *testing.T) {
	timeoutSeconds := int64(44)
	environment := createKnativeServiceTestEnvironment(t, &traitv1.KnativeServiceTrait{
		TimeoutSeconds: &timeoutSeconds,
	})
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative-service"))

	ksvc := environment.Resources.GetKnativeService(func(service *serving.Service) bool {
		return service.Name == KnativeServiceTestName
	})
	assert.NotNil(t, ksvc)

	assert.Equal(t, *ksvc.Spec.Template.Spec.TimeoutSeconds, int64(44))
}

func createKnativeServiceTestEnvironment(t *testing.T, trait *traitv1.KnativeServiceTrait) *Environment {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	environment := &Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      KnativeServiceTestName,
				Namespace: KnativeServiceTestNamespace,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "routes.java",
							Content: `from("direct:test").log("hello");`,
						},
						Language: v1.LanguageJavaSource,
					},
					{
						DataSpec: v1.DataSpec{
							Name:    "rests.xml",
							Content: `<rest path="/test"></rest>`,
						},
						Language: v1.LanguageXML,
					},
				},
				Traits: v1.Traits{
					KnativeService: trait,
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterKubernetes,
				Build: v1.IntegrationPlatformBuildSpec{
					RuntimeVersion: catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}

	environment.Platform.ResyncStatusFullConfig()

	_, _, err = traitCatalog.apply(environment)
	require.NoError(t, err)

	return environment
}
func TestServiceAnnotation(t *testing.T) {
	annotationsTest := map[string]string{"haproxy.router.openshift.io/balance": boolean.TrueString}

	environment := createKnativeServiceTestEnvironment(t, &traitv1.KnativeServiceTrait{
		Annotations: map[string]string{"haproxy.router.openshift.io/balance": boolean.TrueString},
	})

	traitsCatalog := environment.Catalog
	_, _, err := traitsCatalog.apply(environment)

	require.NoError(t, err)

	service := environment.Resources.GetKnativeService(func(s *serving.Service) bool {
		return s.Name == KnativeServiceTestName
	})

	assert.NotNil(t, service)
	assert.True(t, reflect.DeepEqual(service.GetAnnotations(), annotationsTest))
}

func TestKnativeServiceAuto(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("platform-http:test").log("hello")`))
	require.NoError(t, err)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      KnativeServiceTestName,
				Namespace: KnativeServiceTestNamespace,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:        "routes.java",
							Content:     string(compressedRoute),
							Compression: true,
						},
						Language: v1.LanguageJavaSource,
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, traits, err := traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.Equal(t, &traitv1.KnativeServiceTrait{
		Trait: traitv1.Trait{
			Enabled: ptr.To(true),
		},
	}, traits.KnativeService)
}

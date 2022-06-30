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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/envvar"
	"github.com/apache/camel-k/pkg/util/gzip"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
)

const (
	KnativeServiceTestNamespace = "ns"
	KnativeServiceTestName      = "test"
)

func TestKnativeService(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("undertow:test").log("hello")`))
	assert.NoError(t, err)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
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
							Name:        "routes.js",
							Content:     string(compressedRoute),
							Compression: true,
						},
						Language: v1.LanguageJavaScript,
					},
				},
				Resources: []v1.ResourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:        "my-resource.txt",
							Content:     "",
							Compression: false,
						},
						Type: v1.ResourceTypeData,
					},
				},
				Configuration: []v1.ConfigurationSpec{
					{Type: "configmap", Value: "my-cm"},
					{Type: "secret", Value: "my-secret"},
					{Type: "property", Value: "my-property=my-property-value"},
				},
				Traits: v1.Traits{
					KnativeService: &v1.KnativeServiceTrait{
						Trait: v1.Trait{
							Enabled: pointer.Bool(true),
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
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1.RegistrySpec{Address: "registry"},
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	err = traitCatalog.apply(&environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative"))
	assert.NotNil(t, envvar.Get(environment.EnvVars, "CAMEL_KNATIVE_CONFIGURATION"))
	assert.Equal(t, 5, environment.Resources.Size())

	s := environment.Resources.GetKnativeService(func(service *serving.Service) bool {
		return service.Name == KnativeServiceTestName
	})

	assert.NotNil(t, s)

	spec := s.Spec.ConfigurationSpec.Template.Spec

	assert.Len(t, spec.Containers[0].VolumeMounts, 6)
	assert.Len(t, spec.Volumes, 6)

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
	assert.Contains(t, names, "test-resource-000")

	environment.Resources.VisitConfigMap(func(cm *corev1.ConfigMap) {
		if cm.Name == "test-properties" {
			_, ok := cm.Data["application.properties"]
			assert.True(t, ok)
		}
	})

	assert.Equal(t, "file:/etc/camel/sources/routes.js", environment.ApplicationProperties["camel.k.sources[0].location"])
	assert.Equal(t, "js", environment.ApplicationProperties["camel.k.sources[0].language"])
	assert.Equal(t, "true", environment.ApplicationProperties["camel.k.sources[0].compressed"])
	test.EnvVarHasValue(t, spec.Containers[0].Env, "CAMEL_K_CONF", "/etc/camel/application.properties")
	test.EnvVarHasValue(t, spec.Containers[0].Env, "CAMEL_K_CONF_D", "/etc/camel/conf.d")
}

func TestKnativeServiceWithCustomContainerName(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
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
					Deployer: &v1.DeployerTrait{
						Kind: "knative-service",
					},
					KnativeService: &v1.KnativeServiceTrait{
						Trait: v1.Trait{
							Enabled: pointer.Bool(true),
						},
						Auto: pointer.Bool(false),
					},
					Container: &v1.ContainerTrait{
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
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1.RegistrySpec{Address: "registry"},
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	err = traitCatalog.apply(&environment)

	assert.Nil(t, err)
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

func TestKnativeServiceWithResr(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
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
							Name:    "routes.js",
							Content: `from("direct:test").log("hello")`,
						},
						Language: v1.LanguageJavaScript,
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
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1.RegistrySpec{Address: "registry"},
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	err = traitCatalog.apply(&environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative"))

	assert.NotNil(t, environment.Resources.GetKnativeService(func(service *serving.Service) bool {
		return service.Name == KnativeServiceTestName
	}))
}

func TestKnativeServiceWithRollout(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
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
							Name:    "routes.js",
							Content: `from("direct:test").log("hello")`,
						},
						Language: v1.LanguageJavaScript,
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
					KnativeService: &v1.KnativeServiceTrait{
						RolloutDuration: "60s",
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
				Cluster: v1.IntegrationPlatformClusterKubernetes,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	err = traitCatalog.apply(&environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative-service"))

	ksvc := environment.Resources.GetKnativeService(func(service *serving.Service) bool {
		return service.Name == KnativeServiceTestName
	})
	assert.NotNil(t, ksvc)

	assert.Equal(t, ksvc.Annotations[knativeServingRolloutDurationAnnotation], "60s")
}

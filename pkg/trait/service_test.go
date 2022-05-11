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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/gzip"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
)

const (
	ServiceTestNamespace = "ns"
	ServiceTestName      = "test"
)

func TestServiceWithDefaults(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("netty-http:test").log("hello")`))
	assert.NoError(t, err)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
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
				Traits: map[string]v1.TraitSpec{
					"service": test.TraitSpecFromMap(t, map[string]interface{}{
						"enabled": true,
						"auto":    false,
					}),
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
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("service"))
	assert.NotNil(t, environment.GetTrait("container"))

	s := environment.Resources.GetService(func(service *corev1.Service) bool {
		return service.Name == ServiceTestName
	})
	d := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == ServiceTestName
	})

	assert.NotNil(t, d)
	assert.NotNil(t, s)

	assert.Len(t, s.Spec.Ports, 1)
	assert.Equal(t, int32(80), s.Spec.Ports[0].Port)
	assert.Equal(t, "http", s.Spec.Ports[0].Name)
	assert.Equal(t, "http", s.Spec.Ports[0].TargetPort.String())

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Len(t, d.Spec.Template.Spec.Containers[0].Ports, 1)
	assert.Equal(t, int32(8080), d.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
	assert.Equal(t, "http", d.Spec.Template.Spec.Containers[0].Ports[0].Name)

	assert.Empty(t, s.Spec.Type) // empty means ClusterIP
}

func TestService(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("netty-http:test").log("hello")`))
	assert.NoError(t, err)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: ServiceTestNamespace,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
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
				Traits: map[string]v1.TraitSpec{
					"service": test.TraitSpecFromMap(t, map[string]interface{}{
						"enabled": true,
					}),
					"container": test.TraitSpecFromMap(t, map[string]interface{}{
						"enabled":         true,
						"auto":            false,
						"expose":          true,
						"port":            8081,
						"portName":        "http-8081",
						"servicePort":     81,
						"servicePortName": "http-81",
					}),
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
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("service"))
	assert.NotNil(t, environment.GetTrait("container"))

	s := environment.Resources.GetService(func(service *corev1.Service) bool {
		return service.Name == ServiceTestName
	})
	d := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == ServiceTestName
	})

	assert.NotNil(t, d)
	assert.NotNil(t, s)

	assert.Len(t, s.Spec.Ports, 1)
	assert.Equal(t, int32(81), s.Spec.Ports[0].Port)
	assert.Equal(t, "http-81", s.Spec.Ports[0].Name)
	assert.Equal(t, "http-8081", s.Spec.Ports[0].TargetPort.String())

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Len(t, d.Spec.Template.Spec.Containers[0].Ports, 1)
	assert.Equal(t, int32(8081), d.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
	assert.Equal(t, "http-8081", d.Spec.Template.Spec.Containers[0].Ports[0].Name)
}

func TestServiceWithCustomContainerName(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: ServiceTestNamespace,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
				Traits: map[string]v1.TraitSpec{
					"service": test.TraitSpecFromMap(t, map[string]interface{}{
						"enabled": true,
						"auto":    false,
					}),
					"container": test.TraitSpecFromMap(t, map[string]interface{}{
						"name": "my-container-name",
					}),
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
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("service"))
	assert.NotNil(t, environment.GetTrait("container"))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)
	assert.NotNil(t, d)

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)

	trait := test.TraitSpecToMap(t, environment.Integration.Spec.Traits["container"])
	assert.Equal(
		t,
		trait["name"],
		d.Spec.Template.Spec.Containers[0].Name,
	)
}

func TestServiceWithNodePort(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("netty-http:test").log("hello")`))
	assert.NoError(t, err)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
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
				Traits: map[string]v1.TraitSpec{
					"service": test.TraitSpecFromMap(t, map[string]interface{}{
						"enabled":  true,
						"auto":     false,
						"nodePort": true,
					}),
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
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("service"))
	assert.NotNil(t, environment.GetTrait("container"))

	s := environment.Resources.GetService(func(service *corev1.Service) bool {
		return service.Name == ServiceTestName
	})
	d := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == ServiceTestName
	})

	assert.NotNil(t, d)
	assert.NotNil(t, s)

	assert.Len(t, s.Spec.Ports, 1)
	assert.Equal(t, int32(80), s.Spec.Ports[0].Port)

	assert.Equal(t, corev1.ServiceTypeNodePort, s.Spec.Type)
}

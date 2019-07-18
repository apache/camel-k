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
	"context"
	"testing"

	"github.com/scylladb/go-set/strset"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

const (
	ServiceTestNamespace = "ns"
	ServiceTestName      = "test"
)

func TestServiceWithDefaults(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(context.TODO(), nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1alpha1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: "ns",
			},
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
			Spec: v1alpha1.IntegrationSpec{
				Profile: v1alpha1.TraitProfileKubernetes,
				Sources: []v1alpha1.SourceSpec{
					{
						DataSpec: v1alpha1.DataSpec{
							Name:        "routes.js",
							Content:     `from("undertow:test").log("hello")`,
							Compression: true,
						},
						Language: v1alpha1.LanguageJavaScript,
					},
				},
				Traits: map[string]v1alpha1.TraitSpec{
					"service": {
						Configuration: map[string]string{
							"enabled": "true",
							"auto":    "false",
						},
					},
				},
			},
		},
		IntegrationKit: &v1alpha1.IntegrationKit{
			Status: v1alpha1.IntegrationKitStatus{
				Phase: v1alpha1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: v1alpha1.IntegrationPlatformClusterOpenShift,
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1alpha1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1alpha1.IntegrationPlatformRegistrySpec{Address: "registry"},
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
		Classpath:      strset.New(),
	}

	err = traitCatalog.apply(&environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait(ID("deployment")))
	assert.NotNil(t, environment.GetTrait(ID("service")))
	assert.NotNil(t, environment.GetTrait(ID("container")))

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
}

func TestService(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(context.TODO(), nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1alpha1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: ServiceTestNamespace,
			},
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
			Spec: v1alpha1.IntegrationSpec{
				Profile: v1alpha1.TraitProfileKubernetes,
				Sources: []v1alpha1.SourceSpec{
					{
						DataSpec: v1alpha1.DataSpec{
							Name:        "routes.js",
							Content:     `from("undertow:test").log("hello")`,
							Compression: true,
						},
						Language: v1alpha1.LanguageJavaScript,
					},
				},
				Traits: map[string]v1alpha1.TraitSpec{
					"service": {
						Configuration: map[string]string{
							"enabled": "true",
						},
					},
					"container": {
						Configuration: map[string]string{
							"enabled":           "true",
							"auto":              "false",
							"expose":            "true",
							"port":              "8081",
							"port-name":         "http-8081",
							"service-port":      "81",
							"service-port-name": "http-81",
						},
					},
				},
			},
		},
		IntegrationKit: &v1alpha1.IntegrationKit{
			Status: v1alpha1.IntegrationKitStatus{
				Phase: v1alpha1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: v1alpha1.IntegrationPlatformClusterOpenShift,
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1alpha1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1alpha1.IntegrationPlatformRegistrySpec{Address: "registry"},
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
		Classpath:      strset.New(),
	}

	err = traitCatalog.apply(&environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait(ID("deployment")))
	assert.NotNil(t, environment.GetTrait(ID("service")))
	assert.NotNil(t, environment.GetTrait(ID("container")))

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
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(context.TODO(), nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1alpha1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: ServiceTestNamespace,
			},
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
			Spec: v1alpha1.IntegrationSpec{
				Profile: v1alpha1.TraitProfileKubernetes,
				Traits: map[string]v1alpha1.TraitSpec{
					"service": {
						Configuration: map[string]string{
							"enabled": "true",
							"auto":    "false",
						},
					},
					"container": {
						Configuration: map[string]string{
							"name": "my-container-name",
						},
					},
				},
			},
		},
		IntegrationKit: &v1alpha1.IntegrationKit{
			Status: v1alpha1.IntegrationKitStatus{
				Phase: v1alpha1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: v1alpha1.IntegrationPlatformClusterOpenShift,
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1alpha1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1alpha1.IntegrationPlatformRegistrySpec{Address: "registry"},
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
		Classpath:      strset.New(),
	}

	err = traitCatalog.apply(&environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait(ID("deployment")))
	assert.NotNil(t, environment.GetTrait(ID("service")))
	assert.NotNil(t, environment.GetTrait(ID("container")))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)
	assert.NotNil(t, d)

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(
		t,
		environment.Integration.Spec.Traits["container"].Configuration["name"],
		d.Spec.Template.Spec.Containers[0].Name,
	)
}

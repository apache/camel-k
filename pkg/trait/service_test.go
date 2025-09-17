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
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/gzip"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

const (
	ServiceTestNamespace = "ns"
	ServiceTestName      = "test"
)

func TestServiceWithDefaults(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("netty-http:test").log("hello");`))
	require.NoError(t, err)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
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
							Name:        "routes.java",
							Content:     string(compressedRoute),
							Compression: true,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Service: &traitv1.ServiceTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
						Auto: ptr.To(false),
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

	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
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
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("netty-http:test").log("hello")`))
	require.NoError(t, err)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
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
							Name:        "routes.java",
							Content:     string(compressedRoute),
							Compression: true,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Service: &traitv1.ServiceTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
					},
					Container: &traitv1.ContainerTrait{
						PlatformBaseTrait: traitv1.PlatformBaseTrait{},
						Auto:              ptr.To(false),
						Expose:            ptr.To(true),
						Port:              8081,
						PortName:          "http-8081",
						ServicePort:       81,
						ServicePortName:   "http-81",
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

	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
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
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
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
				Traits: v1.Traits{
					Service: &traitv1.ServiceTrait{
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

	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("service"))
	assert.NotNil(t, environment.GetTrait("container"))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)
	assert.NotNil(t, d)

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)

	trait := environment.Integration.Spec.Traits.Container
	assert.Equal(
		t,
		trait.Name,
		d.Spec.Template.Spec.Containers[0].Name,
	)
}

func TestServiceWithNodePort(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("netty-http:test").log("hello")`))
	require.NoError(t, err)

	serviceType := traitv1.ServiceTypeNodePort
	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
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
							Name:        "routes.java",
							Content:     string(compressedRoute),
							Compression: true,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Service: &traitv1.ServiceTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
						Auto: ptr.To(false),
						Type: &serviceType,
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

	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
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

// When service and knative-service are enabled at the integration scope in knative profile
// knative-service has the priority and the k8s service is not run.
func TestServiceWithKnativeServiceEnabled(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("netty-http:test").log("hello")`))
	require.NoError(t, err)

	environment := Environment{
		CamelCatalog:   catalog,
		Catalog:        traitCatalog,
		Client:         client,
		IntegrationKit: &v1.IntegrationKit{},
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: "ns",
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
					Service: &traitv1.ServiceTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
						Auto: ptr.To(false),
					},
					KnativeService: &traitv1.KnativeServiceTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
					},
				},
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

	deploymentCondition := NewIntegrationCondition(
		"Deployment",
		v1.IntegrationConditionDeploymentAvailable,
		corev1.ConditionFalse,
		"DeploymentAvailable",
		"controller strategy: knative-service",
	)
	serviceCondition := NewIntegrationCondition(
		"Service",
		v1.IntegrationConditionTraitInfo,
		corev1.ConditionTrue,
		"TraitConfiguration",
		"explicitly disabled by the platform: knative-service trait has priority over this trait",
	)
	conditions, traits, err := traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.Contains(t, conditions, deploymentCondition)
	assert.Contains(t, conditions, serviceCondition)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.Nil(t, environment.GetTrait(serviceTraitID))
	assert.NotNil(t, environment.GetTrait(knativeServiceTraitID))
}

func TestServicesWithKnativeProfile(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("netty-http:test").log("hello")`))
	require.NoError(t, err)

	environment := Environment{
		CamelCatalog:   catalog,
		Catalog:        traitCatalog,
		Client:         client,
		IntegrationKit: &v1.IntegrationKit{},
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: "ns",
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

	deploymentCondition := NewIntegrationCondition(
		"Deployment",
		v1.IntegrationConditionDeploymentAvailable,
		corev1.ConditionFalse,
		"DeploymentAvailable",
		"controller strategy: knative-service",
	)
	serviceCondition := NewIntegrationCondition(
		"Service",
		v1.IntegrationConditionTraitInfo,
		corev1.ConditionTrue,
		"TraitConfiguration",
		"explicitly disabled by the platform: knative-service trait has priority over this trait",
	)
	conditions, traits, err := traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.Contains(t, conditions, deploymentCondition)
	assert.Contains(t, conditions, serviceCondition)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.Nil(t, environment.GetTrait(serviceTraitID))
	assert.NotNil(t, environment.GetTrait(knativeServiceTraitID))
}

// When the knative-service is disabled at the IntegrationPlatform, the k8s service is enabled.
func TestServiceWithKnativeServiceDisabledInIntegrationPlatform(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("netty-http:test").log("hello")`))
	require.NoError(t, err)

	environment := Environment{
		CamelCatalog:   catalog,
		Catalog:        traitCatalog,
		Client:         client,
		IntegrationKit: &v1.IntegrationKit{},
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: "ns",
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
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
				Traits: v1.Traits{
					KnativeService: &traitv1.KnativeServiceTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(false),
						},
					},
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

	expectedCondition := NewIntegrationCondition(
		"KnativeService",
		v1.IntegrationConditionKnativeServiceAvailable,
		corev1.ConditionFalse,
		"KnativeServiceNotAvailable",
		"explicitly disabled",
	)
	conditions, traits, err := traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.Contains(t, conditions, expectedCondition)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait(serviceTraitID))
	assert.Nil(t, environment.GetTrait(knativeServiceTraitID))
}

func TestServiceAutoConfiguration(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("netty-http:test").log("hello")`))
	require.NoError(t, err)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
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

	_, traits, err := traitCatalog.apply(&environment)
	require.NoError(t, err)
	assert.Equal(t, ptr.To(true), traits.Service.Enabled)
}

func TestServiceAnnotationsAndLables(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("netty-http:test").log("hello")`))
	require.NoError(t, err)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
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
							Name:        "routes.java",
							Content:     string(compressedRoute),
							Compression: true,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Service: &traitv1.ServiceTrait{
						Auto: ptr.To(true),
						Annotations: map[string]string{
							"annotation-1": "value-1",
							"annotation-2": "value-2",
						},
						Labels: map[string]string{
							"label-1": "v1",
							"label-2": "v2",
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

	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	s := environment.Resources.GetService(func(service *corev1.Service) bool {
		return service.Name == ServiceTestName
	})
	assert.NotNil(t, s)
	assert.NotNil(t, s.Annotations)
	assert.Len(t, s.Annotations, 2)
	assert.Equal(t, "value-1", s.Annotations["annotation-1"])
	assert.Equal(t, "value-2", s.Annotations["annotation-2"])
	// There are other labels added by other traits
	assert.Len(t, s.Labels, 5)
	assert.Equal(t, "v1", s.Labels["label-1"])
	assert.Equal(t, "v2", s.Labels["label-2"])
	assert.Equal(t, ServiceTestName, s.Labels[v1.IntegrationLabel])
}

func TestServicePorts(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("timer:test").log("hello")`))
	require.NoError(t, err)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
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
							Name:        "routes.java",
							Content:     string(compressedRoute),
							Compression: true,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Service: &traitv1.ServiceTrait{
						Ports: []string{"my-port-1;1;8001", "my-port-udp;2;8002;UDP"},
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

	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	s := environment.Resources.GetService(func(service *corev1.Service) bool {
		return service.Name == ServiceTestName
	})
	assert.NotNil(t, s)
	// TODO: while we don't deprecate the usage of container.port
	// we must assume that the default port is exposed as there is no way to
	// know if the managed port is available or not
	//assert.Len(t, s.Spec.Ports, 2)
	assert.Len(t, s.Spec.Ports, 3)
	assert.Contains(t, s.Spec.Ports, corev1.ServicePort{
		Name:       "my-port-1",
		Port:       1,
		TargetPort: intstr.FromInt32(8001),
		Protocol:   corev1.ProtocolTCP,
	})
	assert.Contains(t, s.Spec.Ports, corev1.ServicePort{
		Name:       "my-port-udp",
		Port:       2,
		TargetPort: intstr.FromInt32(8002),
		Protocol:   corev1.ProtocolUDP,
	})
}

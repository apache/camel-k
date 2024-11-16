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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	serving "knative.dev/serving/pkg/apis/serving/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func TestContainerWithDefaults(t *testing.T) {
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
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
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

	conditions, traits, err := traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.Empty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("container"))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)

	assert.NotNil(t, d)
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, defaultContainerName, d.Spec.Template.Spec.Containers[0].Name)
}

func TestContainerWithOpenshift(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	// Integration is in another constrained namespace
	constrainedIntNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "myuser",
			Annotations: map[string]string{
				"openshift.io/sa.scc.mcs":                 "s0:c26,c5",
				"openshift.io/sa.scc.supplemental-groups": "1000860000/10000",
				"openshift.io/sa.scc.uid-range":           "1000860000/10000",
			},
		},
	}

	client, _ := internal.NewFakeClient(constrainedIntNamespace)
	traitCatalog := NewCatalog(nil)

	// enable openshift
	fakeClient := client.(*internal.FakeClient) //nolint
	fakeClient.EnableOpenshiftDiscovery()

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: "myuser",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
			},
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
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

	conditions, traits, err := traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("container"))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)

	assert.NotNil(t, d)
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, defaultContainerName, d.Spec.Template.Spec.Containers[0].Name)
	assert.Equal(t, ptr.To(defaultContainerRunAsNonRoot), d.Spec.Template.Spec.Containers[0].SecurityContext.RunAsNonRoot)
	assert.Equal(t, ptr.To(int64(1000860000)), d.Spec.Template.Spec.Containers[0].SecurityContext.RunAsUser)
}

func TestContainerWithCustomName(t *testing.T) {
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
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
				Traits: v1.Traits{
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
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
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

	conditions, traits, err := traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("container"))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)

	assert.NotNil(t, d)
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)

	trait := environment.Integration.Spec.Traits.Container
	assert.Equal(t, trait.Name, d.Spec.Template.Spec.Containers[0].Name)
}

func TestContainerWithCustomImage(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	environment := Environment{
		Ctx:          context.TODO(),
		Client:       client,
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: "ns",
				UID:       types.UID(uuid.NewString()),
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
				Traits: v1.Traits{
					Container: &traitv1.ContainerTrait{
						Image: "foo/bar:1.0.0",
					},
				},
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
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

	conditions, traits, err := traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)

	for _, postAction := range environment.PostActions {
		require.NoError(t, postAction(&environment))
	}

	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployer"))
	assert.NotNil(t, environment.GetTrait("container"))
	assert.Nil(t, environment.Integration.Status.IntegrationKit)
	assert.Equal(t, environment.Integration.Spec.Traits.Container.Image, environment.Integration.Status.Image)
}

func TestContainerWithCustomImageAndIntegrationKit(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	environment := Environment{
		Ctx:          context.TODO(),
		Client:       client,
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: "ns",
				UID:       types.UID(uuid.NewString()),
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
				Traits: v1.Traits{
					Container: &traitv1.ContainerTrait{
						Image: "foo/bar:1.0.0",
					},
				},
				IntegrationKit: &corev1.ObjectReference{
					Name:      "bad-" + ServiceTestName,
					Namespace: "ns",
				},
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
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

	conditions, traits, err := traitCatalog.apply(&environment)
	require.Error(t, err)
	assert.Empty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.Contains(t, err.Error(), "unsupported configuration: a container image has been set in conjunction with an IntegrationKit")
}

func TestContainerWithImagePullPolicy(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	environment := Environment{
		Ctx:            context.TODO(),
		Client:         client,
		CamelCatalog:   catalog,
		Catalog:        traitCatalog,
		IntegrationKit: &v1.IntegrationKit{},
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
				Traits: v1.Traits{
					Container: &traitv1.ContainerTrait{
						ImagePullPolicy: "Always",
					},
				},
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Build: v1.IntegrationPlatformBuildSpec{
					RuntimeVersion: catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		Resources: kubernetes.NewCollection(),
	}
	environment.Integration.Status.Phase = v1.IntegrationPhaseDeploying
	environment.Platform.ResyncStatusFullConfig()

	conditions, traits, err := traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)

	container := environment.GetIntegrationContainer()

	assert.Equal(t, corev1.PullAlways, container.ImagePullPolicy)
}

func TestDeploymentContainerPorts(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	environment := Environment{
		Ctx:            context.TODO(),
		Client:         client,
		CamelCatalog:   catalog,
		Catalog:        traitCatalog,
		IntegrationKit: &v1.IntegrationKit{},
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
				Traits: v1.Traits{
					Container: &traitv1.ContainerTrait{
						Port:        8081,
						ServicePort: 8081,
					},
				},
				Sources: []v1.SourceSpec{
					{
						Language: v1.LanguageJavaSource,
						DataSpec: v1.DataSpec{
							Name: "MyTest.java",
							Content: `
							public class MyRouteBuilder extends RouteBuilder {
								@Override
								public void configure() throws Exception {
									from("netty-http:http://0.0.0.0:8081/hello").log("Received message: ${body}");
								}
							}
							`,
						},
					},
				},
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Build: v1.IntegrationPlatformBuildSpec{
					RuntimeVersion: catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		Resources: kubernetes.NewCollection(),
	}
	environment.Integration.Status.Phase = v1.IntegrationPhaseDeploying
	environment.Platform.ResyncStatusFullConfig()

	_, _, err = traitCatalog.apply(&environment)
	require.NoError(t, err)
	container := environment.GetIntegrationContainer()
	assert.Len(t, container.Ports, 1)
	assert.Equal(t, int32(8081), container.Ports[0].ContainerPort)
	assert.Equal(t, "http", container.Ports[0].Name)
	svc := environment.Resources.GetServiceForIntegration(environment.Integration)
	assert.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(8081), svc.Spec.Ports[0].Port)
}

func TestKnativeServiceContainerPorts(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	environment := Environment{
		Ctx:            context.TODO(),
		Client:         client,
		CamelCatalog:   catalog,
		Catalog:        traitCatalog,
		IntegrationKit: &v1.IntegrationKit{},
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Traits: v1.Traits{
					Container: &traitv1.ContainerTrait{
						Port:        8081,
						ServicePort: 8081,
					},
				},
				Sources: []v1.SourceSpec{
					{
						Language: v1.LanguageJavaSource,
						DataSpec: v1.DataSpec{
							Name: "MyTest.java",
							Content: `
							public class MyRouteBuilder extends RouteBuilder {
								@Override
								public void configure() throws Exception {
									from("netty-http:http://0.0.0.0:8081/hello").log("Received message: ${body}");
								}
							}
							`,
						},
					},
				},
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Build: v1.IntegrationPlatformBuildSpec{
					RuntimeVersion: catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		Resources: kubernetes.NewCollection(),
	}
	environment.Integration.Status.Phase = v1.IntegrationPhaseDeploying
	environment.Platform.ResyncStatusFullConfig()

	_, _, err = traitCatalog.apply(&environment)
	require.NoError(t, err)
	container := environment.GetIntegrationContainer()
	assert.Len(t, container.Ports, 1)
	assert.Equal(t, int32(8081), container.Ports[0].ContainerPort)
	assert.Equal(t, defaultKnativeContainerPortName, container.Ports[0].Name)
}

func TestDefaultKubernetesSecurityContext(t *testing.T) {
	environment := createSettingContextEnvironment(t, v1.TraitProfileKubernetes)
	traitCatalog := NewCatalog(nil)

	conditions, traits, err := traitCatalog.apply(environment)

	require.NoError(t, err)
	assert.Empty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("container"))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)

	assert.NotNil(t, d)
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, defaultContainerName, d.Spec.Template.Spec.Containers[0].Name)
	assert.Equal(t, ptr.To(defaultContainerRunAsNonRoot), d.Spec.Template.Spec.Containers[0].SecurityContext.RunAsNonRoot)
	assert.Nil(t, d.Spec.Template.Spec.Containers[0].SecurityContext.RunAsUser)
	assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, d.Spec.Template.Spec.Containers[0].SecurityContext.SeccompProfile.Type)
	assert.Equal(t, ptr.To(defaultContainerAllowPrivilegeEscalation), d.Spec.Template.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation)
	assert.Equal(t, []corev1.Capability{defaultContainerCapabilitiesDrop}, d.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities.Drop)
	assert.Nil(t, d.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities.Add)
}

func TestDefaultKnativeSecurityContext(t *testing.T) {
	environment := createSettingContextEnvironment(t, v1.TraitProfileKnative)
	environment.Integration.Spec.Traits.KnativeService = &traitv1.KnativeServiceTrait{
		Trait: traitv1.Trait{
			Enabled: ptr.To(true),
		},
	}
	traitCatalog := NewCatalog(nil)

	conditions, traits, err := traitCatalog.apply(environment)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.Nil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("knative-service"))
	assert.NotNil(t, environment.GetTrait("container"))

	s := environment.Resources.GetKnativeService(func(service *serving.Service) bool {
		return service.Name == ServiceTestName
	})

	assert.NotNil(t, s)
	assert.Len(t, s.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, defaultContainerName, s.Spec.Template.Spec.Containers[0].Name)
	assert.Equal(t, ptr.To(defaultContainerRunAsNonRoot), s.Spec.Template.Spec.Containers[0].SecurityContext.RunAsNonRoot)
	assert.Nil(t, s.Spec.Template.Spec.Containers[0].SecurityContext.RunAsUser)
	assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, s.Spec.Template.Spec.Containers[0].SecurityContext.SeccompProfile.Type)
	assert.Equal(t, ptr.To(defaultContainerAllowPrivilegeEscalation), s.Spec.Template.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation)
	assert.Equal(t, []corev1.Capability{defaultContainerCapabilitiesDrop}, s.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities.Drop)
	assert.Nil(t, s.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities.Add)
}

func TestUserSecurityContext(t *testing.T) {
	environment := createSettingContextEnvironment(t, v1.TraitProfileKubernetes)
	environment.Integration.Spec.Traits = v1.Traits{
		Container: &traitv1.ContainerTrait{
			RunAsNonRoot:             ptr.To(false),
			RunAsUser:                ptr.To(int64(1000)),
			SeccompProfileType:       "Unconfined",
			AllowPrivilegeEscalation: ptr.To(true),
			CapabilitiesDrop:         []corev1.Capability{"DROP"},
			CapabilitiesAdd:          []corev1.Capability{"ADD"},
		},
	}
	traitCatalog := NewCatalog(nil)

	conditions, traits, err := traitCatalog.apply(environment)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("container"))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)

	assert.NotNil(t, d)
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, ptr.To(false), d.Spec.Template.Spec.Containers[0].SecurityContext.RunAsNonRoot)
	assert.Equal(t, ptr.To(int64(1000)), d.Spec.Template.Spec.Containers[0].SecurityContext.RunAsUser)
	assert.Equal(t, corev1.SeccompProfileTypeUnconfined, d.Spec.Template.Spec.Containers[0].SecurityContext.SeccompProfile.Type)
	assert.Equal(t, ptr.To(true), d.Spec.Template.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation)
	assert.Equal(t, []corev1.Capability{"DROP"}, d.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities.Drop)
	assert.Equal(t, []corev1.Capability{"ADD"}, d.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities.Add)
}

func TestUserDefaultResources(t *testing.T) {
	environment := createSettingContextEnvironment(t, v1.TraitProfileKubernetes)
	traitCatalog := NewCatalog(nil)
	conditions, traits, err := traitCatalog.apply(environment)

	require.NoError(t, err)
	assert.Empty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("container"))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)

	assert.NotNil(t, d)
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, resource.MustParse("500m"), *d.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu())
	assert.Equal(t, resource.MustParse("125m"), *d.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu())
	assert.Equal(t, resource.MustParse("512Mi"), *d.Spec.Template.Spec.Containers[0].Resources.Limits.Memory())
	assert.Equal(t, resource.MustParse("128Mi"), *d.Spec.Template.Spec.Containers[0].Resources.Requests.Memory())
}

func createSettingContextEnvironment(t *testing.T, profile v1.TraitProfile) *Environment {
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
				Namespace: "myuser",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: profile,
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
			},
			Spec: v1.IntegrationPlatformSpec{
				Build: v1.IntegrationPlatformBuildSpec{
					Registry:       v1.RegistrySpec{Address: "registry"},
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

	return &environment
}

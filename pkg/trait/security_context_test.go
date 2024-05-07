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
	serving "knative.dev/serving/pkg/apis/serving/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/test"
)

func TestDefaultPodKubernetesSecurityContextInitializationPhase(t *testing.T) {
	environment := createPodSettingContextEnvironment(t, v1.TraitProfileKubernetes)
	environment.Integration.Status.Phase = v1.IntegrationPhaseInitialization
	traitCatalog := NewCatalog(nil)

	conditions, err := traitCatalog.apply(environment)

	require.NoError(t, err)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.Nil(t, environment.GetTrait("security-context"))
}

func TestDefaultPodKubernetesSecurityContext(t *testing.T) {
	environment := createPodSettingContextEnvironment(t, v1.TraitProfileKubernetes)
	traitCatalog := NewCatalog(nil)

	conditions, err := traitCatalog.apply(environment)

	require.NoError(t, err)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("security-context"))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)

	assert.NotNil(t, d)
	assert.Equal(t, pointer.Bool(defaultPodRunAsNonRoot), d.Spec.Template.Spec.SecurityContext.RunAsNonRoot)
	assert.Nil(t, d.Spec.Template.Spec.SecurityContext.RunAsUser)
	assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, d.Spec.Template.Spec.SecurityContext.SeccompProfile.Type)
}

func TestDefaultPodOpenshiftSecurityContext(t *testing.T) {
	environment := createOpenshiftPodSettingContextEnvironment(t, v1.TraitProfileOpenShift)
	traitCatalog := NewCatalog(nil)

	conditions, err := traitCatalog.apply(environment)

	require.NoError(t, err)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("security-context"))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)

	assert.NotNil(t, d)
	assert.Equal(t, pointer.Bool(defaultPodRunAsNonRoot), d.Spec.Template.Spec.SecurityContext.RunAsNonRoot)
	assert.NotNil(t, d.Spec.Template.Spec.SecurityContext.RunAsUser)
	assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, d.Spec.Template.Spec.SecurityContext.SeccompProfile.Type)
}

func TestDefaultPodKnativeSecurityContext(t *testing.T) {
	environment := createPodSettingContextEnvironment(t, v1.TraitProfileKnative)
	traitCatalog := NewCatalog(nil)

	conditions, err := traitCatalog.apply(environment)

	require.NoError(t, err)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.Nil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("knative-service"))
	assert.Nil(t, environment.GetTrait("security-context"))

	s := environment.Resources.GetKnativeService(func(service *serving.Service) bool {
		return service.Name == ServiceTestName
	})

	assert.NotNil(t, s)
	assert.Nil(t, s.Spec.Template.Spec.SecurityContext)
}

func TestUserPodSecurityContext(t *testing.T) {
	environment := createPodSettingContextEnvironment(t, v1.TraitProfileKubernetes)
	environment.Integration.Spec.Traits = v1.Traits{
		SecurityContext: &traitv1.SecurityContextTrait{
			RunAsNonRoot:       pointer.Bool(false),
			RunAsUser:          pointer.Int64(1000),
			SeccompProfileType: "Unconfined",
		},
	}
	traitCatalog := NewCatalog(nil)

	conditions, err := traitCatalog.apply(environment)

	require.NoError(t, err)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("security-context"))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)

	assert.NotNil(t, d)
	assert.Equal(t, pointer.Bool(false), d.Spec.Template.Spec.SecurityContext.RunAsNonRoot)
	assert.Equal(t, pointer.Int64(1000), d.Spec.Template.Spec.SecurityContext.RunAsUser)
	assert.Equal(t, corev1.SeccompProfileTypeUnconfined, d.Spec.Template.Spec.SecurityContext.SeccompProfile.Type)
}

func createPodSettingContextEnvironment(t *testing.T, profile v1.TraitProfile) *Environment {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)
	client, _ := test.NewFakeClient()
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

func createOpenshiftPodSettingContextEnvironment(t *testing.T, profile v1.TraitProfile) *Environment {
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

	client, _ := test.NewFakeClient(constrainedIntNamespace)
	traitCatalog := NewCatalog(nil)

	// enable openshift
	fakeClient := client.(*test.FakeClient) //nolint
	fakeClient.EnableOpenshiftDiscovery()
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

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

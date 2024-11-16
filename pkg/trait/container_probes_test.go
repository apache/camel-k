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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func newTestProbesEnv(t *testing.T, integration *v1.Integration) Environment {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)
	assert.NotNil(t, catalog)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	return Environment{
		Catalog:      traitCatalog,
		CamelCatalog: catalog,
		Client:       client,
		Platform: &v1.IntegrationPlatform{
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
				IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
					Build: v1.IntegrationPlatformBuildSpec{
						RuntimeVersion: catalog.Runtime.Version,
					},
				},
			},
		},
		IntegrationKit:        &v1.IntegrationKit{},
		Integration:           integration,
		Resources:             kubernetes.NewCollection(),
		ApplicationProperties: make(map[string]string),
	}
}

func TestProbesDependencies(t *testing.T) {
	integration := &v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Health: &traitv1.HealthTrait{
					Trait: traitv1.Trait{
						Enabled: ptr.To(true),
					},
				},
			},
		},
	}

	env := newTestProbesEnv(t, integration)
	env.Integration.Status.Phase = v1.IntegrationPhaseInitialization

	conditions, traits, err := env.Catalog.apply(&env)
	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.Contains(t, env.Integration.Status.Dependencies, "mvn:org.apache.camel.quarkus:camel-quarkus-microprofile-health")
}

func TestProbesOnDeployment(t *testing.T) {
	integration := &v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Health: &traitv1.HealthTrait{
					Trait: traitv1.Trait{
						Enabled: ptr.To(true),
					},
					LivenessProbeEnabled:  ptr.To(true),
					ReadinessProbeEnabled: ptr.To(true),
					LivenessTimeout:       1234,
				},
			},
		},
	}

	env := newTestProbesEnv(t, integration)
	env.Integration.Status.Phase = v1.IntegrationPhaseDeploying

	conditions, traits, err := env.Catalog.apply(&env)
	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)

	container := env.GetIntegrationContainer()

	assert.Equal(t, "", container.LivenessProbe.HTTPGet.Host)
	assert.Equal(t, int32(defaultContainerPort), container.LivenessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, defaultLivenessProbePath, container.LivenessProbe.HTTPGet.Path)
	assert.Equal(t, corev1.URISchemeHTTP, container.ReadinessProbe.HTTPGet.Scheme)
	assert.Equal(t, "", container.ReadinessProbe.HTTPGet.Host)
	assert.Equal(t, int32(defaultContainerPort), container.ReadinessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, defaultReadinessProbePath, container.ReadinessProbe.HTTPGet.Path)
	assert.Equal(t, corev1.URISchemeHTTP, container.LivenessProbe.HTTPGet.Scheme)
	assert.Equal(t, int32(1234), container.LivenessProbe.TimeoutSeconds)
}

func TestProbesOnDeploymentWithCustomScheme(t *testing.T) {
	integration := &v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Health: &traitv1.HealthTrait{
					Trait: traitv1.Trait{
						Enabled: ptr.To(true),
					},
					LivenessProbeEnabled:  ptr.To(true),
					ReadinessProbeEnabled: ptr.To(true),
					LivenessScheme:        "HTTPS",
					ReadinessScheme:       "HTTPS",
					LivenessTimeout:       1234,
				},
			},
		},
	}

	env := newTestProbesEnv(t, integration)
	env.Integration.Status.Phase = v1.IntegrationPhaseDeploying

	conditions, traits, err := env.Catalog.apply(&env)
	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)

	container := env.GetIntegrationContainer()

	assert.Equal(t, "", container.LivenessProbe.HTTPGet.Host)
	assert.Equal(t, int32(defaultContainerPort), container.LivenessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, defaultLivenessProbePath, container.LivenessProbe.HTTPGet.Path)
	assert.Equal(t, corev1.URISchemeHTTPS, container.ReadinessProbe.HTTPGet.Scheme)
	assert.Equal(t, "", container.ReadinessProbe.HTTPGet.Host)
	assert.Equal(t, int32(defaultContainerPort), container.ReadinessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, defaultReadinessProbePath, container.ReadinessProbe.HTTPGet.Path)
	assert.Equal(t, corev1.URISchemeHTTPS, container.LivenessProbe.HTTPGet.Scheme)
	assert.Equal(t, int32(1234), container.LivenessProbe.TimeoutSeconds)
}

func TestProbesOnKnativeService(t *testing.T) {
	integration := &v1.Integration{
		Spec: v1.IntegrationSpec{
			Profile: v1.TraitProfileKnative,
			Traits: v1.Traits{
				KnativeService: &traitv1.KnativeServiceTrait{
					Trait: traitv1.Trait{
						Enabled: ptr.To(true),
					},
				},
				Health: &traitv1.HealthTrait{
					Trait: traitv1.Trait{
						Enabled: ptr.To(true),
					},
					LivenessProbeEnabled:  ptr.To(true),
					ReadinessProbeEnabled: ptr.To(true),
					LivenessTimeout:       1234,
				},
			},
		},
	}

	env := newTestProbesEnv(t, integration)
	env.Integration.Status.Phase = v1.IntegrationPhaseDeploying

	serviceOverrideCondition := NewIntegrationCondition(
		"Service",
		v1.IntegrationConditionTraitInfo,
		corev1.ConditionTrue,
		"TraitConfiguration",
		"explicitly disabled by the platform: knative-service trait has priority over this trait",
	)
	ctrlStrategyCondition := NewIntegrationCondition(
		"Deployment",
		v1.IntegrationConditionDeploymentAvailable,
		corev1.ConditionFalse,
		"DeploymentAvailable",
		"controller strategy: knative-service",
	)

	conditions, traits, err := env.Catalog.apply(&env)
	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.Contains(t, conditions, ctrlStrategyCondition)
	assert.Contains(t, conditions, serviceOverrideCondition)

	container := env.GetIntegrationContainer()

	assert.Equal(t, "", container.LivenessProbe.HTTPGet.Host)
	assert.Equal(t, int32(defaultContainerPort), container.LivenessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, defaultLivenessProbePath, container.LivenessProbe.HTTPGet.Path)
	assert.Equal(t, "", container.ReadinessProbe.HTTPGet.Host)
	assert.Equal(t, int32(defaultContainerPort), container.ReadinessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, defaultReadinessProbePath, container.ReadinessProbe.HTTPGet.Path)
	assert.Equal(t, int32(1234), container.LivenessProbe.TimeoutSeconds)
}

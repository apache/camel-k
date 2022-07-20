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
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func newTestProbesEnv(t *testing.T, integration *v1.Integration) Environment {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)
	assert.NotNil(t, catalog)

	traitCatalog := NewCatalog(nil)

	return Environment{
		Catalog:               traitCatalog,
		CamelCatalog:          catalog,
		Platform:              &v1.IntegrationPlatform{},
		Integration:           integration,
		Resources:             kubernetes.NewCollection(),
		ApplicationProperties: make(map[string]string),
	}
}

func TestProbesDependencies(t *testing.T) {
	integration := &v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Container: &traitv1.ContainerTrait{
					DeprecatedProbesEnabled: pointer.Bool(true),
				},
			},
		},
	}

	env := newTestProbesEnv(t, integration)
	env.Integration.Status.Phase = v1.IntegrationPhaseInitialization

	err := env.Catalog.apply(&env)
	assert.Nil(t, err)

	assert.Contains(t, env.Integration.Status.Dependencies,
		"mvn:org.apache.camel.quarkus:camel-quarkus-microprofile-health")
}

func TestProbesOnDeployment(t *testing.T) {
	integration := &v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Container: &traitv1.ContainerTrait{
					DeprecatedProbesEnabled:   pointer.Bool(true),
					Expose:                    pointer.Bool(true),
					DeprecatedLivenessTimeout: 1234,
				},
			},
		},
	}

	env := newTestProbesEnv(t, integration)
	env.Integration.Status.Phase = v1.IntegrationPhaseDeploying

	err := env.Catalog.apply(&env)
	assert.Nil(t, err)

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
				Container: &traitv1.ContainerTrait{
					DeprecatedProbesEnabled:   pointer.Bool(true),
					Expose:                    pointer.Bool(true),
					DeprecatedLivenessTimeout: 1234,
					DeprecatedLivenessScheme:  "HTTPS",
					DeprecatedReadinessScheme: "HTTPS",
				},
			},
		},
	}

	env := newTestProbesEnv(t, integration)
	env.Integration.Status.Phase = v1.IntegrationPhaseDeploying

	err := env.Catalog.apply(&env)
	assert.Nil(t, err)

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
						Enabled: pointer.Bool(true),
					},
				},
				Container: &traitv1.ContainerTrait{
					DeprecatedProbesEnabled:   pointer.Bool(true),
					Expose:                    pointer.Bool(true),
					DeprecatedLivenessTimeout: 1234,
				},
			},
		},
	}

	env := newTestProbesEnv(t, integration)
	env.Integration.Status.Phase = v1.IntegrationPhaseDeploying

	err := env.Catalog.apply(&env)
	assert.Nil(t, err)

	container := env.GetIntegrationContainer()

	assert.Equal(t, "", container.LivenessProbe.HTTPGet.Host)
	assert.Equal(t, int32(0), container.LivenessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, defaultLivenessProbePath, container.LivenessProbe.HTTPGet.Path)
	assert.Equal(t, "", container.ReadinessProbe.HTTPGet.Host)
	assert.Equal(t, int32(0), container.ReadinessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, defaultReadinessProbePath, container.ReadinessProbe.HTTPGet.Path)
	assert.Equal(t, int32(1234), container.LivenessProbe.TimeoutSeconds)
}

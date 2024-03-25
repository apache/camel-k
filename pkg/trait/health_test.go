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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/test"
)

func TestHealthTrait(t *testing.T) {
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
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
				Traits: v1.Traits{
					Health: &trait.HealthTrait{
						Trait: trait.Trait{
							Enabled: pointer.Bool(true),
						},
						LivenessProbeEnabled:  pointer.Bool(true),
						ReadinessProbeEnabled: pointer.Bool(true),
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
	_, err = traitCatalog.apply(&environment)
	require.NoError(t, err)

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)
	assert.NotNil(t, d)
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.NotNil(t, d.Spec.Template.Spec.Containers[0].LivenessProbe)
	assert.Equal(t, "/q/health/live", d.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Path)
	assert.NotNil(t, d.Spec.Template.Spec.Containers[0].ReadinessProbe)
	assert.Equal(t, "/q/health/ready", d.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Path)
	assert.Nil(t, d.Spec.Template.Spec.Containers[0].StartupProbe)

	// Change traits configuration
	environment.Integration.Spec.Traits.Health.LivenessProbeEnabled = pointer.Bool(false)
	environment.Integration.Spec.Traits.Health.ReadinessProbeEnabled = pointer.Bool(false)
	environment.Integration.Spec.Traits.Health.StartupProbeEnabled = pointer.Bool(true)

	environment.Platform.ResyncStatusFullConfig()
	_, err = traitCatalog.apply(&environment)
	require.NoError(t, err)
	d = environment.Resources.GetDeploymentForIntegration(environment.Integration)
	assert.NotNil(t, d)
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Nil(t, d.Spec.Template.Spec.Containers[0].LivenessProbe)
	assert.Nil(t, d.Spec.Template.Spec.Containers[0].ReadinessProbe)
	assert.NotNil(t, d.Spec.Template.Spec.Containers[0].StartupProbe)
	assert.Equal(t, "/q/health/started", d.Spec.Template.Spec.Containers[0].StartupProbe.HTTPGet.Path)

}

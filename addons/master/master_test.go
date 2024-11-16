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

package master

import (
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestMasterOn(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, err := newFakeClient()
	require.NoError(t, err)
	traitCatalog := trait.NewCatalog(nil)

	environment := trait.Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "Master.java",
							Content: `from("master:lock:timer:tick").to("log:test")`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{},
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
				Profile: v1.TraitProfileKnative,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]trait.Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	mt := NewMasterTrait()
	mt.InjectClient(client)
	// Initialization phase
	configured, conditions, err := mt.Configure(&environment)
	require.NoError(t, err)
	assert.Empty(t, conditions)
	assert.True(t, configured)
	err = mt.Apply(&environment)
	require.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Capabilities, 1)
	assert.Equal(t, "master", environment.Integration.Status.Capabilities[0])
	// Running phase
	environment.Integration.Status.Phase = v1.IntegrationPhaseRunning
	err = mt.Apply(&environment)
	require.NoError(t, err)
	assert.Equal(t, "test-lock", environment.ApplicationProperties["camel.k.master.resourceName"])
	assert.Equal(t, leaseResourceType, environment.ApplicationProperties["camel.k.master.resourceType"])
	assert.Equal(t, "camel.apache.org/integration", environment.ApplicationProperties["camel.k.master.labelKey"])
	assert.Equal(t, "test", environment.ApplicationProperties["camel.k.master.labelValue"])
	assert.Equal(t, "${camel.k.master.resourceName}", environment.ApplicationProperties["quarkus.camel.cluster.kubernetes.resource-name"])
	assert.Equal(t, "${camel.k.master.resourceType}", environment.ApplicationProperties["quarkus.camel.cluster.kubernetes.resource-type"])
	assert.Equal(t, "${camel.k.master.labelValue}", environment.ApplicationProperties["quarkus.camel.cluster.kubernetes.labels.\"camel.apache.org/integration\""])
	roles := 0
	roleBindings := 0
	for _, obj := range environment.Resources.Items() {
		// make sure it contains a Role and a RoleBinding
		if obj.GetObjectKind().GroupVersionKind().Kind == "Role" {
			roles++
		}
		if obj.GetObjectKind().GroupVersionKind().Kind == "RoleBinding" {
			roleBindings++
		}
	}
	assert.Equal(t, 1, roles)
	assert.Equal(t, 1, roleBindings)
}

func TestMasterOff(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, err := newFakeClient()
	require.NoError(t, err)
	traitCatalog := trait.NewCatalog(nil)

	environment := trait.Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "NonMaster.java",
							Content: `from("timer:tick").to("log:test")`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{},
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
				Profile: v1.TraitProfileKnative,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]trait.Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	mt := NewMasterTrait()
	mt.InjectClient(client)
	// Initialization phase
	configured, conditions, err := mt.Configure(&environment)
	require.NoError(t, err)
	assert.Empty(t, conditions)
	assert.False(t, configured)
}

func TestMasterAuto(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, err := newFakeClient()
	require.NoError(t, err)
	traitCatalog := trait.NewCatalog(nil)

	environment := trait.Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "Master.java",
							Content: `from("master:lock:timer:tick").to("log:test")`,
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
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
				Profile: v1.TraitProfileKnative,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]trait.Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	mt := NewMasterTrait()
	mt.InjectClient(client)
	trait, _ := mt.(*masterTrait)
	// Initialization phase
	configured, conditions, err := trait.Configure(&environment)
	require.NoError(t, err)
	assert.Empty(t, conditions)
	assert.True(t, configured)
	err = trait.Apply(&environment)
	require.NoError(t, err)

	expectedTrait := &masterTrait{
		Trait: Trait{
			Trait: traitv1.Trait{
				Enabled: ptr.To(true),
			},
			ResourceName: ptr.To("test-lock"),
			LabelValue:   ptr.To("test"),
		},
	}
	assert.Equal(t, expectedTrait.Trait, trait.Trait)
}

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

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func TestBuilderTraitNotAppliedBecauseOfNilKit(t *testing.T) {
	environments := []*Environment{
		createBuilderTestEnv(v1.IntegrationPlatformClusterOpenShift, v1.IntegrationPlatformBuildPublishStrategyS2I),
		createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyKaniko),
	}

	for _, e := range environments {
		e := e // pin
		e.IntegrationKit = nil

		t.Run(string(e.Platform.Status.Cluster), func(t *testing.T) {
			err := NewBuilderTestCatalog().apply(e)

			assert.Nil(t, err)
			assert.NotEmpty(t, e.ExecutedTraits)
			assert.Nil(t, e.GetTrait("builder"))
			assert.Empty(t, e.BuildTasks)
		})
	}
}

func TestBuilderTraitNotAppliedBecauseOfNilPhase(t *testing.T) {
	environments := []*Environment{
		createBuilderTestEnv(v1.IntegrationPlatformClusterOpenShift, v1.IntegrationPlatformBuildPublishStrategyS2I),
		createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyKaniko),
	}

	for _, e := range environments {
		e := e // pin
		e.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseInitialization

		t.Run(string(e.Platform.Status.Cluster), func(t *testing.T) {
			err := NewBuilderTestCatalog().apply(e)

			assert.Nil(t, err)
			assert.NotEmpty(t, e.ExecutedTraits)
			assert.Nil(t, e.GetTrait("builder"))
			assert.Empty(t, e.BuildTasks)
		})
	}
}

func TestS2IBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterOpenShift, v1.IntegrationPlatformBuildPublishStrategyS2I)
	err := NewBuilderTestCatalog().apply(env)

	assert.Nil(t, err)
	assert.NotEmpty(t, env.ExecutedTraits)
	assert.NotNil(t, env.GetTrait("builder"))
	assert.NotEmpty(t, env.BuildTasks)
	assert.Len(t, env.BuildTasks, 2)
	assert.NotNil(t, env.BuildTasks[0].Builder)
	assert.NotNil(t, env.BuildTasks[1].S2i)
}

func TestKanikoBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyKaniko)
	err := NewBuilderTestCatalog().apply(env)

	assert.Nil(t, err)
	assert.NotEmpty(t, env.ExecutedTraits)
	assert.NotNil(t, env.GetTrait("builder"))
	assert.NotEmpty(t, env.BuildTasks)
	assert.Len(t, env.BuildTasks, 2)
	assert.NotNil(t, env.BuildTasks[0].Builder)
	assert.NotNil(t, env.BuildTasks[1].Kaniko)
}

func createBuilderTestEnv(cluster v1.IntegrationPlatformCluster, strategy v1.IntegrationPlatformBuildPublishStrategy) *Environment {
	c, err := camel.DefaultCatalog()
	if err != nil {
		panic(err)
	}

	kanikoCache := false
	res := &Environment{
		C:            context.TODO(),
		CamelCatalog: c,
		Catalog:      NewCatalog(context.TODO(), nil),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseBuildSubmitted,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: cluster,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy:  strategy,
					Registry:         v1.IntegrationPlatformRegistrySpec{Address: "registry"},
					RuntimeVersion:   defaults.DefaultRuntimeVersion,
					RuntimeProvider:  v1.RuntimeProviderQuarkus,
					KanikoBuildCache: &kanikoCache,
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}

	res.Platform.ResyncStatusFullConfig()

	return res
}

func NewBuilderTestCatalog() *Catalog {
	return NewCatalog(context.TODO(), nil)
}

func TestMavenPropertyBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyKaniko)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Properties = append(builderTrait.Properties, "build-time-prop1=build-time-value1")

	err := builderTrait.Apply(env)

	assert.Nil(t, err)
	assert.Equal(t, "build-time-value1", env.BuildTasks[0].Builder.Maven.Properties["build-time-prop1"])
}

func createNominalBuilderTraitTest() *builderTrait {
	builderTrait := newBuilderTrait().(*builderTrait)
	builderTrait.Enabled = BoolP(true)

	return builderTrait
}

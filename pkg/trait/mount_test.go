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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/gzip"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
)

func TestMountVolumesEmpty(t *testing.T) {
	traitCatalog := NewCatalog(nil)

	environment := getNominalEnv(t, traitCatalog)
	environment.Integration.Spec.Traits = v1.Traits{} // empty traits
	environment.Platform.ResyncStatusFullConfig()

	err := traitCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("mount"))

	s := environment.Resources.GetDeployment(func(service *appsv1.Deployment) bool {
		return service.Name == "hello"
	})
	assert.NotNil(t, s)
	spec := s.Spec.Template.Spec

	assert.Len(t, spec.Containers[0].VolumeMounts, 2)
	assert.Len(t, spec.Volumes, 2)
}

func TestMountVolumesIntegrationPhaseDeploying(t *testing.T) {
	traitCatalog := NewCatalog(nil)

	environment := getNominalEnv(t, traitCatalog)
	environment.Platform.ResyncStatusFullConfig()

	err := traitCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("mount"))

	s := environment.Resources.GetDeployment(func(service *appsv1.Deployment) bool {
		return service.Name == "hello"
	})
	assert.NotNil(t, s)
	spec := s.Spec.Template.Spec

	assert.Len(t, spec.Containers[0].VolumeMounts, 5)
	assert.Len(t, spec.Volumes, 5)

	assert.Condition(t, func() bool {
		for _, v := range spec.Containers[0].VolumeMounts {
			if v.Name == "my-pvc" {
				return true
			}
		}
		return false
	})
	assert.Condition(t, func() bool {
		for _, v := range spec.Containers[0].VolumeMounts {
			if v.Name == "my-cm" {
				return true
			}
		}
		return false
	})
	assert.Condition(t, func() bool {
		for _, v := range spec.Volumes {
			if v.Name == "my-secret" {
				return true
			}
		}
		return false
	})
}

func TestMountVolumesIntegrationPhaseInitialization(t *testing.T) {
	traitCatalog := NewCatalog(nil)

	environment := getNominalEnv(t, traitCatalog)
	environment.Integration.Status.Phase = v1.IntegrationPhaseInitialization
	environment.Platform.ResyncStatusFullConfig()

	err := traitCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.Nil(t, environment.GetTrait("mount"))

	s := environment.Resources.GetDeployment(func(service *appsv1.Deployment) bool {
		return service.Name == "hello"
	})
	assert.Nil(t, s)
}

func getNominalEnv(t *testing.T, traitCatalog *Catalog) *Environment {
	t.Helper()
	fakeClient, _ := test.NewFakeClient()
	catalog, _ := camel.DefaultCatalog()
	compressedRoute, _ := gzip.CompressBase64([]byte(`from("undertow:test").log("hello")`))

	return &Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Ctx:          context.Background(),
		Client:       fakeClient,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hello",
				Namespace: "default",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
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
				Traits: v1.Traits{
					Mount: &traitv1.MountTrait{
						Configs:   []string{"configmap:my-cm"},
						Resources: []string{"secret:my-secret"},
						Volumes:   []string{"my-pvc:/over/the/rainbow"},
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
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
}

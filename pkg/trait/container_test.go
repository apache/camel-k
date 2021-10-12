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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
)

func TestContainerWithDefaults(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
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
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	err = traitCatalog.apply(&environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("container"))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)

	assert.NotNil(t, d)
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, defaultContainerName, d.Spec.Template.Spec.Containers[0].Name)
}

func TestContainerWithCustomName(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
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
				Traits: map[string]v1.TraitSpec{
					"container": test.TraitSpecFromMap(t, map[string]interface{}{
						"name": "my-container-name",
					}),
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
	environment.Platform.ResyncStatusFullConfig()

	err = traitCatalog.apply(&environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployment"))
	assert.NotNil(t, environment.GetTrait("container"))

	d := environment.Resources.GetDeploymentForIntegration(environment.Integration)

	assert.NotNil(t, d)
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)

	trait := test.TraitSpecToMap(t, environment.Integration.Spec.Traits["container"])
	assert.Equal(t, trait["name"], d.Spec.Template.Spec.Containers[0].Name)
}

func TestContainerWithCustomImage(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	client, _ := test.NewFakeClient()
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
				Traits: map[string]v1.TraitSpec{
					"container": test.TraitSpecFromMap(t, map[string]interface{}{
						"image": "foo/bar:1.0.0",
					}),
				},
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
	environment.Platform.ResyncStatusFullConfig()

	err = traitCatalog.apply(&environment)
	assert.Nil(t, err)

	for _, postAction := range environment.PostActions {
		assert.Nil(t, postAction(&environment))
	}

	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("deployer"))
	assert.NotNil(t, environment.GetTrait("container"))
	assert.Equal(t, "kit-"+ServiceTestName, environment.Integration.Status.IntegrationKit.Name)

	ikt := v1.IntegrationKit{}
	key := ctrl.ObjectKey{
		Namespace: "ns",
		Name:      "kit-" + ServiceTestName,
	}

	err = client.Get(context.TODO(), key, &ikt)
	assert.Nil(t, err)
	assert.Equal(t, environment.Integration.ObjectMeta.UID, ikt.ObjectMeta.OwnerReferences[0].UID)

	trait := test.TraitSpecToMap(t, environment.Integration.Spec.Traits["container"])
	assert.Equal(t, trait["image"], ikt.Spec.Image)
}

func TestContainerWithCustomImageAndIntegrationKit(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	client, _ := test.NewFakeClient()
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
				Traits: map[string]v1.TraitSpec{
					"container": test.TraitSpecFromMap(t, map[string]interface{}{
						"image": "foo/bar:1.0.0",
					}),
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
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	err = traitCatalog.apply(&environment)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unsupported configuration: a container image has been set in conjunction with an IntegrationKit")
}

func TestContainerWithCustomImageAndDeprecatedIntegrationKit(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	client, _ := test.NewFakeClient()
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
				Traits: map[string]v1.TraitSpec{
					"container": test.TraitSpecFromMap(t, map[string]interface{}{
						"image": "foo/bar:1.0.0",
					}),
				},
				Kit: "bad-" + ServiceTestName,
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
	environment.Platform.ResyncStatusFullConfig()

	err = traitCatalog.apply(&environment)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unsupported configuration: a container image has been set in conjunction with an IntegrationKit")
}

func TestContainerWithImagePullPolicy(t *testing.T) {
	target := appsv1.Deployment{}

	env := newTestProbesEnv(t, v1.RuntimeProviderQuarkus)
	env.Integration.Status.Phase = v1.IntegrationPhaseDeploying
	env.Resources.Add(&target)

	ctr := newTestContainerTrait()
	ctr.ImagePullPolicy = "Always"

	err := ctr.Apply(&env)
	assert.Nil(t, err)
	assert.Equal(t, corev1.PullAlways, target.Spec.Template.Spec.Containers[0].ImagePullPolicy)

	ctr.ImagePullPolicy = "MustFail"

	ok, err := ctr.Configure(&env)
	assert.False(t, ok)
	assert.NotNil(t, err)
}

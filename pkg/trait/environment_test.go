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

	"github.com/scylladb/go-set/strset"
	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func TestDefaultEnvironment(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	env := Environment{
		CamelCatalog: catalog,
		Catalog:      NewCatalog(context.TODO(), nil),
		Integration: &v1alpha1.Integration{
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
			Spec: v1alpha1.IntegrationSpec{
				Profile: v1alpha1.TraitProfileOpenShift,
			},
		},
		IntegrationKit: &v1alpha1.IntegrationKit{
			Status: v1alpha1.IntegrationKitStatus{
				Phase: v1alpha1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
			},
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: v1alpha1.IntegrationPlatformClusterOpenShift,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
		Classpath:      strset.New(),
	}
	env.Platform.ResyncStatusFullConfig()

	err = NewEnvironmentTestCatalog().apply(&env)

	assert.Nil(t, err)

	ns := false
	name := false
	ck := false

	env.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
		for _, e := range deployment.Spec.Template.Spec.Containers[0].Env {
			if e.Name == envVarNamespace {
				ns = true
			}
			if e.Name == envVarPodName {
				name = true
			}
			if e.Name == envVarCamelKVersion {
				ck = true
			}
		}
	})

	assert.True(t, ns)
	assert.True(t, name)
	assert.True(t, ck)
}

func TestEnabledContainerMetaDataEnvVars(t *testing.T) {
	c, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	env := Environment{
		CamelCatalog: c,
		Catalog:      NewCatalog(context.TODO(), nil),
		Integration: &v1alpha1.Integration{
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
			Spec: v1alpha1.IntegrationSpec{
				Profile: v1alpha1.TraitProfileOpenShift,
				Traits: map[string]v1alpha1.TraitSpec{
					"environment": {
						Configuration: map[string]string{
							"container-meta": "true",
						},
					},
				},
			},
		},
		IntegrationKit: &v1alpha1.IntegrationKit{
			Status: v1alpha1.IntegrationKitStatus{
				Phase: v1alpha1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
			},
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: v1alpha1.IntegrationPlatformClusterOpenShift,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
		Classpath:      strset.New(),
	}
	env.Platform.ResyncStatusFullConfig()

	err = NewEnvironmentTestCatalog().apply(&env)

	assert.Nil(t, err)

	ns := false
	name := false
	ck := false

	env.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
		for _, e := range deployment.Spec.Template.Spec.Containers[0].Env {
			if e.Name == envVarNamespace {
				ns = true
			}
			if e.Name == envVarPodName {
				name = true
			}
			if e.Name == envVarCamelKVersion {
				ck = true
			}
		}
	})

	assert.True(t, ns)
	assert.True(t, name)
	assert.True(t, ck)
}

func TestDisabledContainerMetaDataEnvVars(t *testing.T) {
	c, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	env := Environment{
		CamelCatalog: c,
		Catalog:      NewCatalog(context.TODO(), nil),
		Integration: &v1alpha1.Integration{
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
			Spec: v1alpha1.IntegrationSpec{
				Profile: v1alpha1.TraitProfileOpenShift,
				Traits: map[string]v1alpha1.TraitSpec{
					"environment": {
						Configuration: map[string]string{
							"container-meta": "false",
						},
					},
				},
			},
		},
		IntegrationKit: &v1alpha1.IntegrationKit{
			Status: v1alpha1.IntegrationKitStatus{
				Phase: v1alpha1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
			},
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: v1alpha1.IntegrationPlatformClusterOpenShift,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
		Classpath:      strset.New(),
	}
	env.Platform.ResyncStatusFullConfig()

	err = NewEnvironmentTestCatalog().apply(&env)

	assert.Nil(t, err)

	ns := false
	name := false
	ck := false

	env.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
		for _, e := range deployment.Spec.Template.Spec.Containers[0].Env {
			if e.Name == envVarNamespace {
				ns = true
			}
			if e.Name == envVarPodName {
				name = true
			}
			if e.Name == envVarCamelKVersion {
				ck = true
			}
		}
	})

	assert.False(t, ns)
	assert.False(t, name)
	assert.True(t, ck)
}

func NewEnvironmentTestCatalog() *Catalog {
	return NewCatalog(context.TODO(), nil)
}

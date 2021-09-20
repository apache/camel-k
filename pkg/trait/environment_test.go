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
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
)

func TestDefaultEnvironment(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	env := mockEnvironment(catalog)
	env.Platform.ResyncStatusFullConfig()

	err = NewEnvironmentTestCatalog().apply(&env)

	assert.Nil(t, err)

	ns := false
	name := false
	ck := false
	cms := false
	secrets := false

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
			if e.Name == envVarMountPathConfigMaps {
				cms = true
			}
			if e.Name == envVarMountPathSecrets {
				secrets = true
			}
		}
	})

	assert.True(t, ns)
	assert.True(t, name)
	assert.True(t, ck)
	assert.True(t, cms)
	assert.True(t, secrets)
}

func TestEnabledContainerMetaDataEnvVars(t *testing.T) {
	c, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	env := mockEnvironment(c)
	env.Integration.Spec.Traits = map[string]v1.TraitSpec{
		"environment": test.TraitSpecFromMap(t, map[string]interface{}{
			"containerMeta": true,
		}),
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

	env := mockEnvironment(c)
	env.Integration.Spec.Traits = map[string]v1.TraitSpec{
		"environment": test.TraitSpecFromMap(t, map[string]interface{}{
			"containerMeta": false,
		}),
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

func TestCustomEnvVars(t *testing.T) {
	c, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	env := mockEnvironment(c)
	env.Integration.Spec.Traits = map[string]v1.TraitSpec{
		"environment": test.TraitSpecFromMap(t, map[string]interface{}{
			"vars": []string{"key1=val1", "key2 = val2"},
		}),
	}
	env.Platform.ResyncStatusFullConfig()

	err = NewEnvironmentTestCatalog().apply(&env)

	assert.Nil(t, err)

	userK1 := false
	userK2 := false

	env.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
		for _, e := range deployment.Spec.Template.Spec.Containers[0].Env {
			if e.Name == "key1" {
				userK1 = e.Value == "val1"
			}
			if e.Name == "key2" {
				userK2 = e.Value == "val2"
			}
		}
	})

	assert.True(t, userK1)
	assert.True(t, userK2)
}

func NewEnvironmentTestCatalog() *Catalog {
	return NewCatalog(context.TODO(), nil)
}

func mockEnvironment(catalog *camel.RuntimeCatalog) Environment {
	return Environment{
		CamelCatalog: catalog,
		Catalog:      NewCatalog(context.TODO(), nil),
		Integration: &v1.Integration{
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileOpenShift,
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
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
}

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

	"github.com/apache/camel-k/pkg/util/test"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestDeployment = "test"
	TestProperties = "test-properties"
)

func TestOpenShiftTraits(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterOpenShift, "camel:core")
	res := processTestEnv(t, env)

	assert.NotEmpty(t, env.ExecutedTraits)
	assert.NotNil(t, env.GetTrait(ID("deployment")))
	assert.Nil(t, env.GetTrait(ID("service")))
	assert.Nil(t, env.GetTrait(ID("route")))
	assert.NotNil(t, env.GetTrait(ID("owner")))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == TestProperties
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == TestDeployment
	}))
}

func TestOpenShiftTraitsWithWeb(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterOpenShift, "from('undertow:http').to('log:info')")
	res := processTestEnv(t, env)
	assert.NotNil(t, env.GetTrait(ID("deployment")))
	assert.NotNil(t, env.GetTrait(ID("service")))
	assert.NotNil(t, env.GetTrait(ID("route")))
	assert.NotNil(t, env.GetTrait(ID("owner")))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == TestProperties
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == TestDeployment
	}))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == TestDeployment
	}))
	assert.NotNil(t, res.GetRoute(func(svc *routev1.Route) bool {
		return svc.Name == TestDeployment
	}))
}

func TestOpenShiftTraitsWithWebAndConfig(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterOpenShift, "from('undertow:http').to('log:info')")
	env.Integration.Spec.Traits = make(map[string]v1alpha1.IntegrationTraitSpec)
	env.Integration.Spec.Traits["service"] = v1alpha1.IntegrationTraitSpec{
		Configuration: map[string]string{
			"port": "7071",
		},
	}
	res := processTestEnv(t, env)
	assert.NotNil(t, env.GetTrait(ID("service")))
	assert.NotNil(t, env.GetTrait(ID("route")))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == TestDeployment && svc.Spec.Ports[0].TargetPort.StrVal == "http"
	}))
}

func TestOpenShiftTraitsWithWebAndDisabledTrait(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterOpenShift, "from('undertow:http').to('log:info')")
	env.Integration.Spec.Traits = make(map[string]v1alpha1.IntegrationTraitSpec)
	env.Integration.Spec.Traits["service"] = v1alpha1.IntegrationTraitSpec{
		Configuration: map[string]string{
			"enabled": "false",
			"port":    "7071",
		},
	}
	res := processTestEnv(t, env)
	assert.Nil(t, env.GetTrait(ID("service")))
	assert.Nil(t, env.GetTrait(ID("route"))) // No route without service
	assert.Nil(t, res.GetService(func(svc *corev1.Service) bool {
		return true
	}))
}

func TestKubernetesTraits(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterKubernetes, "from('timer:tick').to('log:info')")
	res := processTestEnv(t, env)
	assert.NotNil(t, env.GetTrait(ID("deployment")))
	assert.Nil(t, env.GetTrait(ID("service")))
	assert.Nil(t, env.GetTrait(ID("route")))
	assert.NotNil(t, env.GetTrait(ID("owner")))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == TestProperties
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == TestDeployment
	}))
}

func TestKubernetesTraitsWithWeb(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterKubernetes, "from('servlet:http').to('log:info')")
	res := processTestEnv(t, env)
	assert.NotNil(t, env.GetTrait(ID("deployment")))
	assert.NotNil(t, env.GetTrait(ID("service")))
	assert.Nil(t, env.GetTrait(ID("route")))
	assert.NotNil(t, env.GetTrait(ID("owner")))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == TestProperties
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == TestDeployment
	}))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == TestDeployment
	}))
}

func TestTraitDecode(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterOpenShift, "")
	env.Integration.Spec.Traits = make(map[string]v1alpha1.IntegrationTraitSpec)
	svcTrait := v1alpha1.IntegrationTraitSpec{
		Configuration: map[string]string{
			"enabled": "false",
			"port":    "7071",
			"cippa":   "lippa",
		},
	}
	env.Integration.Spec.Traits["service"] = svcTrait

	svc := newServiceTrait()
	err := svcTrait.Decode(svc)

	assert.Nil(t, err)
	assert.Equal(t, 7071, svc.Port)
	assert.NotNil(t, svc.Enabled)
	assert.Equal(t, false, *svc.Enabled)
}

func processTestEnv(t *testing.T, env *Environment) *kubernetes.Collection {
	catalog := NewTraitTestCatalog()
	err := catalog.apply(env)
	assert.Nil(t, err)
	return env.Resources
}

func createTestEnv(t *testing.T, cluster v1alpha1.IntegrationPlatformCluster, script string) *Environment {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	return &Environment{
		CamelCatalog: catalog,
		Integration: &v1alpha1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      TestDeployment,
				Namespace: "ns",
			},
			Spec: v1alpha1.IntegrationSpec{
				Sources: []v1alpha1.SourceSpec{
					{
						DataSpec: v1alpha1.DataSpec{
							Name:    "file.groovy",
							Content: script,
						},
						Language: v1alpha1.LanguageGroovy,
					},
				},
			},
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
		},
		Context: &v1alpha1.IntegrationContext{},
		Platform: &v1alpha1.IntegrationPlatform{
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: cluster,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
}

func NewTraitTestCatalog() *Catalog {
	return NewCatalog(context.TODO(), nil)
}

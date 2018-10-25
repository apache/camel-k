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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOpenShiftTraits(t *testing.T) {
	env := createTestEnv(v1alpha1.IntegrationPlatformClusterOpenShift, "camel:core")
	res := processTestEnv(t, env)
	assert.NotEmpty(t, env.ExecutedTraits)
	assert.Contains(t, env.ExecutedTraits, ID("deployment"))
	assert.NotContains(t, env.ExecutedTraits, ID("service"))
	assert.NotContains(t, env.ExecutedTraits, ID("route"))
	assert.Contains(t, env.ExecutedTraits, ID("owner"))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == "test"
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == "test"
	}))
}

func TestOpenShiftTraitsWithWeb(t *testing.T) {
	env := createTestEnv(v1alpha1.IntegrationPlatformClusterOpenShift, "camel:core", "camel:undertow")
	res := processTestEnv(t, env)
	assert.Contains(t, env.ExecutedTraits, ID("deployment"))
	assert.Contains(t, env.ExecutedTraits, ID("service"))
	assert.Contains(t, env.ExecutedTraits, ID("route"))
	assert.Contains(t, env.ExecutedTraits, ID("owner"))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == "test"
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == "test"
	}))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == "test"
	}))
	assert.NotNil(t, res.GetRoute(func(svc *routev1.Route) bool {
		return svc.Name == "test"
	}))
}

func TestOpenShiftTraitsWithWebAndConfig(t *testing.T) {
	env := createTestEnv(v1alpha1.IntegrationPlatformClusterOpenShift, "camel:core", "camel:undertow")
	env.Integration.Spec.Traits = make(map[string]v1alpha1.IntegrationTraitSpec)
	env.Integration.Spec.Traits["service"] = v1alpha1.IntegrationTraitSpec{
		Configuration: map[string]string{
			"port": "7071",
		},
	}
	res := processTestEnv(t, env)
	assert.Contains(t, env.ExecutedTraits, ID("service"))
	assert.Contains(t, env.ExecutedTraits, ID("route"))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == "test" && svc.Spec.Ports[0].TargetPort.IntVal == int32(7071)
	}))
}

func TestOpenShiftTraitsWithWebAndDisabledTrait(t *testing.T) {
	env := createTestEnv(v1alpha1.IntegrationPlatformClusterOpenShift, "camel:core", "camel:undertow")
	env.Integration.Spec.Traits = make(map[string]v1alpha1.IntegrationTraitSpec)
	env.Integration.Spec.Traits["service"] = v1alpha1.IntegrationTraitSpec{
		Configuration: map[string]string{
			"enabled": "false",
			"port":    "7071",
		},
	}
	res := processTestEnv(t, env)
	assert.NotContains(t, env.ExecutedTraits, ID("service"))
	assert.NotContains(t, env.ExecutedTraits, ID("route")) // No route without service
	assert.Nil(t, res.GetService(func(svc *corev1.Service) bool {
		return true
	}))
}

func TestKubernetesTraits(t *testing.T) {
	env := createTestEnv(v1alpha1.IntegrationPlatformClusterKubernetes, "camel:core")
	res := processTestEnv(t, env)
	assert.Contains(t, env.ExecutedTraits, ID("deployment"))
	assert.NotContains(t, env.ExecutedTraits, ID("service"))
	assert.NotContains(t, env.ExecutedTraits, ID("route"))
	assert.Contains(t, env.ExecutedTraits, ID("owner"))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == "test"
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == "test"
	}))
}

func TestKubernetesTraitsWithWeb(t *testing.T) {
	env := createTestEnv(v1alpha1.IntegrationPlatformClusterKubernetes, "camel:core", "camel:servlet")
	res := processTestEnv(t, env)
	assert.Contains(t, env.ExecutedTraits, ID("deployment"))
	assert.Contains(t, env.ExecutedTraits, ID("service"))
	assert.NotContains(t, env.ExecutedTraits, ID("route"))
	assert.Contains(t, env.ExecutedTraits, ID("owner"))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == "test"
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == "test"
	}))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == "test"
	}))
}

func TestTraitDecode(t *testing.T) {
	env := createTestEnv(v1alpha1.IntegrationPlatformClusterOpenShift)
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
	assert.Equal(t, false, svc.IsEnabled())
}

func processTestEnv(t *testing.T, env *environment) *kubernetes.Collection {
	resources := kubernetes.NewCollection()
	catalog := NewCatalog()
	err := catalog.customize(env, resources)
	assert.Nil(t, err)
	return resources
}

func createTestEnv(cluster v1alpha1.IntegrationPlatformCluster, dependencies ...string) *environment {
	return &environment{
		Integration: &v1alpha1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Spec: v1alpha1.IntegrationSpec{
				Dependencies: dependencies,
			},
		},
		Context: &v1alpha1.IntegrationContext{},
		Platform: &v1alpha1.IntegrationPlatform{
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: cluster,
			},
		},
		ExecutedTraits: make([]ID, 0),
	}
}

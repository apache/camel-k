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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigureHealthTraitDoesSucceed(t *testing.T) {
	ht, environment := createNominalHealthTrait(t)
	configured, condition, err := ht.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
	assert.Nil(t, condition)
}

func TestConfigureHealthTraitDisabled(t *testing.T) {
	enabled := false
	ht, environment := createNominalHealthTrait(t)
	ht.Enabled = &enabled
	configured, condition, err := ht.Configure(environment)

	assert.False(t, configured)
	assert.Nil(t, err)
	assert.Nil(t, condition)
}

func TestApplyHealthTraitDefault(t *testing.T) {
	ht, environment := createNominalHealthTrait(t)
	configured, condition, err := ht.Configure(environment)
	assert.True(t, configured)
	assert.Nil(t, err)
	assert.Nil(t, condition)

	err = ht.Apply(environment)
	assert.Nil(t, err)
	assert.Equal(t, "/q/health/ready", environment.GetIntegrationContainer().ReadinessProbe.HTTPGet.Path)
	assert.Equal(t, corev1.URISchemeHTTP, environment.GetIntegrationContainer().ReadinessProbe.HTTPGet.Scheme)
	assert.Equal(t, "8080", environment.GetIntegrationContainer().ReadinessProbe.HTTPGet.Port.String())
}
func TestApplyHealthTraitLivenessDefault(t *testing.T) {
	enabled := true
	ht, environment := createNominalHealthTrait(t)
	ht.LivenessProbeEnabled = &enabled
	configured, condition, err := ht.Configure(environment)
	assert.True(t, configured)
	assert.Nil(t, err)
	assert.Nil(t, condition)

	err = ht.Apply(environment)
	assert.Nil(t, err)
	assert.Equal(t, "/q/health/live", environment.GetIntegrationContainer().LivenessProbe.HTTPGet.Path)
	assert.Equal(t, corev1.URISchemeHTTP, environment.GetIntegrationContainer().LivenessProbe.HTTPGet.Scheme)
	assert.Equal(t, "8080", environment.GetIntegrationContainer().LivenessProbe.HTTPGet.Port.String())
}

func TestApplyHealthTraitStartupDefault(t *testing.T) {
	enabled := true
	ht, environment := createNominalHealthTrait(t)
	ht.StartupProbeEnabled = &enabled
	configured, condition, err := ht.Configure(environment)
	assert.True(t, configured)
	assert.Nil(t, err)
	assert.Nil(t, condition)

	err = ht.Apply(environment)
	assert.Nil(t, err)
	assert.Equal(t, "/q/health/started", environment.GetIntegrationContainer().StartupProbe.HTTPGet.Path)
	assert.Equal(t, corev1.URISchemeHTTP, environment.GetIntegrationContainer().StartupProbe.HTTPGet.Scheme)
	assert.Equal(t, "8080", environment.GetIntegrationContainer().StartupProbe.HTTPGet.Port.String())
}

func createNominalHealthTrait(t *testing.T) (*healthTrait, *Environment) {
	t.Helper()
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)
	trait, _ := newHealthTrait().(*healthTrait)

	environment := &Environment{
		CamelCatalog: catalog,
		Catalog:      NewCatalog(nil),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "integration-name",
				Generation: 1,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseRunning,
			},
		},
		Resources: kubernetes.NewCollection(),
	}

	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "integration"},
					},
				},
			},
		},
	}
	environment.Resources.Add(&deployment)

	return trait, environment
}

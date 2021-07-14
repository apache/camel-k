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
	"strings"
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"
)

func TestPullSecret(t *testing.T) {
	e, deployment := getEnvironmentAndDeployment()

	trait := newPullSecretTrait().(*pullSecretTrait)
	trait.SecretName = "xxxy"
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)

	err = trait.Apply(e)
	assert.Nil(t, err)
	assert.Contains(t, deployment.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: "xxxy"})
}

func TestPullSecretDoesNothingWhenNotSetOnPlatform(t *testing.T) {
	e, _ := getEnvironmentAndDeployment()
	e.Platform = &v1.IntegrationPlatform{}

	trait := newPullSecretTrait()
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)
}

func TestPullSecretAuto(t *testing.T) {
	e, _ := getEnvironmentAndDeployment()

	trait := newPullSecretTrait().(*pullSecretTrait)
	trait.Auto = newFalse()
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)
}

func TestPullSecretImagePullerDelegation(t *testing.T) {
	e, _ := getEnvironmentAndDeployment()

	trait := newPullSecretTrait().(*pullSecretTrait)
	trait.Auto = newFalse()
	trait.ImagePullerDelegation = newTrue()
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)
	assert.True(t, *trait.ImagePullerDelegation)

	err = trait.Apply(e)
	assert.Nil(t, err)

	found := false
	for _, item := range e.Resources.Items() {
		found = strings.HasPrefix(item.GetName(), "camel-k-puller")
		if found {
			break
		}
	}
	assert.True(t, found)
}

func getEnvironmentAndDeployment() (*Environment, *appsv1.Deployment) {
	e := &Environment{}
	e.Integration = &v1.Integration{
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseDeploying,
		},
	}

	deployment := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{},
			},
		},
	}
	e.Resources = kubernetes.NewCollection(&deployment)

	return e, &deployment
}

func newFalse() *bool {
	b := false
	return &b
}

func newTrue() *bool {
	b := true
	return &b
}

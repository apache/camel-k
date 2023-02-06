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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stretchr/testify/assert"
)

func TestPullSecret(t *testing.T) {
	e, deployment := getEnvironmentAndDeployment(t)

	trait, _ := newPullSecretTrait().(*pullSecretTrait)
	trait.SecretName = "xxxy"
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)

	err = trait.Apply(e)
	assert.Nil(t, err)
	assert.Contains(t, deployment.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: "xxxy"})
}

func TestPullSecretDoesNothingWhenNotSetOnPlatform(t *testing.T) {
	e, _ := getEnvironmentAndDeployment(t)
	e.Platform = &v1.IntegrationPlatform{}

	trait := newPullSecretTrait()
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)
}

func TestPullSecretAuto(t *testing.T) {
	e, _ := getEnvironmentAndDeployment(t)

	trait, _ := newPullSecretTrait().(*pullSecretTrait)
	trait.Auto = pointer.Bool(false)
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)
}

func TestPullSecretImagePullerDelegation(t *testing.T) {
	e, _ := getEnvironmentAndDeployment(t)

	trait, _ := newPullSecretTrait().(*pullSecretTrait)
	trait.Auto = pointer.Bool(false)
	trait.ImagePullerDelegation = pointer.Bool(true)
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)
	assert.True(t, *trait.ImagePullerDelegation)

	err = trait.Apply(e)
	assert.NoError(t, err)

	var roleBinding rbacv1.RoleBinding
	roleBindingKey := client.ObjectKey{
		Namespace: "test",
		Name:      "camel-k-puller-test-default",
	}
	err = e.Client.Get(e.Ctx, roleBindingKey, &roleBinding)
	assert.NoError(t, err)
	assert.Len(t, roleBinding.Subjects, 1)
}

func getEnvironmentAndDeployment(t *testing.T) (*Environment, *appsv1.Deployment) {
	t.Helper()

	e := &Environment{}
	e.Integration = &v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "myit",
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseDeploying,
		},
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "myit",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{},
			},
		},
	}
	e.Resources = kubernetes.NewCollection(&deployment)

	var err error
	e.Ctx = context.TODO()
	e.Client, err = test.NewFakeClient(e.Integration, &deployment)
	assert.NoError(t, err)

	return e, &deployment
}

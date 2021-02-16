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

	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func TestConfigureTolerationTraitDoesSucceed(t *testing.T) {
	tolerationTrait, environment, _ := createNominalTolerationTest()
	configured, err := tolerationTrait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
}

func TestApplyPodTolerationLabelsDoesSucceed(t *testing.T) {
	tolerationTrait, environment, deployment := createNominalTolerationTest()
	tolerationTrait.Toleration = util.BoolP(true)
	tolerationTrait.Key = "my-toleration"
	tolerationTrait.Operator = "Equal"
	tolerationTrait.Value = "my-value"
	tolerationTrait.Effect = "NoExecute"
	tolerationTrait.TolerationSeconds = "300"

	err := tolerationTrait.Apply(environment)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(deployment.Spec.Template.Spec.Tolerations))
	toleration := deployment.Spec.Template.Spec.Tolerations[0]
	assert.Equal(t, "my-toleration", toleration.Key)
	assert.Equal(t, corev1.TolerationOpEqual, toleration.Operator)
	assert.Equal(t, "my-value", toleration.Value)
	assert.Equal(t, corev1.TaintEffectNoExecute, toleration.Effect)
	assert.Equal(t, int64(300), *toleration.TolerationSeconds)
}

func TestConfigureTolerationTraitMissingValue(t *testing.T) {
	tolerationTrait, environment, _ := createNominalTolerationTest()
	tolerationTrait.Toleration = util.BoolP(true)
	tolerationTrait.Key = "my-toleration"
	tolerationTrait.Operator = "Equal"

	success, err := tolerationTrait.Configure(environment)

	assert.Equal(t, false, success)
	assert.NotNil(t, err)
}

func createNominalTolerationTest() (*tolerationTrait, *Environment, *appsv1.Deployment) {
	trait := newTolerationTrait().(*tolerationTrait)
	enabled := true
	trait.Enabled = &enabled

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "integration-name",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{},
		},
	}

	environment := &Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "integration-name",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		Resources: kubernetes.NewCollection(deployment),
	}

	return trait, environment, deployment
}

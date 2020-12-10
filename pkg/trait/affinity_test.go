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

func TestConfigureAffinityTraitDoesSucceed(t *testing.T) {
	affinityTrait, environment, _ := createNominalAffinityTest()
	configured, err := affinityTrait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
}

func TestConfigureAffinityTraitWithConflictingAffinitiesFails(t *testing.T) {
	affinityTrait, environment, _ := createNominalAffinityTest()
	affinityTrait.PodAffinity = util.BoolP(true)
	affinityTrait.PodAntiAffinity = util.BoolP(true)
	configured, err := affinityTrait.Configure(environment)

	assert.False(t, configured)
	assert.NotNil(t, err)
}

func TestConfigureDisabledAffinityTraitFails(t *testing.T) {
	affinityTrait, environment, _ := createNominalAffinityTest()
	affinityTrait.Enabled = new(bool)
	configured, err := affinityTrait.Configure(environment)

	assert.False(t, configured)
	assert.Nil(t, err)
}

func TestApplyEmptyAffinityLabelsDoesSucceed(t *testing.T) {
	affinityTrait, environment, _ := createNominalAffinityTest()

	err := affinityTrait.Apply(environment)

	assert.Nil(t, err)
}

func TestApplyNodeAffinityLabelsDoesSucceed(t *testing.T) {
	affinityTrait, environment, deployment := createNominalAffinityTest()
	affinityTrait.NodeAffinityLabels = []string{"criteria = value"}

	err := affinityTrait.Apply(environment)

	assert.Nil(t, err)
	assert.NotNil(t, deployment.Spec.Template.Spec.Affinity.NodeAffinity)
	nodeAffinity := deployment.Spec.Template.Spec.Affinity.NodeAffinity
	assert.NotNil(t, nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0])
	nodeSelectorRequirement := nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
	assert.Equal(t, "criteria", nodeSelectorRequirement.Key)
	assert.Equal(t, corev1.NodeSelectorOpIn, nodeSelectorRequirement.Operator)
	assert.ElementsMatch(t, [1]string{"value"}, nodeSelectorRequirement.Values)
}

func TestApplyPodAntiAffinityLabelsDoesSucceed(t *testing.T) {
	affinityTrait, environment, deployment := createNominalAffinityTest()
	affinityTrait.PodAntiAffinity = util.BoolP(true)
	affinityTrait.PodAntiAffinityLabels = []string{"criteria != value"}

	err := affinityTrait.Apply(environment)

	assert.Nil(t, err)
	assert.NotNil(t, deployment.Spec.Template.Spec.Affinity.PodAntiAffinity)
	podAntiAffinity := deployment.Spec.Template.Spec.Affinity.PodAntiAffinity
	assert.NotNil(t, podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0])
	userRequirement := podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0]
	assert.Equal(t, "criteria", userRequirement.Key)
	assert.Equal(t, metav1.LabelSelectorOpNotIn, userRequirement.Operator)
	assert.ElementsMatch(t, [1]string{"value"}, userRequirement.Values)
	assert.NotNil(t, podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[1])
	integrationRequirement := podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[1]
	assert.Equal(t, v1.IntegrationLabel, integrationRequirement.Key)
	assert.Equal(t, metav1.LabelSelectorOpIn, integrationRequirement.Operator)
	assert.ElementsMatch(t, [1]string{"integration-name"}, integrationRequirement.Values)
}

func TestApplyPodAffinityLabelsDoesSucceed(t *testing.T) {
	affinityTrait, environment, deployment := createNominalAffinityTest()
	affinityTrait.PodAffinity = util.BoolP(true)
	affinityTrait.PodAffinityLabels = []string{"!criteria"}

	err := affinityTrait.Apply(environment)

	assert.Nil(t, err)
	assert.NotNil(t, deployment.Spec.Template.Spec.Affinity.PodAffinity)
	podAffinity := deployment.Spec.Template.Spec.Affinity.PodAffinity
	assert.NotNil(t, podAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0])
	userRequirement := podAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0]
	assert.Equal(t, "criteria", userRequirement.Key)
	assert.Equal(t, metav1.LabelSelectorOpDoesNotExist, userRequirement.Operator)
	assert.NotNil(t, podAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[1])
	integrationRequirement := podAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[1]
	assert.Equal(t, v1.IntegrationLabel, integrationRequirement.Key)
	assert.Equal(t, metav1.LabelSelectorOpIn, integrationRequirement.Operator)
	assert.ElementsMatch(t, [1]string{"integration-name"}, integrationRequirement.Values)
}

func createNominalAffinityTest() (*affinityTrait, *Environment, *appsv1.Deployment) {
	trait := newAffinityTrait().(*affinityTrait)
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

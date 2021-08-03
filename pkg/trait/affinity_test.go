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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestConfigureAffinityTraitDoesSucceed(t *testing.T) {
	affinityTrait := createNominalAffinityTest()
	environment, _ := createNominalDeploymentTraitTest()
	configured, err := affinityTrait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
}

func TestConfigureAffinityTraitWithConflictingAffinitiesFails(t *testing.T) {
	affinityTrait := createNominalAffinityTest()
	environment, _ := createNominalDeploymentTraitTest()
	affinityTrait.PodAffinity = BoolP(true)
	affinityTrait.PodAntiAffinity = BoolP(true)
	configured, err := affinityTrait.Configure(environment)

	assert.False(t, configured)
	assert.NotNil(t, err)
}

func TestConfigureDisabledAffinityTraitFails(t *testing.T) {
	affinityTrait := createNominalAffinityTest()
	affinityTrait.Enabled = new(bool)
	environment, _ := createNominalDeploymentTraitTest()
	configured, err := affinityTrait.Configure(environment)

	assert.False(t, configured)
	assert.Nil(t, err)
}

func TestApplyAffinityMissingDeployment(t *testing.T) {
	tolerationTrait := createNominalAffinityTest()

	environment := createNominalMissingDeploymentTraitTest()
	err := tolerationTrait.Apply(environment)

	assert.NotNil(t, err)
}

func TestApplyEmptyAffinityLabelsDoesSucceed(t *testing.T) {
	affinityTrait := createNominalAffinityTest()

	environment, deployment := createNominalDeploymentTraitTest()
	testApplyEmptyAffinityLabelsDoesSucceed(t, affinityTrait, environment, deployment.Spec.Template.Spec.Affinity)

	environment, knativeService := createNominalKnativeServiceTraitTest()
	testApplyEmptyAffinityLabelsDoesSucceed(t, affinityTrait, environment, knativeService.Spec.Template.Spec.Affinity)

	environment, cronJob := createNominalCronJobTraitTest()
	testApplyEmptyAffinityLabelsDoesSucceed(t, affinityTrait, environment, cronJob.Spec.JobTemplate.Spec.Template.Spec.Affinity)
}

func testApplyEmptyAffinityLabelsDoesSucceed(t *testing.T, trait *affinityTrait, environment *Environment, affinity *corev1.Affinity) {
	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.Nil(t, affinity)
}

func TestApplyNodeAffinityLabelsDoesSucceed(t *testing.T) {
	affinityTrait := createNominalAffinityTest()
	affinityTrait.NodeAffinityLabels = []string{"criteria = value"}

	environment, deployment := createNominalDeploymentTraitTest()
	testApplyNodeAffinityLabelsDoesSucceed(t, affinityTrait, environment, &deployment.Spec.Template.Spec)

	environment, knativeService := createNominalKnativeServiceTraitTest()
	testApplyNodeAffinityLabelsDoesSucceed(t, affinityTrait, environment, &knativeService.Spec.Template.Spec.PodSpec)

	environment, cronJob := createNominalCronJobTraitTest()
	testApplyNodeAffinityLabelsDoesSucceed(t, affinityTrait, environment, &cronJob.Spec.JobTemplate.Spec.Template.Spec)
}

func testApplyNodeAffinityLabelsDoesSucceed(t *testing.T, trait *affinityTrait, environment *Environment, podSpec *corev1.PodSpec) {
	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.NotNil(t, podSpec.Affinity.NodeAffinity)
	nodeAffinity := podSpec.Affinity.NodeAffinity
	assert.NotNil(t, nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0])
	nodeSelectorRequirement := nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
	assert.Equal(t, "criteria", nodeSelectorRequirement.Key)
	assert.Equal(t, corev1.NodeSelectorOpIn, nodeSelectorRequirement.Operator)
	assert.ElementsMatch(t, [1]string{"value"}, nodeSelectorRequirement.Values)
}

func TestApplyPodAntiAffinityLabelsDoesSucceed(t *testing.T) {
	affinityTrait := createNominalAffinityTest()
	affinityTrait.PodAntiAffinity = BoolP(true)
	affinityTrait.PodAntiAffinityLabels = []string{"criteria != value"}

	environment, deployment := createNominalDeploymentTraitTest()
	testApplyPodAntiAffinityLabelsDoesSucceed(t, affinityTrait, environment, &deployment.Spec.Template.Spec)

	environment, knativeService := createNominalKnativeServiceTraitTest()
	testApplyPodAntiAffinityLabelsDoesSucceed(t, affinityTrait, environment, &knativeService.Spec.Template.Spec.PodSpec)

	environment, cronJob := createNominalCronJobTraitTest()
	testApplyPodAntiAffinityLabelsDoesSucceed(t, affinityTrait, environment, &cronJob.Spec.JobTemplate.Spec.Template.Spec)
}

func testApplyPodAntiAffinityLabelsDoesSucceed(t *testing.T, trait *affinityTrait, environment *Environment, podSpec *corev1.PodSpec) {
	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.NotNil(t, podSpec.Affinity.PodAntiAffinity)
	podAntiAffinity := podSpec.Affinity.PodAntiAffinity
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
	affinityTrait := createNominalAffinityTest()
	affinityTrait.PodAffinity = BoolP(true)
	affinityTrait.PodAffinityLabels = []string{"!criteria"}

	environment, deployment := createNominalDeploymentTraitTest()
	testApplyPodAffinityLabelsDoesSucceed(t, affinityTrait, environment, &deployment.Spec.Template.Spec)

	environment, knativeService := createNominalKnativeServiceTraitTest()
	testApplyPodAffinityLabelsDoesSucceed(t, affinityTrait, environment, &knativeService.Spec.Template.Spec.PodSpec)

	environment, cronJob := createNominalCronJobTraitTest()
	testApplyPodAffinityLabelsDoesSucceed(t, affinityTrait, environment, &cronJob.Spec.JobTemplate.Spec.Template.Spec)
}

func testApplyPodAffinityLabelsDoesSucceed(t *testing.T, trait *affinityTrait, environment *Environment, podSpec *corev1.PodSpec) {
	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.NotNil(t, podSpec.Affinity.PodAffinity)
	podAffinity := podSpec.Affinity.PodAffinity
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

func createNominalAffinityTest() *affinityTrait {
	trait := newAffinityTrait().(*affinityTrait)
	enabled := true
	trait.Enabled = &enabled

	return trait
}

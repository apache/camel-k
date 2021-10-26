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

	"gopkg.in/yaml.v2"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func TestConfigurePodTraitDoesSucceed(t *testing.T) {
	trait, environment, _ := createPodTest("")
	configured, err := trait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)

	configured, err = trait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
}

func TestSimpleChange(t *testing.T) {
	templateString := `containers:
  - name: second-container
    env:
      - name: test
        value: test`
	template := testPodTemplateSpec(t, templateString)

	assert.Equal(t, 3, len(template.Spec.Containers))
}

func TestMergeArrays(t *testing.T) {
	templateString :=
		"{containers: [{name: second-container, " +
			"env: [{name: SOME_VARIABLE, value: SOME_VALUE}, {name: SOME_VARIABLE2, value: SOME_VALUE2}]}, " +
			"{name: integration, env: [{name: TEST_ADDED_CUSTOM_VARIABLE, value: value}]}" +
			"]" +
			"}"
	templateSpec := testPodTemplateSpec(t, templateString)

	assert.NotNil(t, getContainer(templateSpec.Spec.Containers, "second-container"))
	assert.Equal(t, "SOME_VALUE", containsEnvVariables(templateSpec, "second-container", "SOME_VARIABLE"))
	assert.Equal(t, "SOME_VALUE2", containsEnvVariables(templateSpec, "second-container", "SOME_VARIABLE2"))
	assert.True(t, len(getContainer(templateSpec.Spec.Containers, "integration").Env) > 1)
	assert.Equal(t, "value", containsEnvVariables(templateSpec, "integration", "TEST_ADDED_CUSTOM_VARIABLE"))
}

func TestChangeEnvVariables(t *testing.T) {
	templateString := "{containers: [" +
		"{name: second, env: [{name: TEST_VARIABLE, value: TEST_VALUE}]}, " +
		"{name: integration, env: [{name: CAMEL_K_DIGEST, value: new_value}]}" +
		"]}"
	templateSpec := testPodTemplateSpec(t, templateString)

	// Check if env var was added in second container
	assert.Equal(t, containsEnvVariables(templateSpec, "second", "TEST_VARIABLE"), "TEST_VALUE")
	assert.Equal(t, 3, len(getContainer(templateSpec.Spec.Containers, "second").Env))

	// Check if env var was changed
	assert.Equal(t, containsEnvVariables(templateSpec, "integration", "CAMEL_K_DIGEST"), "new_value")
}

// nolint: unparam
func createPodTest(podSpecTemplate string) (*podTrait, *Environment, *appsv1.Deployment) {
	trait, _ := newPodTrait().(*podTrait)
	trait.Enabled = BoolP(true)

	var podSpec v1.PodSpec
	if podSpecTemplate != "" {
		_ = yaml.Unmarshal([]byte(podSpecTemplate), &podSpec)
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-template-test-integration",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-template",
					Labels: map[string]string{
						v1.IntegrationLabel: "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "integration",
							Env: []corev1.EnvVar{
								{
									Name:  "CAMEL_K_DIGEST",
									Value: "vO3wwJHC7-uGEiFFVac0jq6rZT5EZNw56Ae5gKKFZZsk",
								},
								{
									Name:  "CAMEL_K_CONF",
									Value: "/etc/camel/conf/application.properties",
								},
							},
						},
						{
							Name: "second",
							Env: []corev1.EnvVar{
								{
									Name:  "SOME_VARIABLE",
									Value: "SOME_VALUE",
								},
								{
									Name:  "SOME_VARIABLE2",
									Value: "SOME_VALUE2",
								},
							},
						},
					},
				},
			},
		},
	}

	environment := &Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-template-test-integration",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				PodTemplate: &v1.PodSpecTemplate{
					Spec: podSpec,
				},
			},
		},

		Resources: kubernetes.NewCollection(deployment),
	}

	return trait, environment, deployment
}

func containsEnvVariables(template corev1.PodTemplateSpec, containerName string, name string) string {
	container := getContainer(template.Spec.Containers, containerName)
	for i := range container.Env {
		env := container.Env[i]
		if env.Name == name {
			return env.Value
		}
	}
	return "not found!"
}

func getContainer(containers []corev1.Container, name string) *corev1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}

func testPodTemplateSpec(t *testing.T, template string) corev1.PodTemplateSpec {
	t.Helper()

	trait, environment, _ := createPodTest(template)

	_, err := trait.Configure(environment)
	assert.Nil(t, err)

	err = trait.Apply(environment)
	assert.Nil(t, err)

	deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == "pod-template-test-integration"
	})

	return deployment.Spec.Template
}

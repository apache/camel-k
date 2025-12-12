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
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

const (
	defaultContainerAnnotationKey = "kubectl.kubernetes.io/default-container"
	testNamespace                 = "test-ns"
	testIntegrationName           = "test-integration"
)

func TestDeploymentHasDefaultContainerAnnotation(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()
	err := deploymentTrait.Apply(environment)
	require.NoError(t, err)

	deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return true
	})
	require.NotNil(t, deployment, "Deployment should not be nil")

	podTemplateAnnotations := deployment.Spec.Template.ObjectMeta.Annotations

	t.Logf("Pod Template Annotations: %v", podTemplateAnnotations)

	value, exists := podTemplateAnnotations[defaultContainerAnnotationKey]

	assert.True(t, exists,
		"Expected annotation '%s' to be present on pod template, but it was not found. "+
			"Current annotations: %v", defaultContainerAnnotationKey, podTemplateAnnotations)

	if exists {
		assert.Equal(t, defaultContainerName, value,
			"Expected default container annotation to point to '%s', got '%s'",
			defaultContainerName, value)
	}
}

func TestDeploymentDefaultContainerAnnotationWithUserAnnotations(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()

	environment.Integration.Annotations = map[string]string{
		"user-annotation-1": "value1",
		"user-annotation-2": "value2",
	}

	err := deploymentTrait.Apply(environment)
	require.NoError(t, err)

	deployment := environment.Resources.GetDeployment(func(d *appsv1.Deployment) bool { return true })
	require.NotNil(t, deployment)

	annotations := deployment.Spec.Template.ObjectMeta.Annotations

	assert.Equal(t, defaultContainerName, annotations[defaultContainerAnnotationKey],
		"Default container annotation should be set to '%s'", defaultContainerName)

	t.Logf("All pod template annotations: %v", annotations)
}

func TestDeploymentDefaultContainerAnnotationValue(t *testing.T) {
	deploymentTrait, environment := createNominalDeploymentTest()

	err := deploymentTrait.Apply(environment)
	require.NoError(t, err)

	deployment := environment.Resources.GetDeployment(func(d *appsv1.Deployment) bool { return true })
	require.NotNil(t, deployment)

	value := deployment.Spec.Template.ObjectMeta.Annotations[defaultContainerAnnotationKey]

	assert.Equal(t, "integration", value,
		"Annotation value should be 'integration' to match the default container name")
}

func TestKnativeServiceHasDefaultContainerAnnotation(t *testing.T) {
	environment := createKnativeServiceTestEnvironment(t, &traitv1.KnativeServiceTrait{
		Trait: traitv1.Trait{
			Enabled: ptr.To(true),
		},
	})

	service := environment.Resources.GetKnativeService(func(s *serving.Service) bool {
		return true
	})
	require.NotNil(t, service, "Knative Service should not be nil")

	revisionAnnotations := service.Spec.ConfigurationSpec.Template.ObjectMeta.Annotations

	t.Logf("Knative Revision Template Annotations: %v", revisionAnnotations)

	value, exists := revisionAnnotations[defaultContainerAnnotationKey]

	assert.True(t, exists,
		"Expected annotation '%s' to be present on Knative revision template, but it was not found. "+
			"Current annotations: %v", defaultContainerAnnotationKey, revisionAnnotations)

	if exists {
		assert.Equal(t, defaultContainerName, value,
			"Expected default container annotation to point to '%s', got '%s'",
			defaultContainerName, value)
	}
}

func TestKnativeServiceDefaultContainerAnnotationWithAutoScaling(t *testing.T) {
	minScale := 1
	maxScale := 5
	environment := createKnativeServiceTestEnvironment(t, &traitv1.KnativeServiceTrait{
		Trait: traitv1.Trait{
			Enabled: ptr.To(true),
		},
		MinScale: &minScale,
		MaxScale: &maxScale,
	})

	service := environment.Resources.GetKnativeService(func(s *serving.Service) bool { return true })
	require.NotNil(t, service)

	annotations := service.Spec.ConfigurationSpec.Template.ObjectMeta.Annotations

	assert.Equal(t, defaultContainerName, annotations[defaultContainerAnnotationKey],
		"Default container annotation should be present")
	assert.Equal(t, "1", annotations["autoscaling.knative.dev/minScale"],
		"Min scale annotation should be present")
	assert.Equal(t, "5", annotations["autoscaling.knative.dev/maxScale"],
		"Max scale annotation should be present")

	t.Logf("All Knative revision annotations: %v", annotations)
}

func TestCronJobHasDefaultContainerAnnotation(t *testing.T) {
	environment := createCronJobTestEnvironment(t)

	cronJob := environment.Resources.GetCronJob(func(c *batchv1.CronJob) bool {
		return true
	})
	require.NotNil(t, cronJob, "CronJob should not be nil")

	podTemplateAnnotations := cronJob.Spec.JobTemplate.Spec.Template.ObjectMeta.Annotations

	t.Logf("CronJob Pod Template Annotations: %v", podTemplateAnnotations)

	value, exists := podTemplateAnnotations[defaultContainerAnnotationKey]

	assert.True(t, exists,
		"Expected annotation '%s' to be present on CronJob pod template, but it was not found. "+
			"Current annotations: %v", defaultContainerAnnotationKey, podTemplateAnnotations)

	if exists {
		assert.Equal(t, defaultContainerName, value,
			"Expected default container annotation to point to '%s', got '%s'",
			defaultContainerName, value)
	}
}

func TestCronJobDefaultContainerAnnotationWithUserAnnotations(t *testing.T) {
	environment := createCronJobTestEnvironmentWithAnnotations(t, map[string]string{
		"custom.annotation/key": "custom-value",
	})

	cronJob := environment.Resources.GetCronJob(func(c *batchv1.CronJob) bool { return true })
	require.NotNil(t, cronJob)

	annotations := cronJob.Spec.JobTemplate.Spec.Template.ObjectMeta.Annotations

	assert.Equal(t, defaultContainerName, annotations[defaultContainerAnnotationKey],
		"Default container annotation should be set to '%s'", defaultContainerName)

	t.Logf("All CronJob pod template annotations: %v", annotations)
}

func TestAllControllerStrategiesHaveDefaultContainerAnnotation(t *testing.T) {
	t.Run("Deployment", func(t *testing.T) {
		deploymentTrait, environment := createNominalDeploymentTest()
		err := deploymentTrait.Apply(environment)
		require.NoError(t, err)

		deployment := environment.Resources.GetDeployment(func(d *appsv1.Deployment) bool { return true })
		require.NotNil(t, deployment)

		assert.Equal(t, defaultContainerName,
			deployment.Spec.Template.ObjectMeta.Annotations[defaultContainerAnnotationKey],
			"Deployment should have default-container annotation")
	})

	t.Run("KnativeService", func(t *testing.T) {
		environment := createKnativeServiceTestEnvironment(t, &traitv1.KnativeServiceTrait{
			Trait: traitv1.Trait{Enabled: ptr.To(true)},
		})

		service := environment.Resources.GetKnativeService(func(s *serving.Service) bool { return true })
		require.NotNil(t, service)

		assert.Equal(t, defaultContainerName,
			service.Spec.ConfigurationSpec.Template.ObjectMeta.Annotations[defaultContainerAnnotationKey],
			"Knative Service should have default-container annotation")
	})

	t.Run("CronJob", func(t *testing.T) {
		environment := createCronJobTestEnvironment(t)

		cronJob := environment.Resources.GetCronJob(func(c *batchv1.CronJob) bool { return true })
		require.NotNil(t, cronJob)

		assert.Equal(t, defaultContainerName,
			cronJob.Spec.JobTemplate.Spec.Template.ObjectMeta.Annotations[defaultContainerAnnotationKey],
			"CronJob should have default-container annotation")
	})
}

func TestDefaultContainerAnnotationValueIsConsistent(t *testing.T) {
	expectedValue := "integration"

	t.Run("Deployment uses correct value", func(t *testing.T) {
		deploymentTrait, environment := createNominalDeploymentTest()
		_ = deploymentTrait.Apply(environment)
		deployment := environment.Resources.GetDeployment(func(d *appsv1.Deployment) bool { return true })
		require.NotNil(t, deployment)
		assert.Equal(t, expectedValue, deployment.Spec.Template.ObjectMeta.Annotations[defaultContainerAnnotationKey])
	})

	t.Run("KnativeService uses correct value", func(t *testing.T) {
		environment := createKnativeServiceTestEnvironment(t, &traitv1.KnativeServiceTrait{
			Trait: traitv1.Trait{Enabled: ptr.To(true)},
		})
		service := environment.Resources.GetKnativeService(func(s *serving.Service) bool { return true })
		require.NotNil(t, service)
		assert.Equal(t, expectedValue, service.Spec.ConfigurationSpec.Template.ObjectMeta.Annotations[defaultContainerAnnotationKey])
	})

	t.Run("CronJob uses correct value", func(t *testing.T) {
		environment := createCronJobTestEnvironment(t)
		cronJob := environment.Resources.GetCronJob(func(c *batchv1.CronJob) bool { return true })
		require.NotNil(t, cronJob)
		assert.Equal(t, expectedValue, cronJob.Spec.JobTemplate.Spec.Template.ObjectMeta.Annotations[defaultContainerAnnotationKey])
	})
}

func createCronJobTestEnvironment(t *testing.T) *Environment {
	t.Helper()
	return createCronJobTestEnvironmentWithAnnotations(t, nil)
}

func createCronJobTestEnvironmentWithAnnotations(t *testing.T, annotations map[string]string) *Environment {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	environment := &Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:        testIntegrationName,
				Namespace:   testNamespace,
				Annotations: annotations,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "routes.java",
							Content: `from("cron:tab?schedule=0 0/2 * * ?").to("log:test")`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Cron: &traitv1.CronTrait{},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Build: v1.IntegrationPlatformBuildSpec{
					RuntimeVersion: catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	c, err := newFakeClient(testNamespace)
	require.NoError(t, err)

	tc := NewCatalog(c)
	_, _, err = tc.apply(environment)
	require.NoError(t, err)

	return environment
}

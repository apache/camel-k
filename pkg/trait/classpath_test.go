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

	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/scylladb/go-set/strset"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestConfigureClasspathTraitInRightPhasesDoesSucceed(t *testing.T) {
	trait, environment := createNominalTest()

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.True(t, configured)
}

func TestConfigureClasspathTraitInWrongIntegrationPhaseDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalTest()
	environment.Integration.Status.Phase = v1alpha1.IntegrationPhaseError

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestConfigureClasspathTraitInWrongIntegrationKitPhaseDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalTest()
	environment.IntegrationKit.Status.Phase = v1alpha1.IntegrationKitPhaseWaitingForPlatform

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestConfigureClasspathDisabledTraitDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalTest()
	trait.Enabled = new(bool)

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestConfigureClasspathTraitPlaftormIntegrationKitLazyInstantiation(t *testing.T) {
	trait, environment := createNominalTest()
	environment.IntegrationKit = nil
	environment.Integration.Namespace = "kit-namespace"
	environment.Integration.Status.Kit = "kit-name"

	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.Equal(t, strset.New("/etc/camel/resources", "./resources"), environment.Classpath)
}

func TestConfigureClasspathTraitExternalIntegrationKitLazyInstantiation(t *testing.T) {
	trait, environment := createTestWithKitType(v1alpha1.IntegrationKitTypeExternal)
	environment.IntegrationKit = nil
	environment.Integration.Namespace = "kit-namespace"
	environment.Integration.Status.Kit = "kit-name"

	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.Equal(t, strset.New("/etc/camel/resources", "./resources", "/deployments/dependencies/*"), environment.Classpath)
}

func TestConfigureClasspathTraitWithIntegrationKitStatusArtifact(t *testing.T) {
	trait, environment := createNominalTest()
	environment.IntegrationKit.Status.Artifacts = []v1alpha1.Artifact{{ID: "", Location: "", Target: "/dep/target"}}

	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.NotNil(t, environment.Classpath)
	assert.Equal(t, strset.New("/etc/camel/resources", "./resources", "/dep/target"), environment.Classpath)
}

func TestConfigureClasspathTraitWithDeploymentResource(t *testing.T) {
	trait, environment := createNominalTest()

	container := corev1.Container{
		VolumeMounts: []corev1.VolumeMount{{MountPath: "/mount/path"}},
	}

	deployment := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
		},
	}

	environment.Resources = kubernetes.NewCollection(&deployment)

	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.NotNil(t, environment.Classpath)
	assert.Equal(t, strset.New("/etc/camel/resources", "./resources"), environment.Classpath)

	assert.Len(t, container.Env, 0)
}

func TestConfigureClasspathTraitWithKNativeResource(t *testing.T) {
	trait, environment := createNominalTest()

	container := corev1.Container{
		VolumeMounts: []corev1.VolumeMount{{MountPath: "/mount/path"}},
	}

	s := serving.Service{
		Spec: serving.ServiceSpec{
			RunLatest: &serving.RunLatestType{
				Configuration: serving.ConfigurationSpec{
					RevisionTemplate: serving.RevisionTemplateSpec{
						Spec: serving.RevisionSpec{
							Container: container,
						},
					},
				},
			},
		},
	}

	environment.Resources = kubernetes.NewCollection(&s)

	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.NotNil(t, environment.Classpath)
	assert.Equal(t, strset.New("/etc/camel/resources", "./resources", "/mount/path"), environment.Classpath)

	assert.Len(t, container.Env, 0)
}

func TestConfigureClasspathTraitWithNominalIntegrationKit(t *testing.T) {
	trait, environment := createNominalTest()

	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.NotNil(t, environment.Classpath)
	assert.Equal(t, strset.New("/etc/camel/resources", "./resources"), environment.Classpath)
}

func createNominalTest() (*classpathTrait, *Environment) {
	return createTestWithKitType(v1alpha1.IntegrationKitTypePlatform)
}

func createTestWithKitType(kitType string) (*classpathTrait, *Environment) {

	client, _ := test.NewFakeClient(
		&v1alpha1.IntegrationKit{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.IntegrationKindKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kit-namespace",
				Name:      "kit-name",
				Labels: map[string]string{
					"camel.apache.org/kit.type": kitType,
				},
			},
		},
	)

	trait := newClasspathTrait()
	enabled := true
	trait.Enabled = &enabled
	trait.ctx = context.TODO()
	trait.client = client

	environment := &Environment{
		Integration: &v1alpha1.Integration{
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
		},
		IntegrationKit: &v1alpha1.IntegrationKit{
			Status: v1alpha1.IntegrationKitStatus{
				Phase: v1alpha1.IntegrationKitPhaseReady,
			},
		},
	}

	return trait, environment
}

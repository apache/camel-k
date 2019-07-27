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
	"sort"
	"strings"
	"testing"

	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/scylladb/go-set/strset"
	"github.com/stretchr/testify/assert"
)

func TestConfigureClasspathTraitInRightPhasesDoesSucceed(t *testing.T) {
	trait, environment := createNominalClasspathTest()

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.True(t, configured)
}

func TestConfigureClasspathTraitInWrongIntegrationPhaseDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalClasspathTest()
	environment.Integration.Status.Phase = v1alpha1.IntegrationPhaseError

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestConfigureClasspathTraitInWrongIntegrationKitPhaseDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalClasspathTest()
	environment.IntegrationKit.Status.Phase = v1alpha1.IntegrationKitPhaseWaitingForPlatform

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestConfigureClasspathDisabledTraitDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalClasspathTest()
	trait.Enabled = new(bool)

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestApplyClasspathTraitPlaftormIntegrationKitLazyInstantiation(t *testing.T) {
	trait, environment := createNominalClasspathTest()
	environment.IntegrationKit = nil
	environment.Integration.Namespace = "kit-namespace"
	environment.Integration.Status.Kit = "kit-name"

	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.Equal(t, strset.New("/etc/camel/resources", "./resources"), environment.Classpath)
}

func TestApplyClasspathTraitExternalIntegrationKitLazyInstantiation(t *testing.T) {
	trait, environment := createClasspathTestWithKitType(v1alpha1.IntegrationKitTypeExternal)
	environment.IntegrationKit = nil
	environment.Integration.Namespace = "kit-namespace"
	environment.Integration.Status.Kit = "kit-name"

	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.Equal(t, strset.New("/etc/camel/resources", "./resources", "/deployments/dependencies/*"), environment.Classpath)
}

func TestApplyClasspathTraitWithIntegrationKitStatusArtifact(t *testing.T) {
	trait, environment := createNominalClasspathTest()
	environment.IntegrationKit.Status.Artifacts = []v1alpha1.Artifact{{ID: "", Location: "", Target: "/dep/target"}}

	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.NotNil(t, environment.Classpath)
	assert.Equal(t, strset.New("/etc/camel/resources", "./resources", "/dep/target"), environment.Classpath)
}

func TestApplyClasspathTraitWithDeploymentResource(t *testing.T) {
	trait, environment := createNominalClasspathTest()

	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/mount/path",
								},
							},
						},
					},
				},
			},
		},
	}

	environment.Resources.Add(&d)

	err := trait.Apply(environment)

	cp := environment.Classpath.List()
	sort.Strings(cp)

	assert.Nil(t, err)
	assert.NotNil(t, environment.Classpath)
	assert.Equal(t, strset.New("/etc/camel/resources", "./resources", "/mount/path"), environment.Classpath)
	assert.Len(t, d.Spec.Template.Spec.Containers[0].Env, 1)
	assert.Equal(t, "JAVA_CLASSPATH", d.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, strings.Join(cp, ":"), d.Spec.Template.Spec.Containers[0].Env[0].Value)
}

func TestApplyClasspathTraitWithKNativeResource(t *testing.T) {
	trait, environment := createNominalClasspathTest()

	s := serving.Service{}
	s.Spec.ConfigurationSpec.Template = &serving.RevisionTemplateSpec{}
	s.Spec.ConfigurationSpec.Template.Spec.Containers = []corev1.Container{
		{
			Name: defaultContainerName,
			VolumeMounts: []corev1.VolumeMount{
				{
					MountPath: "/mount/path",
				},
			},
		},
	}

	environment.Resources.Add(&s)

	err := trait.Apply(environment)

	cp := environment.Classpath.List()
	sort.Strings(cp)

	assert.Nil(t, err)
	assert.NotNil(t, environment.Classpath)
	assert.ElementsMatch(t, []string{"/etc/camel/resources", "./resources", "/mount/path"}, cp)
	assert.Len(t, s.Spec.ConfigurationSpec.Template.Spec.Containers[0].Env, 1)
	assert.Equal(t, "JAVA_CLASSPATH", s.Spec.ConfigurationSpec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, strings.Join(cp, ":"), s.Spec.ConfigurationSpec.Template.Spec.Containers[0].Env[0].Value)
}

func TestApplyClasspathTraitWithNominalIntegrationKit(t *testing.T) {
	trait, environment := createNominalClasspathTest()

	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.NotNil(t, environment.Classpath)
	assert.Equal(t, strset.New("/etc/camel/resources", "./resources"), environment.Classpath)
}

func createNominalClasspathTest() (*classpathTrait, *Environment) {
	return createClasspathTestWithKitType(v1alpha1.IntegrationKitTypePlatform)
}

func createClasspathTestWithKitType(kitType string) (*classpathTrait, *Environment) {

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
		Catalog: NewCatalog(context.TODO(), nil),
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
		Resources: kubernetes.NewCollection(),
	}

	return trait, environment
}

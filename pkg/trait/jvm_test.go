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
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/scylladb/go-set/strset"
	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
)

func TestConfigureJvmTraitInRightPhasesDoesSucceed(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.True(t, configured)
}

func TestConfigureJvmTraitInWrongIntegrationPhaseDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	environment.Integration.Status.Phase = v1.IntegrationPhaseError

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestConfigureJvmTraitInWrongIntegrationKitPhaseDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	environment.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseWaitingForPlatform

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestConfigureJvmDisabledTraitDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	trait.Enabled = new(bool)

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestApplyJvmTraitWithDeploymentResource(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)

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

	assert.Nil(t, err)

	cp := strset.New("./resources", configResourcesMountPath, dataResourcesMountPath, "/mount/path").List()
	sort.Strings(cp)

	assert.Equal(t, []string{
		"-cp",
		fmt.Sprintf("./resources:%s:%s:/mount/path", configResourcesMountPath, dataResourcesMountPath),
		"io.quarkus.bootstrap.runner.QuarkusEntryPoint",
	}, d.Spec.Template.Spec.Containers[0].Args)
}

func TestApplyJvmTraitWithKNativeResource(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)

	s := serving.Service{}
	s.Spec.ConfigurationSpec.Template = serving.RevisionTemplateSpec{}
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

	assert.Nil(t, err)

	cp := strset.New("./resources", configResourcesMountPath, dataResourcesMountPath, "/mount/path").List()
	sort.Strings(cp)

	assert.Equal(t, []string{
		"-cp",
		fmt.Sprintf("./resources:%s:%s:/mount/path", configResourcesMountPath, dataResourcesMountPath),
		"io.quarkus.bootstrap.runner.QuarkusEntryPoint",
	}, s.Spec.Template.Spec.Containers[0].Args)
}

func TestApplyJvmTraitWithDebugEnabled(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	trait.Debug = util.BoolP(true)
	trait.DebugSuspend = util.BoolP(true)

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

	assert.Nil(t, err)

	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args,
		"-agentlib:jdwp=transport=dt_socket,server=y,suspend=y,address=*:5005",
	)
}

func TestApplyJvmTraitWithExternalKitType(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypeExternal)

	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
						},
					},
				},
			},
		},
	}

	environment.Resources.Add(&d)

	err := trait.Apply(environment)
	assert.Nil(t, err)

	container := environment.getIntegrationContainer()

	assert.Equal(t, 3, len(container.Args))
	assert.Equal(t, "-cp", container.Args[0])

	// classpath JAR location segments must be wildcarded for an external kit
	for _, cp := range strings.Split(container.Args[1], ":") {
		if strings.HasPrefix(cp, builder.DeploymentDir) {
			assert.True(t, strings.HasSuffix(cp, "/*"))
		}
	}

	assert.Equal(t, "io.quarkus.bootstrap.runner.QuarkusEntryPoint", container.Args[2])
}

func createNominalJvmTest(kitType string) (*jvmTrait, *Environment) {
	catalog, _ := camel.DefaultCatalog()

	client, _ := test.NewFakeClient()

	trait := newJvmTrait().(*jvmTrait)
	trait.Enabled = util.BoolP(true)
	trait.PrintCommand = util.BoolP(false)
	trait.Ctx = context.TODO()
	trait.Client = client

	environment := &Environment{
		Catalog:      NewCatalog(context.TODO(), nil),
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.IntegrationKitKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kit-namespace",
				Name:      "kit-name",
				Labels: map[string]string{
					"camel.apache.org/kit.type": kitType,
				},
			},
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Resources: kubernetes.NewCollection(),
	}

	return trait, environment
}

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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"

	"github.com/stretchr/testify/assert"

	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestProbesDeps(t *testing.T) {
	e := Environment{
		Integration: &v1alpha1.Integration{
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseInitialization,
			},
		},
	}

	enabled := true

	tr := newProbesTrait()
	tr.Enabled = &enabled
	tr.BindPort = 9191

	ok, err := tr.Configure(&e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = tr.Apply(&e)
	assert.Nil(t, err)
	assert.Contains(t, e.Integration.Status.Dependencies, "runtime:health")
}

func TestProbesOnDeployment(t *testing.T) {
	target := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
		},
	}

	e := Environment{
		Resources: kubernetes.NewCollection(&target),
		Integration: &v1alpha1.Integration{
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
		},
	}

	enabled := true

	tr := newProbesTrait()
	tr.Enabled = &enabled
	tr.BindPort = 9191
	tr.LivenessTimeout = 1234

	ok, err := tr.Configure(&e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = tr.Apply(&e)
	assert.Nil(t, err)
	assert.Equal(t, "", target.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Host)
	assert.Equal(t, int32(9191), target.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, "/health", target.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Path)
	assert.Equal(t, "", target.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Host)
	assert.Equal(t, int32(9191), target.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, "/health", target.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Path)
	assert.Equal(t, int32(1234), target.Spec.Template.Spec.Containers[0].LivenessProbe.TimeoutSeconds)
}

func TestProbesOnKnativeService(t *testing.T) {
	target := serving.Service{
		Spec: serving.ServiceSpec{
			RunLatest: &serving.RunLatestType{},
		},
	}

	e := Environment{
		Resources: kubernetes.NewCollection(&target),
		Integration: &v1alpha1.Integration{
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
		},
	}

	enabled := true

	tr := newProbesTrait()
	tr.Enabled = &enabled
	tr.BindPort = 9191
	tr.ReadinessTimeout = 4321

	ok, err := tr.Configure(&e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = tr.Apply(&e)
	assert.Nil(t, err)
	assert.Nil(t, target.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.LivenessProbe)
	assert.Nil(t, target.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.ReadinessProbe)
}

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
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"

	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func NewIstioTestEnv(t *testing.T, d *appsv1.Deployment, s *serving.Service) Environment {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	env := Environment{
		Catalog:      NewEnvironmentTestCatalog(),
		CamelCatalog: catalog,
		Integration: &v1alpha1.Integration{
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
			Spec: v1alpha1.IntegrationSpec{
				Traits: map[string]v1alpha1.TraitSpec{
					"istio": {
						Configuration: map[string]string{
							"enabled": "true",
						},
					},
				},
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: v1alpha1.IntegrationPlatformClusterOpenShift,
				Profile: v1alpha1.TraitProfileKnative,
			},
		},
		EnvVars:   make([]corev1.EnvVar, 0),
		Resources: kubernetes.NewCollection(s, d),
	}

	return env
}

func TestIstioInject(t *testing.T) {
	s := serving.Service{
		Spec: serving.ServiceSpec{
			RunLatest: &serving.RunLatestType{
				Configuration: serving.ConfigurationSpec{
					RevisionTemplate: serving.RevisionTemplateSpec{},
				},
			},
		},
	}
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{},
		},
	}

	env := NewIstioTestEnv(t, &d, &s)

	err := env.Catalog.apply(&env)
	assert.Nil(t, err)

	assert.Empty(t, s.Spec.RunLatest.Configuration.RevisionTemplate.Annotations[istioSidecarInjectAnnotation])
	assert.NotEmpty(t, d.Spec.Template.Annotations[istioSidecarInjectAnnotation])
}

func TestIstioForcedInjectTrue(t *testing.T) {
	s := serving.Service{
		Spec: serving.ServiceSpec{
			RunLatest: &serving.RunLatestType{
				Configuration: serving.ConfigurationSpec{
					RevisionTemplate: serving.RevisionTemplateSpec{},
				},
			},
		},
	}
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{},
		},
	}

	env := NewIstioTestEnv(t, &d, &s)
	env.Integration.Spec.Traits["istio"].Configuration["inject"] = "true"

	err := env.Catalog.apply(&env)
	assert.Nil(t, err)

	assert.Equal(t, "true", s.Spec.RunLatest.Configuration.RevisionTemplate.Annotations[istioSidecarInjectAnnotation])
	assert.Equal(t, "true", d.Spec.Template.Annotations[istioSidecarInjectAnnotation])
}

func TestIstioForcedInjectFalse(t *testing.T) {
	s := serving.Service{
		Spec: serving.ServiceSpec{
			RunLatest: &serving.RunLatestType{
				Configuration: serving.ConfigurationSpec{
					RevisionTemplate: serving.RevisionTemplateSpec{},
				},
			},
		},
	}
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{},
		},
	}

	env := NewIstioTestEnv(t, &d, &s)
	env.Integration.Spec.Traits["istio"].Configuration["inject"] = "false"

	err := env.Catalog.apply(&env)
	assert.Nil(t, err)

	assert.Equal(t, "false", s.Spec.RunLatest.Configuration.RevisionTemplate.Annotations[istioSidecarInjectAnnotation])
	assert.Equal(t, "false", d.Spec.Template.Annotations[istioSidecarInjectAnnotation])
}

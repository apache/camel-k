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
	"k8s.io/utils/pointer"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func NewIstioTestEnv(t *testing.T, d *appsv1.Deployment, s *serving.Service, enabled bool) Environment {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	env := Environment{
		Catalog:      NewEnvironmentTestCatalog(),
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Traits: v1.Traits{},
			},
		},
		Platform: &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
			},
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Profile: v1.TraitProfileKnative,
			},
		},
		EnvVars:   make([]corev1.EnvVar, 0),
		Resources: kubernetes.NewCollection(s, d),
	}
	env.Platform.ResyncStatusFullConfig()

	if enabled {
		env.Integration.Spec.Traits.Istio = &v1.IstioTrait{
			Trait: v1.Trait{
				Enabled: pointer.Bool(true),
			},
		}
	}

	return env
}

func TestIstioInject(t *testing.T) {
	s := serving.Service{
		Spec: serving.ServiceSpec{
			ConfigurationSpec: serving.ConfigurationSpec{
				Template: serving.RevisionTemplateSpec{},
			},
		},
	}
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{},
		},
	}

	env := NewIstioTestEnv(t, &d, &s, true)
	err := env.Catalog.apply(&env)
	assert.Nil(t, err)

	assert.Empty(t, s.Spec.ConfigurationSpec.Template.Annotations[istioSidecarInjectAnnotation])
	assert.NotEmpty(t, d.Spec.Template.Annotations[istioSidecarInjectAnnotation])
}

func TestIstioForcedInjectTrue(t *testing.T) {
	s := serving.Service{
		Spec: serving.ServiceSpec{
			ConfigurationSpec: serving.ConfigurationSpec{
				Template: serving.RevisionTemplateSpec{},
			},
		},
	}
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{},
		},
	}

	env := NewIstioTestEnv(t, &d, &s, true)
	env.Integration.Spec.Traits.Istio.Enabled = pointer.Bool(true)
	env.Integration.Spec.Traits.Istio.Inject = pointer.Bool(true)

	err := env.Catalog.apply(&env)
	assert.Nil(t, err)

	assert.Equal(t, "true", s.Spec.ConfigurationSpec.Template.Annotations[istioSidecarInjectAnnotation])
	assert.Equal(t, "true", d.Spec.Template.Annotations[istioSidecarInjectAnnotation])
}

func TestIstioForcedInjectFalse(t *testing.T) {
	s := serving.Service{
		Spec: serving.ServiceSpec{
			ConfigurationSpec: serving.ConfigurationSpec{
				Template: serving.RevisionTemplateSpec{},
			},
		},
	}
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{},
		},
	}

	env := NewIstioTestEnv(t, &d, &s, true)
	env.Integration.Spec.Traits.Istio.Enabled = pointer.Bool(true)
	env.Integration.Spec.Traits.Istio.Inject = pointer.Bool(false)

	err := env.Catalog.apply(&env)
	assert.Nil(t, err)

	assert.Equal(t, "false", s.Spec.ConfigurationSpec.Template.Annotations[istioSidecarInjectAnnotation])
	assert.Equal(t, "false", d.Spec.Template.Annotations[istioSidecarInjectAnnotation])
}

func TestIstioDisabled(t *testing.T) {
	s := serving.Service{
		Spec: serving.ServiceSpec{
			ConfigurationSpec: serving.ConfigurationSpec{
				Template: serving.RevisionTemplateSpec{},
			},
		},
	}
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{},
		},
	}

	env := NewIstioTestEnv(t, &d, &s, false)

	err := env.Catalog.apply(&env)
	assert.Nil(t, err)
	assert.NotContains(t, env.ExecutedTraits, "istio")
}

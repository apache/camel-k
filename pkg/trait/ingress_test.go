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

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func TestConfigureIngressTraitDoesSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	configured, condition, err := ingressTrait.Configure(environment)

	assert.True(t, configured)
	require.NoError(t, err)
	assert.Nil(t, condition)
	assert.Len(t, environment.Integration.Status.Conditions, 0)
	assert.Nil(t, condition)

}

func TestConfigureDisabledIngressTraitDoesNotSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	ingressTrait.Enabled = ptr.To(false)

	expectedCondition := NewIntegrationCondition(
		"Ingress",
		v1.IntegrationConditionExposureAvailable,
		corev1.ConditionFalse,
		v1.IntegrationConditionIngressNotAvailableReason,
		"explicitly disabled",
	)
	configured, condition, err := ingressTrait.Configure(environment)

	assert.False(t, configured)
	require.NoError(t, err)
	assert.NotNil(t, condition)
	assert.Equal(t, expectedCondition, condition)
}

func TestConfigureIngressTraitInWrongPhaseDoesNotSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseError

	configured, condition, err := ingressTrait.Configure(environment)

	assert.True(t, configured)
	require.NoError(t, err)
	assert.Nil(t, condition)
	assert.Len(t, environment.Integration.Status.Conditions, 0)
}

func TestConfigureAutoIngressTraitWithUserServiceDoesSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	ingressTrait.Auto = nil

	configured, condition, err := ingressTrait.Configure(environment)

	assert.True(t, configured)
	require.NoError(t, err)
	assert.Nil(t, condition)
	assert.Len(t, environment.Integration.Status.Conditions, 0)
}

func TestApplyIngressTraitWithoutUserServiceDoesNotSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	environment.Resources = kubernetes.NewCollection()

	err := ingressTrait.Apply(environment)

	require.Error(t, err)
	assert.Equal(t, "cannot apply ingress trait: no target service", err.Error())
	assert.Len(t, environment.Resources.Items(), 0)
}

func TestApplyIngressTraitDoesSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()

	err := ingressTrait.Apply(environment)

	require.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	assert.Len(t, environment.Resources.Items(), 2)
	environment.Resources.Visit(func(resource runtime.Object) {
		if ingress, ok := resource.(*networkingv1.Ingress); ok {
			assert.Equal(t, "service-name", ingress.Name)
			assert.Equal(t, "namespace", ingress.Namespace)
			assert.Len(t, ingress.Spec.Rules, 1)
			assert.Equal(t, "hostname", ingress.Spec.Rules[0].Host)
			assert.Len(t, ingress.Spec.Rules[0].HTTP.Paths, 1)
			assert.Equal(t, "service-name", ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name)
			assert.Equal(t, "/", ingress.Spec.Rules[0].HTTP.Paths[0].Path)
			assert.NotNil(t, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
			assert.Nil(t, ingress.Spec.TLS)
			assert.Equal(t, networkingv1.PathTypePrefix, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
			assert.Nil(t, ingress.Spec.IngressClassName)
		}
	})

	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, "service-name(hostname) -> service-name(http)", conditions[0].Message)
}

func TestApplyIngressTraitWithPathDoesSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	ingressTrait.Path = string("/path")

	err := ingressTrait.Apply(environment)

	require.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	assert.Len(t, environment.Resources.Items(), 2)
	environment.Resources.Visit(func(resource runtime.Object) {
		if ingress, ok := resource.(*networkingv1.Ingress); ok {
			assert.Equal(t, "service-name", ingress.Name)
			assert.Equal(t, "namespace", ingress.Namespace)
			assert.Len(t, ingress.Spec.Rules, 1)
			assert.Equal(t, "hostname", ingress.Spec.Rules[0].Host)
			assert.Len(t, ingress.Spec.Rules[0].HTTP.Paths, 1)
			assert.Equal(t, "/path", ingress.Spec.Rules[0].HTTP.Paths[0].Path)
			assert.Equal(t, "service-name", ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name)
			assert.NotNil(t, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
			assert.Equal(t, networkingv1.PathTypePrefix, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
		}
	})

	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, "service-name(hostname) -> service-name(http)", conditions[0].Message)
}

func TestApplyIngressTraitWithPathsDoesSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	ingressTrait.Paths = []string{"/path-a", "/path-b"}

	err := ingressTrait.Apply(environment)

	require.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	assert.Len(t, environment.Resources.Items(), 2)
	environment.Resources.Visit(func(resource runtime.Object) {
		if ingress, ok := resource.(*networkingv1.Ingress); ok {
			assert.Equal(t, "service-name", ingress.Name)
			assert.Equal(t, "namespace", ingress.Namespace)
			assert.Len(t, ingress.Spec.Rules, 1)
			assert.Equal(t, "hostname", ingress.Spec.Rules[0].Host)
			assert.Len(t, ingress.Spec.Rules[0].HTTP.Paths, 2)
			assert.Equal(t, "/path-a", ingress.Spec.Rules[0].HTTP.Paths[0].Path)
			assert.Equal(t, "/path-b", ingress.Spec.Rules[0].HTTP.Paths[1].Path)
			assert.Equal(t, "service-name", ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name)
			assert.NotNil(t, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
			assert.Equal(t, networkingv1.PathTypePrefix, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
		}
	})

	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, "service-name(hostname) -> service-name(http)", conditions[0].Message)
}

func TestApplyIngressTraitWithPathAndPathsDoesSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	ingressTrait.Path = string("/path")
	ingressTrait.Paths = []string{"/path-a", "/path-b"}

	err := ingressTrait.Apply(environment)

	require.NoError(t, err)
	environment.Resources.Visit(func(resource runtime.Object) {
		if ingress, ok := resource.(*networkingv1.Ingress); ok {
			assert.Len(t, ingress.Spec.Rules[0].HTTP.Paths, 3)
		}
	})
}

func TestApplyIngressTraitWithIngressClassNameDoesSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTestWithIngressClassName("someIngressClass")

	err := ingressTrait.Apply(environment)

	require.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	assert.Len(t, environment.Resources.Items(), 2)
	environment.Resources.Visit(func(resource runtime.Object) {
		if ingress, ok := resource.(*networkingv1.Ingress); ok {
			assert.Equal(t, "service-name", ingress.Name)
			assert.Equal(t, "namespace", ingress.Namespace)
			assert.Len(t, ingress.Spec.Rules, 1)
			assert.Equal(t, "hostname", ingress.Spec.Rules[0].Host)
			assert.Len(t, ingress.Spec.Rules[0].HTTP.Paths, 1)
			assert.Equal(t, "service-name", ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name)
			assert.Equal(t, "/", ingress.Spec.Rules[0].HTTP.Paths[0].Path)
			assert.NotNil(t, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
			assert.Nil(t, ingress.Spec.TLS)
			assert.Equal(t, networkingv1.PathTypePrefix, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
			assert.Equal(t, "someIngressClass", *ingress.Spec.IngressClassName)
		}
	})

	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, "service-name(hostname) -> service-name(http)", conditions[0].Message)
}

func TestConfigureTLSIngressTraitWDoesSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	ingressTrait.TLSHosts = []string{"host1.com", "host2.com"}
	ingressTrait.TLSSecretName = "nginxWildcard"

	err := ingressTrait.Apply(environment)

	require.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	assert.Len(t, environment.Resources.Items(), 2)
	environment.Resources.Visit(func(resource runtime.Object) {
		if ingress, ok := resource.(*networkingv1.Ingress); ok {
			assert.Equal(t, "service-name", ingress.Name)
			assert.Equal(t, "namespace", ingress.Namespace)
			assert.Len(t, ingress.Spec.Rules, 1)
			assert.Equal(t, "hostname", ingress.Spec.Rules[0].Host)
			assert.Len(t, ingress.Spec.Rules[0].HTTP.Paths, 1)
			assert.Equal(t, "service-name", ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name)
			assert.Equal(t, "/", ingress.Spec.Rules[0].HTTP.Paths[0].Path)
			assert.NotNil(t, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
			assert.Equal(t, networkingv1.PathTypePrefix, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
			assert.Equal(t, "host1.com", ingress.Spec.TLS[0].Hosts[0])
			assert.Equal(t, "host2.com", ingress.Spec.TLS[0].Hosts[1])
			assert.Equal(t, "nginxWildcard", ingress.Spec.TLS[0].SecretName)
		}
	})

	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, "service-name(hostname) -> service-name(http)", conditions[0].Message)
}

func TestConfigureTLSWithoutHostsIngressTraitWDoesSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	ingressTrait.TLSSecretName = "nginxWildcard"

	err := ingressTrait.Apply(environment)

	require.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	assert.Len(t, environment.Resources.Items(), 2)
	environment.Resources.Visit(func(resource runtime.Object) {
		if ingress, ok := resource.(*networkingv1.Ingress); ok {
			assert.Equal(t, "service-name", ingress.Name)
			assert.Equal(t, "namespace", ingress.Namespace)
			assert.Len(t, ingress.Spec.Rules, 1)
			assert.Equal(t, "hostname", ingress.Spec.Rules[0].Host)
			assert.Len(t, ingress.Spec.Rules[0].HTTP.Paths, 1)
			assert.Equal(t, "service-name", ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name)
			assert.Equal(t, "/", ingress.Spec.Rules[0].HTTP.Paths[0].Path)
			assert.NotNil(t, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
			assert.Equal(t, networkingv1.PathTypePrefix, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
			assert.Nil(t, ingress.Spec.TLS)
		}
	})

	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, "service-name(hostname) -> service-name(http)", conditions[0].Message)
}

func TestConfigureTLSWithoutSecretNameIngressTraitWDoesSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	ingressTrait.TLSHosts = []string{"host1.com", "host2.com"}

	err := ingressTrait.Apply(environment)

	require.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	assert.Len(t, environment.Resources.Items(), 2)
	environment.Resources.Visit(func(resource runtime.Object) {
		if ingress, ok := resource.(*networkingv1.Ingress); ok {
			assert.Equal(t, "service-name", ingress.Name)
			assert.Equal(t, "namespace", ingress.Namespace)
			assert.Len(t, ingress.Spec.Rules, 1)
			assert.Equal(t, "hostname", ingress.Spec.Rules[0].Host)
			assert.Len(t, ingress.Spec.Rules[0].HTTP.Paths, 1)
			assert.Equal(t, "service-name", ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name)
			assert.Equal(t, "/", ingress.Spec.Rules[0].HTTP.Paths[0].Path)
			assert.NotNil(t, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
			assert.Equal(t, networkingv1.PathTypePrefix, *ingress.Spec.Rules[0].HTTP.Paths[0].PathType)
			assert.Nil(t, ingress.Spec.TLS)
		}
	})

	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, "service-name(hostname) -> service-name(http)", conditions[0].Message)
}

func createNominalIngressTestWithIngressClassName(ingressClassName string) (*ingressTrait, *Environment) {
	trait, environment := createNominalIngressTest()
	trait.IngressClassName = ingressClassName
	return trait, environment
}

func createNominalIngressTest() (*ingressTrait, *Environment) {
	trait, _ := newIngressTrait().(*ingressTrait)
	trait.Enabled = ptr.To(true)
	trait.Auto = ptr.To(false)
	trait.Host = "hostname"

	environment := &Environment{
		Catalog: NewCatalog(nil),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "integration-name",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		Resources: kubernetes.NewCollection(
			&corev1.Service{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service-name",
					Namespace: "namespace",
					Labels: map[string]string{
						v1.IntegrationLabel:             "integration-name",
						"camel.apache.org/service.type": v1.ServiceTypeUser,
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{},
					Selector: map[string]string{
						v1.IntegrationLabel: "integration-name",
					},
				},
			},
		),
	}

	return trait, environment
}

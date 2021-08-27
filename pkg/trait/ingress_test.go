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
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func TestConfigureIngressTraitDoesSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	configured, err := ingressTrait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 0)
}

func TestConfigureDisabledIngressTraitDoesNotSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	ingressTrait.Enabled = BoolP(false)

	configured, err := ingressTrait.Configure(environment)

	assert.False(t, configured)
	assert.Nil(t, err)
	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, "explicitly disabled", conditions[0].Message)
}

func TestConfigureIngressTraitInWrongPhaseDoesNotSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseError

	configured, err := ingressTrait.Configure(environment)

	assert.False(t, configured)
	assert.Nil(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 0)
}

func TestConfigureAutoIngressTraitWithoutUserServiceDoesNotSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	ingressTrait.Auto = BoolP(true)
	environment.Resources = kubernetes.NewCollection()

	configured, err := ingressTrait.Configure(environment)

	assert.False(t, configured)
	assert.Nil(t, err)
	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, "no host or service defined", conditions[0].Message)
}

func TestConfigureAutoIngressTraitWithEmptyHostDoesNotSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	ingressTrait.Auto = nil
	ingressTrait.Host = ""

	configured, err := ingressTrait.Configure(environment)

	assert.False(t, configured)
	assert.Nil(t, err)
	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, "no host or service defined", conditions[0].Message)
}

func TestConfigureAutoIngressTraitWithUserServiceDoesSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	ingressTrait.Auto = nil

	configured, err := ingressTrait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 0)
}

func TestConfigureIngressTraitWithoutHostDoesNotSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	ingressTrait.Host = ""

	configured, err := ingressTrait.Configure(environment)

	assert.False(t, configured)
	assert.NotNil(t, err)
	assert.Equal(t, "cannot Apply ingress trait: no host defined", err.Error())
	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, "no host defined", conditions[0].Message)
}

func TestApplyIngressTraitWithoutUserServiceDoesNotSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()
	environment.Resources = kubernetes.NewCollection()

	err := ingressTrait.Apply(environment)

	assert.NotNil(t, err)
	assert.Equal(t, "cannot Apply ingress trait: no target service", err.Error())
	assert.Len(t, environment.Resources.Items(), 0)
}

func TestApplyIngressTraitDoesSucceed(t *testing.T) {
	ingressTrait, environment := createNominalIngressTest()

	err := ingressTrait.Apply(environment)

	assert.Nil(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	assert.Len(t, environment.Resources.Items(), 2)
	environment.Resources.Visit(func(resource runtime.Object) {
		if ingress, ok := resource.(*networking.Ingress); ok {
			assert.Equal(t, "service-name", ingress.Name)
			assert.Equal(t, "namespace", ingress.Namespace)
			assert.Equal(t, "service-name", ingress.Spec.DefaultBackend.Service.Name)
			assert.Len(t, ingress.Spec.Rules, 1)
			assert.Equal(t, "hostname", ingress.Spec.Rules[0].Host)
		}
	})

	conditions := environment.Integration.Status.Conditions
	assert.Len(t, conditions, 1)
	assert.Equal(t, "service-name(hostname) -> service-name(http)", conditions[0].Message)
}

func createNominalIngressTest() (*ingressTrait, *Environment) {
	trait := newIngressTrait().(*ingressTrait)
	trait.Enabled = BoolP(true)
	trait.Auto = BoolP(false)
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

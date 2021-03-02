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

	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func TestConfigurePrometheusTraitInRightPhaseDoesSucceed(t *testing.T) {
	trait, environment := createNominalPrometheusTest()

	configured, err := trait.Configure(environment)

	assert.Nil(t, err)
	assert.True(t, configured)
}

func TestConfigurePrometheusTraitInWrongPhaseDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalPrometheusTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseResolvingKit

	configured, err := trait.Configure(environment)

	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestApplyNominalPrometheusTraitDoesSucceed(t *testing.T) {
	trait, environment := createNominalPrometheusTest()

	err := trait.Apply(environment)

	assert.Nil(t, err)

	container := environment.Resources.GetContainerByName(defaultContainerName)
	assert.NotNil(t, container)

	assert.Empty(t, container.Args)

	ports := container.Ports
	assert.Len(t, ports, 1)
	assert.Equal(t, int32(8080), ports[0].ContainerPort)
	assert.Equal(t, corev1.ProtocolTCP, ports[0].Protocol)

	service := environment.Resources.GetService(func(service *corev1.Service) bool {
		return service.Name == "integration-name-prometheus"
	})
	assert.NotNil(t, service)
	assert.Len(t, service.Spec.Ports, 1)

	serviceMonitor := environment.Resources.GetServiceMonitor(func(service *monitoringv1.ServiceMonitor) bool {
		return service.Name == "integration-name"
	})
	assert.NotNil(t, serviceMonitor)

	assert.Len(t, environment.Integration.Status.Conditions, 1)
	condition := environment.Integration.Status.Conditions[0]
	assert.Equal(t, v1.IntegrationConditionPrometheusAvailable, condition.Type)
	assert.Equal(t, corev1.ConditionTrue, condition.Status)
}

func TestApplyPrometheusTraitWithoutContainerDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalPrometheusTest()
	environment.Resources = kubernetes.NewCollection()

	err := trait.Apply(environment)

	assert.Nil(t, err)

	assert.Len(t, environment.Integration.Status.Conditions, 1)
	condition := environment.Integration.Status.Conditions[0]
	assert.Equal(t, v1.IntegrationConditionPrometheusAvailable, condition.Type)
	assert.Equal(t, corev1.ConditionFalse, condition.Status)
}

func TestApplyPrometheusTraitWithServiceDoesSucceed(t *testing.T) {
	trait, environment := createNominalPrometheusTest()
	environment.Resources = kubernetes.NewCollection(
		&appsv1.Deployment{
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
		},
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
		})

	err := trait.Apply(environment)

	assert.Nil(t, err)

	assert.Len(t, environment.Integration.Status.Conditions, 1)
	condition := environment.Integration.Status.Conditions[0]
	assert.Equal(t, v1.IntegrationConditionPrometheusAvailableReason, condition.Reason)
	assert.Equal(t, corev1.ConditionTrue, condition.Status)
}

func TestPrometheusTraitGetServiceMonitor(t *testing.T) {
	trait, environment := createNominalPrometheusTest()

	serviceMonitor, err := trait.getServiceMonitorFor(environment)

	assert.Nil(t, err)

	assert.NotNil(t, serviceMonitor)
	assert.Equal(t, "ServiceMonitor", serviceMonitor.Kind)
	assert.Equal(t, "monitoring.coreos.com/v1", serviceMonitor.APIVersion)
	assert.Equal(t, "integration-name", serviceMonitor.Name)
	assert.Equal(t, "integration-namespace", serviceMonitor.Namespace)
	assert.Equal(t, "integration-name", serviceMonitor.Labels[v1.IntegrationLabel])
	assert.Equal(t, "integration-name", serviceMonitor.Spec.Selector.MatchLabels[v1.IntegrationLabel])
	assert.Len(t, serviceMonitor.Spec.Endpoints, 1)
	assert.Equal(t, "prometheus", serviceMonitor.Spec.Endpoints[0].Port)
	assert.Equal(t, "/q/metrics", serviceMonitor.Spec.Endpoints[0].Path)
}

func createNominalPrometheusTest() (*prometheusTrait, *Environment) {
	trait := newPrometheusTrait().(*prometheusTrait)
	enabled := true
	trait.Enabled = &enabled

	camelCatalog, err := camel.DefaultCatalog()
	if err != nil {
		panic(err)
	}

	environment := &Environment{
		Catalog:      NewCatalog(context.TODO(), nil),
		CamelCatalog: camelCatalog,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "integration-namespace",
				Name:      "integration-name",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		Resources: kubernetes.NewCollection(
			&appsv1.Deployment{
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
			},
		),
	}

	return trait, environment
}

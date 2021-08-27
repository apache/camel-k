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
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit

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

	podMonitor := environment.Resources.GetPodMonitor(func(pm *monitoringv1.PodMonitor) bool {
		return pm.Name == "integration-name"
	})
	assert.NotNil(t, podMonitor)

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

func TestPrometheusTraitGetPodMonitor(t *testing.T) {
	trait, environment := createNominalPrometheusTest()

	podMonitor, err := trait.getPodMonitorFor(environment, defaultContainerPortName)

	assert.Nil(t, err)

	assert.NotNil(t, podMonitor)
	assert.Equal(t, "PodMonitor", podMonitor.Kind)
	assert.Equal(t, "monitoring.coreos.com/v1", podMonitor.APIVersion)
	assert.Equal(t, "integration-name", podMonitor.Name)
	assert.Equal(t, "integration-namespace", podMonitor.Namespace)
	assert.Equal(t, "integration-name", podMonitor.Labels["camel.apache.org/integration"])
	assert.Equal(t, "integration-name", podMonitor.Spec.Selector.MatchLabels["camel.apache.org/integration"])
	assert.Len(t, podMonitor.Spec.PodMetricsEndpoints, 1)
	assert.Equal(t, defaultContainerPortName, podMonitor.Spec.PodMetricsEndpoints[0].Port)
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
		Catalog:      NewCatalog(nil),
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

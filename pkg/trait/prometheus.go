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
	"fmt"
	"strconv"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/envvar"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
)

type prometheusTrait struct {
	BaseTrait `property:",squash"`

	Port                 int    `property:"port"`
	ServiceMonitor       bool   `property:"service-monitor"`
	ServiceMonitorLabels string `property:"service-monitor-labels"`
}

const prometheusPortName = "prometheus"

// The Prometheus trait must be executed prior to the deployment trait
// as it mutates environment variables
func newPrometheusTrait() *prometheusTrait {
	return &prometheusTrait{
		BaseTrait:      newBaseTrait("prometheus"),
		Port:           9779,
		ServiceMonitor: true,
	}
}

func (t *prometheusTrait) Configure(e *Environment) (bool, error) {
	return e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
}

func (t *prometheusTrait) Apply(e *Environment) (err error) {
	if t.Enabled == nil || !*t.Enabled {
		// Deactivate the Prometheus Java agent
		// Note: the AB_PROMETHEUS_OFF environment variable acts as an option flag
		envvar.SetVal(&e.EnvVars, "AB_PROMETHEUS_OFF", "true")
		return nil
	}

	// Configure the Prometheus Java agent
	envvar.SetVal(&e.EnvVars, "AB_PROMETHEUS_PORT", strconv.Itoa(t.Port))

	service, servicePort := t.configureServicePort(e)
	container, containerPort := t.configureContainerPort(e)

	condition := v1alpha1.IntegrationCondition{
		Type:   v1alpha1.IntegrationConditionPrometheusAvailable,
		Status: corev1.ConditionTrue,
		Reason: v1alpha1.IntegrationConditionPrometheusAvailableReason,
	}

	if servicePort != nil {
		service.Spec.Ports = append(service.Spec.Ports, *servicePort)
		condition.Message = fmt.Sprintf("%s(%s/%d) -> ", service.Name, servicePort.Name, servicePort.Port)
	} else {
		condition.Status = corev1.ConditionFalse
		condition.Reason = v1alpha1.IntegrationConditionServiceNotAvailableReason
	}

	if containerPort != nil {
		container.Ports = append(container.Ports, *containerPort)
		condition.Message += fmt.Sprintf("%s(%s/%d)", container.Name, containerPort.Name, containerPort.ContainerPort)
	} else {
		condition.Status = corev1.ConditionFalse
		condition.Reason = v1alpha1.IntegrationConditionContainerNotAvailableReason
	}

	e.Integration.Status.SetConditions(condition)

	if condition.Status == corev1.ConditionFalse {
		return nil
	}

	if t.ServiceMonitor {
		// Add the ServiceMonitor resource
		smt, err := t.getServiceMonitorFor(e)
		if err != nil {
			return err
		}
		e.Resources.Add(smt)
	}

	return nil
}

func (t *prometheusTrait) configureContainerPort(e *Environment) (*corev1.Container, *corev1.ContainerPort) {
	container := e.Resources.GetContainerForIntegration(e.Integration)
	if container == nil {
		return nil, nil
	}

	containerPort := corev1.ContainerPort{
		Name:          prometheusPortName,
		ContainerPort: int32(t.Port),
		Protocol:      corev1.ProtocolTCP,
	}

	return container, &containerPort
}

func (t *prometheusTrait) configureServicePort(e *Environment) (*corev1.Service, *corev1.ServicePort) {
	service := e.Resources.GetServiceForIntegration(e.Integration)
	if service == nil {
		return nil, nil
	}

	servicePort := corev1.ServicePort{
		Name:       prometheusPortName,
		Port:       int32(t.Port),
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromString(prometheusPortName),
	}

	return service, &servicePort
}

func (t *prometheusTrait) getServiceMonitorFor(e *Environment) (*monitoringv1.ServiceMonitor, error) {
	labels, err := parseCsvMap(&t.ServiceMonitorLabels)
	if err != nil {
		return nil, err
	}
	labels["camel.apache.org/integration"] = e.Integration.Name

	smt := monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceMonitor",
			APIVersion: "monitoring.coreos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Integration.Name,
			Namespace: e.Integration.Namespace,
			Labels:    labels,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"camel.apache.org/integration": e.Integration.Name,
				},
			},
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "prometheus",
				},
			},
		},
	}
	return &smt, nil
}

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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/apache/camel-k/deploy"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
)

// The Prometheus trait configures a Prometheus-compatible endpoint. This trait also exposes the integration with
//`Service` and `ServiceMonitor` resources, so that the endpoint can be scraped automatically, when using the
// Prometheus Operator.
//
// The metrics exposed vary depending on the configured runtime. With the default Quarkus runtime, metrics are
// exposed using MicroProfile Metrics. While with the Java main runtime, metrics are exposed using the Prometheus
// JMX exporter.
//
// WARNING: The creation of the `ServiceMonitor` resource requires the https://github.com/coreos/prometheus-operator[Prometheus Operator]
// custom resource definition to be installed.
// You can set `service-monitor` to `false` for the Prometheus trait to work without the Prometheus Operator.
//
// The Prometheus trait is disabled by default.
//
// +camel-k:trait=prometheus
type prometheusTrait struct {
	BaseTrait `property:",squash"`
	// The Prometheus endpoint port (default `9779`, or `8080` with Quarkus).
	Port *int `property:"port" json:"port,omitempty"`
	// Whether a `ServiceMonitor` resource is created (default `true`).
	ServiceMonitor *bool `property:"service-monitor" json:"serviceMonitor,omitempty"`
	// The `ServiceMonitor` resource labels, applicable when `service-monitor` is `true`.
	ServiceMonitorLabels []string `property:"service-monitor-labels" json:"serviceMonitorLabels,omitempty"`
	// To use a custom ConfigMap containing the Prometheus JMX exporter configuration (under the `content` ConfigMap key).
	// When this property is left empty (default), Camel K generates a standard Prometheus configuration for the integration.
	// It is not applicable when using Quarkus.
	ConfigMap string `property:"configmap" json:"configMap,omitempty"`
}

const (
	prometheusJmxExporterConfigFileName  = "prometheus-jmx-exporter.yaml"
	prometheusJmxExporterConfigMountPath = "/etc/prometheus"
	prometheusPortName                   = "prometheus"
)

func newPrometheusTrait() Trait {
	return &prometheusTrait{
		BaseTrait:      NewBaseTrait("prometheus", 1900),
		ServiceMonitor: util.BoolP(true),
	}
}

func (t *prometheusTrait) Configure(e *Environment) (bool, error) {
	return t.Enabled != nil && *t.Enabled && e.IntegrationInPhase(
		v1.IntegrationPhaseInitialization,
		v1.IntegrationPhaseDeploying,
		v1.IntegrationPhaseRunning,
	), nil
}

func (t *prometheusTrait) Apply(e *Environment) (err error) {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		// Add the Camel Quarkus MP Metrics extension
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "mvn:org.apache.camel.quarkus:camel-quarkus-microprofile-metrics")
	}

	container := e.getIntegrationContainer()
	if container == nil {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionPrometheusAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionContainerNotAvailableReason,
			"",
		)
		return nil
	}

	condition := v1.IntegrationCondition{
		Type:   v1.IntegrationConditionPrometheusAvailable,
		Status: corev1.ConditionTrue,
		Reason: v1.IntegrationConditionPrometheusAvailableReason,
	}

	port := 8080
	if t.Port == nil {
		t.Port = &port
	}

	// Configure the Prometheus container port
	containerPort := t.getContainerPort()
	controller, err := e.DetermineControllerStrategy()
	if err != nil {
		return err
	}
	// Skip declaring the Prometheus port when Knative is enabled, as only one container port is supported
	if controller != ControllerStrategyKnativeService {
		container.Ports = append(container.Ports, *containerPort)
	}
	condition.Message = fmt.Sprintf("%s(%d)", container.Name, containerPort.ContainerPort)

	// Retrieve the service or create a new one if the service trait is enabled
	serviceEnabled := false
	service := e.Resources.GetServiceForIntegration(e.Integration)
	if service == nil {
		trait := e.Catalog.GetTrait(serviceTraitID)
		if serviceTrait, ok := trait.(*serviceTrait); ok {
			serviceEnabled = serviceTrait.isEnabled()
		}
		if serviceEnabled {
			// Add a new service if not already created
			service = getServiceFor(e)
			// Override the service name if none exists.
			// This is required for Knative Serving, that checks no standard eponymous service exist
			service.Name += "-prometheus"
			e.Resources.Add(service)
		}
	} else {
		serviceEnabled = true
	}

	// Add the service port and service monitor resource
	if serviceEnabled {
		servicePort := t.getServicePort()
		service.Spec.Ports = append(service.Spec.Ports, *servicePort)
		condition.Message = fmt.Sprintf("%s(%s/%d) -> ", service.Name, servicePort.Name, servicePort.Port) + condition.Message

		// Add the ServiceMonitor resource
		if util.IsNilOrTrue(t.ServiceMonitor) {
			smt, err := t.getServiceMonitorFor(e)
			if err != nil {
				return err
			}
			e.Resources.Add(smt)
		}
	} else {
		condition.Status = corev1.ConditionFalse
		condition.Reason = v1.IntegrationConditionServiceNotAvailableReason
	}

	e.Integration.Status.SetConditions(condition)

	return nil
}

func (t *prometheusTrait) getContainerPort() *corev1.ContainerPort {
	containerPort := corev1.ContainerPort{
		ContainerPort: int32(*t.Port),
		Protocol:      corev1.ProtocolTCP,
	}
	return &containerPort
}

func (t *prometheusTrait) getServicePort() *corev1.ServicePort {
	servicePort := corev1.ServicePort{
		Name:     prometheusPortName,
		Port:     int32(*t.Port),
		Protocol: corev1.ProtocolTCP,
		// Avoid relying on named port, as Knative enforces specific values used for content negotiation
		TargetPort: intstr.FromInt(*t.Port),
	}
	return &servicePort
}

func (t *prometheusTrait) getServiceMonitorFor(e *Environment) (*monitoringv1.ServiceMonitor, error) {
	labels, err := keyValuePairArrayAsStringMap(t.ServiceMonitorLabels)
	if err != nil {
		return nil, err
	}
	labels[v1.IntegrationLabel] = e.Integration.Name

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
					v1.IntegrationLabel: e.Integration.Name,
				},
			},
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: prometheusPortName,
				},
			},
		},
	}
	return &smt, nil
}

func (t *prometheusTrait) getJmxExporterConfigMapOrAdd(e *Environment) string {
	if t.ConfigMap != "" {
		return t.ConfigMap
	}

	// Add a default config if not specified by the user
	defaultName := e.Integration.Name + "-prometheus"
	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultName,
			Namespace: e.Integration.Namespace,
			Labels: map[string]string{
				v1.IntegrationLabel: e.Integration.Name,
			},
		},
		Data: map[string]string{
			"content": deploy.ResourceAsString("/prometheus-jmx-exporter.yaml"),
		},
	}
	e.Resources.Add(&cm)
	return defaultName
}

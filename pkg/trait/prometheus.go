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
	"k8s.io/utils/pointer"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/util"
)

type prometheusTrait struct {
	BaseTrait
	traitv1.PrometheusTrait `property:",squash"`
}

func newPrometheusTrait() Trait {
	return &prometheusTrait{
		BaseTrait: NewBaseTrait("prometheus", 1900),
		PrometheusTrait: traitv1.PrometheusTrait{
			PodMonitor: pointer.Bool(true),
		},
	}
}

func (t *prometheusTrait) Configure(e *Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, false) {
		return false, nil
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseInitialization) || e.IntegrationInRunningPhases(), nil
}

func (t *prometheusTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		// Add the Camel Quarkus MP Metrics extension
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies,
			"mvn:org.apache.camel.quarkus:camel-quarkus-microprofile-metrics")
		return nil
	}

	container := e.GetIntegrationContainer()
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

	controller, err := e.DetermineControllerStrategy()
	if err != nil {
		return err
	}

	containerPort := e.getIntegrationContainerPort()
	if containerPort == nil {
		containerPort = t.getContainerPort(e, controller)
		container.Ports = append(container.Ports, *containerPort)
	}

	condition.Message = fmt.Sprintf("%s(%d)", container.Name, containerPort.ContainerPort)

	// Add the PodMonitor resource
	if pointer.BoolDeref(t.PodMonitor, false) {
		portName := containerPort.Name
		// Knative defaults to naming the userland container port "user-port".
		// Let's rely on that default, granted it is not officially part of the Knative
		// runtime contract.
		// See https://github.com/knative/specs/blob/main/specs/serving/runtime-contract.md
		if portName == "" && controller == ControllerStrategyKnativeService {
			portName = "user-port"
		}

		podMonitor, err := t.getPodMonitorFor(e, portName)
		if err != nil {
			return err
		}
		e.Resources.Add(podMonitor)
		condition.Message = fmt.Sprintf("PodMonitor (%s) -> ", podMonitor.Name) + condition.Message
	} else {
		condition.Message = "ContainerPort " + condition.Message
	}

	e.Integration.Status.SetConditions(condition)

	return nil
}

func (t *prometheusTrait) getContainerPort(e *Environment, controller ControllerStrategy) *corev1.ContainerPort {
	var name string
	var port int

	if t := e.Catalog.GetTrait(containerTraitID); t != nil {
		if ct, ok := t.(*containerTrait); ok {
			name = ct.PortName
			port = ct.Port
		}
	}

	// Let's rely on Knative default HTTP negotiation
	if name == "" && controller != ControllerStrategyKnativeService {
		name = defaultContainerPortName
	}

	return &corev1.ContainerPort{
		Name:          name,
		ContainerPort: int32(port),
		Protocol:      corev1.ProtocolTCP,
	}
}

func (t *prometheusTrait) getPodMonitorFor(e *Environment, portName string) (*monitoringv1.PodMonitor, error) {
	labels, err := keyValuePairArrayAsStringMap(t.PodMonitorLabels)
	if err != nil {
		return nil, err
	}
	labels[v1.IntegrationLabel] = e.Integration.Name

	podMonitor := monitoringv1.PodMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodMonitor",
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Integration.Name,
			Namespace: e.Integration.Namespace,
			Labels:    labels,
		},
		Spec: monitoringv1.PodMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					v1.IntegrationLabel: e.Integration.Name,
				},
			},
			PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
				{
					Port: portName,
					Path: "/q/metrics",
				},
			},
		},
	}

	return &podMonitor, nil
}

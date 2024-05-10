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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util"
)

const (
	prometheusTraitID    = "prometheus"
	prometheusTraitOrder = 1900
)

type prometheusTrait struct {
	BaseTrait
	traitv1.PrometheusTrait `property:",squash"`
}

func newPrometheusTrait() Trait {
	return &prometheusTrait{
		BaseTrait: NewBaseTrait(prometheusTraitID, prometheusTraitOrder),
		PrometheusTrait: traitv1.PrometheusTrait{
			PodMonitor: pointer.Bool(true),
		},
	}
}

func (t *prometheusTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil || !pointer.BoolDeref(t.Enabled, false) {
		return false, nil, nil
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseInitialization) || e.IntegrationInRunningPhases(), nil, nil
}

func (t *prometheusTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityPrometheus)
	}

	container := e.GetIntegrationContainer()
	if container == nil {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionPrometheusAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionContainerNotAvailableReason,
			"integration container not available",
		)
		return nil
	}

	condition := v1.IntegrationCondition{
		Type:   v1.IntegrationConditionPrometheusAvailable,
		Status: corev1.ConditionTrue,
		Reason: v1.IntegrationConditionPrometheusAvailableReason,
	}

	containerPort := e.getIntegrationContainerPort()
	if containerPort == nil {
		containerPort = e.createContainerPort()
		container.Ports = append(container.Ports, *containerPort)
	}

	condition.Message = fmt.Sprintf("%s(%d)", container.Name, containerPort.ContainerPort)

	// Add the PodMonitor resource
	if pointer.BoolDeref(t.PodMonitor, false) {
		portName := containerPort.Name
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

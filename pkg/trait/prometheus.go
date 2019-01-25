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
	"strconv"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/envvar"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
)

type prometheusTrait struct {
	BaseTrait `property:",squash"`

	Port int `property:"port"`
}

// The Prometheus trait must be executed prior to the deployment trait
// as it mutates environment variables
func newPrometheusTrait() *prometheusTrait {
	return &prometheusTrait{
		BaseTrait: BaseTrait{
			id: ID("prometheus"),
		},
		Port: 9779,
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

	// TODO: update the existing integration service instead of
	// creating an extra service dedicated to Prometheus
	svc := t.getServiceFor(e)
	e.Resources.Add(svc)
	smt := t.getServiceMonitorFor(e)
	e.Resources.Add(smt)
	return nil
}

func (t *prometheusTrait) getServiceFor(e *Environment) *corev1.Service {
	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Integration.Name + "-prometheus",
			Namespace: e.Integration.Namespace,
			Labels: map[string]string{
				"camel.apache.org/integration": e.Integration.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "prometheus",
					Port:     int32(t.Port),
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"camel.apache.org/integration": e.Integration.Name,
			},
		},
	}

	return &svc
}

func (t *prometheusTrait) getServiceMonitorFor(e *Environment) *monitoringv1.ServiceMonitor {
	smt := monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceMonitor",
			APIVersion: "monitoring.coreos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Integration.Name + "-prometheus",
			Namespace: e.Integration.Namespace,
			Labels: map[string]string{
				// TODO: add the ability to configure additional labels
				"camel.apache.org/integration": e.Integration.Name,
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"camel.apache.org/integration": e.Integration.Name,
				},
			},
			Endpoints: []monitoringv1.Endpoint{
				monitoringv1.Endpoint{
					Port: "prometheus",
				},
			},
		},
	}
	return &smt
}

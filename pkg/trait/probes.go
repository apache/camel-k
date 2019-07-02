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
	"sort"
	"strconv"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util"
	"k8s.io/apimachinery/pkg/util/intstr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type probesTrait struct {
	BaseTrait `property:",squash"`

	BindHost                  string `property:"bind-host"`
	BindPort                  int    `property:"bind-port"`
	Path                      string `property:"path"`
	LivenessInitialDelay      int32  `property:"liveness-initial-delay"`
	LivenessTimeout           int32  `property:"liveness-timeout"`
	LivenessPeriod            int32  `property:"liveness-period"`
	LivenessSuccessThreshold  int32  `property:"liveness-success-threshold"`
	LivenessFailureThreshold  int32  `property:"liveness-failure-threshold"`
	ReadinessInitialDelay     int32  `property:"readiness-initial-delay"`
	ReadinessTimeout          int32  `property:"readiness-timeout"`
	ReadinessPeriod           int32  `property:"readiness-period"`
	ReadinessSuccessThreshold int32  `property:"readiness-success-threshold"`
	ReadinessFailureThreshold int32  `property:"readiness-failure-threshold"`
}

func newProbesTrait() *probesTrait {
	return &probesTrait{
		BaseTrait: newBaseTrait("probes"),
		BindHost:  "0.0.0.0",
		BindPort:  8081,
		Path:      "/health",
	}
}

func (t *probesTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && *t.Enabled {
		return e.IntegrationInPhase(v1alpha1.IntegrationPhaseInitialization) || e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
	}

	return false, nil
}

func (t *probesTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1alpha1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "runtime:health")

		// sort the dependencies to get always the same list if they don't change
		sort.Strings(e.Integration.Status.Dependencies)

		e.Integration.Status.Configuration = append(e.Integration.Status.Configuration,
			//
			// TODO: At the moment the servlet engine is used only for health but we need to
			//       have a dedicated servlet trait and maybe an option to create a dedicated
			//       server for management stuffs like health
			//
			v1alpha1.ConfigurationSpec{Type: "property", Value: "customizer.servlet.enabled=true"},
			v1alpha1.ConfigurationSpec{Type: "property", Value: "customizer.servlet.bindHost=" + t.BindHost},
			v1alpha1.ConfigurationSpec{Type: "property", Value: "customizer.servlet.bindPort=" + strconv.Itoa(t.BindPort)},
			v1alpha1.ConfigurationSpec{Type: "property", Value: "customizer.health.enabled=true"},
			v1alpha1.ConfigurationSpec{Type: "property", Value: "customizer.health.path=" + t.Path},
		)
	}

	if e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		e.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
			if len(deployment.Spec.Template.Spec.Containers) != 1 {
				return
			}

			deployment.Spec.Template.Spec.Containers[0].LivenessProbe = t.newLivenessProbe()
			deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = t.newReadinessProbe()
		})
	}

	return nil
}

func (t *probesTrait) newLivenessProbe() *corev1.Probe {
	p := corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Port: intstr.FromInt(t.BindPort),
				Path: t.Path,
			},
		},
	}

	p.InitialDelaySeconds = t.LivenessInitialDelay
	p.TimeoutSeconds = t.LivenessTimeout
	p.PeriodSeconds = t.LivenessPeriod
	p.SuccessThreshold = t.LivenessSuccessThreshold
	p.FailureThreshold = t.LivenessFailureThreshold

	return &p
}

func (t *probesTrait) newReadinessProbe() *corev1.Probe {
	p := corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Port: intstr.FromInt(t.BindPort),
				Path: t.Path,
			},
		},
	}

	p.InitialDelaySeconds = t.ReadinessInitialDelay
	p.TimeoutSeconds = t.ReadinessTimeout
	p.PeriodSeconds = t.ReadinessPeriod
	p.SuccessThreshold = t.ReadinessSuccessThreshold
	p.FailureThreshold = t.ReadinessFailureThreshold

	return &p
}

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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"k8s.io/apimachinery/pkg/util/intstr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// The Probes trait allows to configure Liveness and Readiness probes on the integration container.
//
// +camel-k:trait=probes
type probesTrait struct {
	BaseTrait `property:",squash"`
	// Configures the host on which the probe is exposed (default `0.0.0.0`).
	BindHost string `property:"bind-host"`
	// Configures the port on which the probe is exposed (default `8080`).
	BindPort int `property:"bind-port"`
	// Path to access on the probe ( default `/health`).
	Path string `property:"path"`
	// Number of seconds after the container has started before liveness probes are initiated.
	LivenessInitialDelay int32 `property:"liveness-initial-delay"`
	// Number of seconds after which the probe times out. Applies to the liveness probe.
	LivenessTimeout int32 `property:"liveness-timeout"`
	// How often to perform the probe. Applies to the liveness probe.
	LivenessPeriod int32 `property:"liveness-period"`
	// Minimum consecutive successes for the probe to be considered successful after having failed. Applies to the liveness probe.
	LivenessSuccessThreshold int32 `property:"liveness-success-threshold"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded. Applies to the liveness probe.
	LivenessFailureThreshold int32 `property:"liveness-failure-threshold"`
	// Number of seconds after the container has started before readiness probes are initiated.
	ReadinessInitialDelay int32 `property:"readiness-initial-delay"`
	// Number of seconds after which the probe times out. Applies to the readiness probe.
	ReadinessTimeout int32 `property:"readiness-timeout"`
	// How often to perform the probe. Applies to the readiness probe.
	ReadinessPeriod int32 `property:"readiness-period"`
	// Minimum consecutive successes for the probe to be considered successful after having failed. Applies to the readiness probe.
	ReadinessSuccessThreshold int32 `property:"readiness-success-threshold"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded. Applies to the readiness probe.
	ReadinessFailureThreshold int32 `property:"readiness-failure-threshold"`
}

func newProbesTrait() Trait {
	return &probesTrait{
		BaseTrait: NewBaseTrait("probes", 2100),
		BindHost:  "0.0.0.0",
		BindPort:  8081,
		Path:      "/health",
	}
}

func (t *probesTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && *t.Enabled {
		return e.IntegrationInPhase(
			v1.IntegrationPhaseInitialization,
			v1.IntegrationPhaseDeploying,
			v1.IntegrationPhaseRunning,
		), nil
	}

	return false, nil
}

func (t *probesTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "mvn:org.apache.camel.k/camel-k-runtime-health")

		// sort the dependencies to get always the same list if they don't change
		sort.Strings(e.Integration.Status.Dependencies)

		e.Integration.Status.Configuration = append(e.Integration.Status.Configuration,
			//
			// TODO: At the moment the servlet engine is used only for health but we need to
			//       have a dedicated servlet trait and maybe an option to create a dedicated
			//       server for management stuffs like health
			//
			v1.ConfigurationSpec{Type: "property", Value: "customizer.servlet.enabled=true"},
			v1.ConfigurationSpec{Type: "property", Value: "customizer.servlet.bindHost=" + t.BindHost},
			v1.ConfigurationSpec{Type: "property", Value: "customizer.servlet.bindPort=" + strconv.Itoa(t.BindPort)},
			v1.ConfigurationSpec{Type: "property", Value: "customizer.health.enabled=true"},
			v1.ConfigurationSpec{Type: "property", Value: "customizer.health.path=" + t.Path},
		)
	}

	if e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
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

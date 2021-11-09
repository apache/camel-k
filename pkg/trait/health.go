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
	"encoding/json"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
)

const (
	defaultLivenessProbePath  = "/q/health/live"
	defaultReadinessProbePath = "/q/health/ready"
)

// The health trait is responsible for configuring the health probes on the integration container.
//
// It's disabled by default.
//
// +camel-k:trait=health
type healthTrait struct {
	BaseTrait `property:",squash"`

	// Configures the liveness probe for the integration container (default `false`).
	LivenessProbeEnabled *bool `property:"liveness-probe-enabled" json:"livenessProbeEnabled,omitempty"`
	// Scheme to use when connecting to the liveness probe (default `HTTP`).
	LivenessScheme string `property:"liveness-scheme" json:"livenessScheme,omitempty"`
	// Number of seconds after the container has started before the liveness probe is initiated.
	LivenessInitialDelay int32 `property:"liveness-initial-delay" json:"livenessInitialDelay,omitempty"`
	// Number of seconds after which the liveness probe times out.
	LivenessTimeout int32 `property:"liveness-timeout" json:"livenessTimeout,omitempty"`
	// How often to perform the liveness probe.
	LivenessPeriod int32 `property:"liveness-period" json:"livenessPeriod,omitempty"`
	// Minimum consecutive successes for the liveness probe to be considered successful after having failed.
	LivenessSuccessThreshold int32 `property:"liveness-success-threshold" json:"livenessSuccessThreshold,omitempty"`
	// Minimum consecutive failures for the liveness probe to be considered failed after having succeeded.
	LivenessFailureThreshold int32 `property:"liveness-failure-threshold" json:"livenessFailureThreshold,omitempty"`

	// Configures the readiness probe for the integration container (default `true`).
	ReadinessProbeEnabled *bool `property:"readiness-probe-enabled" json:"readinessProbeEnabled,omitempty"`
	// Scheme to use when connecting to the readiness probe (default `HTTP`).
	ReadinessScheme string `property:"readiness-scheme" json:"readinessScheme,omitempty"`
	// Number of seconds after the container has started before the readiness probe is initiated.
	ReadinessInitialDelay int32 `property:"readiness-initial-delay" json:"readinessInitialDelay,omitempty"`
	// Number of seconds after which the readiness probe times out.
	ReadinessTimeout int32 `property:"readiness-timeout" json:"readinessTimeout,omitempty"`
	// How often to perform the readiness probe.
	ReadinessPeriod int32 `property:"readiness-period" json:"readinessPeriod,omitempty"`
	// Minimum consecutive successes for the readiness probe to be considered successful after having failed.
	ReadinessSuccessThreshold int32 `property:"readiness-success-threshold" json:"readinessSuccessThreshold,omitempty"`
	// Minimum consecutive failures for the readiness probe to be considered failed after having succeeded.
	ReadinessFailureThreshold int32 `property:"readiness-failure-threshold" json:"readinessFailureThreshold,omitempty"`
}

func newHealthTrait() Trait {
	return &healthTrait{
		BaseTrait:       NewBaseTrait("health", 1700),
		LivenessScheme:  string(corev1.URISchemeHTTP),
		ReadinessScheme: string(corev1.URISchemeHTTP),
	}
}

func (t *healthTrait) Configure(e *Environment) (bool, error) {
	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() {
		return false, nil
	}

	if IsNilOrFalse(t.Enabled) {
		// Source the configuration from the container trait to maintain backward compatibility.
		// This can be removed once the deprecated properties related to health probes are actually
		// removed from the container trait.
		if trait := e.Catalog.GetTrait(containerTraitID); trait != nil {
			if container, ok := trait.(*containerTrait); ok && IsNilOrTrue(container.Enabled) && IsTrue(container.DeprecatedProbesEnabled) {
				config, err := json.Marshal(container)
				if err != nil {
					return false, err
				}
				err = json.Unmarshal(config, t)
				if err != nil {
					return false, err
				}
				t.Enabled = BoolP(true)
				t.LivenessProbeEnabled = BoolP(true)
				t.ReadinessProbeEnabled = BoolP(true)
				return true, err
			}
		}
		return false, nil
	}

	return true, nil
}

func (t *healthTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		if capability, ok := e.CamelCatalog.Runtime.Capabilities[v1.CapabilityHealth]; ok {
			for _, dependency := range capability.Dependencies {
				util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, dependency.GetDependencyID())
			}
			// sort the dependencies to get always the same list if they don't change
			sort.Strings(e.Integration.Status.Dependencies)
		}
		return nil
	}

	if IsNilOrFalse(t.LivenessProbeEnabled) && IsFalse(t.ReadinessProbeEnabled) {
		return nil
	}

	container := e.GetIntegrationContainer()
	var port *intstr.IntOrString
	// Use the default named HTTP container port if it exists.
	// For Knative, the Serving webhook is responsible for setting the user-land port,
	// and associating the probes with the corresponding port.
	if containerPort := e.getIntegrationContainerPort(); containerPort != nil && containerPort.Name == defaultContainerPortName {
		p := intstr.FromString(defaultContainerPortName)
		port = &p
	} else if e.GetTrait(knativeServiceTraitID) == nil {
		p := intstr.FromInt(defaultContainerPort)
		port = &p
	}

	if IsTrue(t.LivenessProbeEnabled) {
		container.LivenessProbe = t.newLivenessProbe(port, defaultLivenessProbePath)
	}
	if IsNilOrTrue(t.ReadinessProbeEnabled) {
		container.ReadinessProbe = t.newReadinessProbe(port, defaultReadinessProbePath)
	}

	return nil
}

func (t *healthTrait) newLivenessProbe(port *intstr.IntOrString, path string) *corev1.Probe {
	p := corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   path,
				Scheme: corev1.URIScheme(t.LivenessScheme),
			},
		},
		InitialDelaySeconds: t.LivenessInitialDelay,
		TimeoutSeconds:      t.LivenessTimeout,
		PeriodSeconds:       t.LivenessPeriod,
		SuccessThreshold:    t.LivenessSuccessThreshold,
		FailureThreshold:    t.LivenessFailureThreshold,
	}

	if port != nil {
		p.Handler.HTTPGet.Port = *port
	}

	return &p
}

func (t *healthTrait) newReadinessProbe(port *intstr.IntOrString, path string) *corev1.Probe {
	p := corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   path,
				Scheme: corev1.URIScheme(t.ReadinessScheme),
			},
		},
		InitialDelaySeconds: t.ReadinessInitialDelay,
		TimeoutSeconds:      t.ReadinessTimeout,
		PeriodSeconds:       t.ReadinessPeriod,
		SuccessThreshold:    t.ReadinessSuccessThreshold,
		FailureThreshold:    t.ReadinessFailureThreshold,
	}

	if port != nil {
		p.Handler.HTTPGet.Port = *port
	}

	return &p
}

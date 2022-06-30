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
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
)

const (
	defaultLivenessProbePath  = "/q/health/live"
	defaultReadinessProbePath = "/q/health/ready"
)

type healthTrait struct {
	BaseTrait
	v1.HealthTrait `property:",squash"`
}

func newHealthTrait() Trait {
	return &healthTrait{
		BaseTrait: NewBaseTrait("health", 1700),
		HealthTrait: v1.HealthTrait{
			LivenessScheme:  string(corev1.URISchemeHTTP),
			ReadinessScheme: string(corev1.URISchemeHTTP),
		},
	}
}

func (t *healthTrait) Configure(e *Environment) (bool, error) {
	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() {
		return false, nil
	}

	if !pointer.BoolDeref(t.Enabled, false) {
		// Source the configuration from the container trait to maintain backward compatibility.
		// This can be removed once the deprecated properties related to health probes are actually
		// removed from the container trait.
		if trait := e.Catalog.GetTrait(containerTraitID); trait != nil {
			if container, ok := trait.(*containerTrait); ok && pointer.BoolDeref(container.Enabled, true) && pointer.BoolDeref(container.DeprecatedProbesEnabled, false) {
				config, err := json.Marshal(container)
				if err != nil {
					return false, err
				}
				err = json.Unmarshal(config, t)
				if err != nil {
					return false, err
				}
				t.Enabled = pointer.Bool(true)
				t.LivenessProbeEnabled = pointer.Bool(true)
				t.ReadinessProbeEnabled = pointer.Bool(true)
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

	if !pointer.BoolDeref(t.LivenessProbeEnabled, false) && !pointer.BoolDeref(t.ReadinessProbeEnabled, true) {
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

	if pointer.BoolDeref(t.LivenessProbeEnabled, false) {
		container.LivenessProbe = t.newLivenessProbe(port, defaultLivenessProbePath)
	}
	if pointer.BoolDeref(t.ReadinessProbeEnabled, true) {
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

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
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util"
)

const (
	healthTraitID    = "health"
	healthTraitOrder = 1700

	defaultLivenessProbePath  = "/q/health/live"
	defaultReadinessProbePath = "/q/health/ready"
	defaultStartupProbePath   = "/q/health/started"
)

type healthTrait struct {
	BaseTrait
	traitv1.HealthTrait `property:",squash"`
}

func newHealthTrait() Trait {
	return &healthTrait{
		BaseTrait: NewBaseTrait(healthTraitID, healthTraitOrder),
		HealthTrait: traitv1.HealthTrait{
			LivenessScheme:  string(corev1.URISchemeHTTP),
			ReadinessScheme: string(corev1.URISchemeHTTP),
			StartupScheme:   string(corev1.URISchemeHTTP),
		},
	}
}

func (t *healthTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.CamelCatalog == nil {
		return false, NewIntegrationConditionPlatformDisabledCatalogMissing(), nil
	}
	if e.Integration == nil ||
		!e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	// The trait must be disabled if a debug operation is ongoing
	if jt := e.Catalog.GetTrait(jvmTraitID); jt != nil {
		if jvm, ok := jt.(*jvmTrait); ok && pointer.BoolDeref(jvm.Debug, false) {
			return false, NewIntegrationConditionPlatformDisabledWithMessage("Health", "debug operation ongoing: incompatible with health checks"), nil
		}
	}

	return pointer.BoolDeref(t.Enabled, true), nil, nil
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

	if !pointer.BoolDeref(t.LivenessProbeEnabled, false) && !pointer.BoolDeref(t.ReadinessProbeEnabled, true) && !pointer.BoolDeref(t.StartupProbeEnabled, false) {
		return nil
	}

	container := e.GetIntegrationContainer()
	if container == nil {
		return fmt.Errorf("unable to find integration container: %s", e.Integration.Name)
	}

	var port *intstr.IntOrString
	containerPort := e.getIntegrationContainerPort()
	if containerPort == nil {
		containerPort = e.createContainerPort()
	}
	p := intstr.FromInt32(containerPort.ContainerPort)
	port = &p

	if e.CamelCatalog.Runtime.Capabilities["health"].Metadata != nil {
		t.setCatalogConfiguration(container, port, e.CamelCatalog.Runtime.Capabilities["health"].Metadata)
	} else {
		t.setDefaultConfiguration(container, port)
	}

	return nil
}

func (t *healthTrait) setCatalogConfiguration(container *corev1.Container, port *intstr.IntOrString, metadata map[string]string) {
	if pointer.BoolDeref(t.LivenessProbeEnabled, false) {
		container.LivenessProbe = t.newLivenessProbe(port, metadata["defaultLivenessProbePath"])
	}
	if pointer.BoolDeref(t.ReadinessProbeEnabled, true) {
		container.ReadinessProbe = t.newReadinessProbe(port, metadata["defaultReadinessProbePath"])
	}
	if pointer.BoolDeref(t.StartupProbeEnabled, false) {
		container.StartupProbe = t.newStartupProbe(port, metadata["defaultStartupProbePath"])
	}
}

// Deprecated: to be removed in future release in favor of func setCatalogConfiguration().
func (t *healthTrait) setDefaultConfiguration(container *corev1.Container, port *intstr.IntOrString) {
	if pointer.BoolDeref(t.LivenessProbeEnabled, false) {
		container.LivenessProbe = t.newLivenessProbe(port, defaultLivenessProbePath)
	}
	if pointer.BoolDeref(t.ReadinessProbeEnabled, true) {
		container.ReadinessProbe = t.newReadinessProbe(port, defaultReadinessProbePath)
	}
	if pointer.BoolDeref(t.StartupProbeEnabled, false) {
		container.StartupProbe = t.newStartupProbe(port, defaultStartupProbePath)
	}
}

func (t *healthTrait) newLivenessProbe(port *intstr.IntOrString, path string) *corev1.Probe {
	p := corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
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
		p.ProbeHandler.HTTPGet.Port = *port
	}

	return &p
}

func (t *healthTrait) newReadinessProbe(port *intstr.IntOrString, path string) *corev1.Probe {
	p := corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
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
		p.ProbeHandler.HTTPGet.Port = *port
	}

	return &p
}

func (t *healthTrait) newStartupProbe(port *intstr.IntOrString, path string) *corev1.Probe {
	p := corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   path,
				Scheme: corev1.URIScheme(t.StartupScheme),
			},
		},
		InitialDelaySeconds: t.StartupInitialDelay,
		TimeoutSeconds:      t.StartupTimeout,
		PeriodSeconds:       t.StartupPeriod,
		SuccessThreshold:    t.StartupSuccessThreshold,
		FailureThreshold:    t.StartupFailureThreshold,
	}

	if port != nil {
		p.ProbeHandler.HTTPGet.Port = *port
	}

	return &p
}

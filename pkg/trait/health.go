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
	"k8s.io/utils/ptr"

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
	}
}

func (t *healthTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil ||
		(!e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases()) ||
		!ptr.Deref(t.Enabled, false) {
		return false, nil, nil
	}

	// The trait must be disabled if a debug operation is ongoing
	if jt := e.Catalog.GetTrait(jvmTraitID); jt != nil {
		if jvm, ok := jt.(*jvmTrait); ok && ptr.Deref(jvm.Debug, false) {
			return false, NewIntegrationConditionPlatformDisabledWithMessage("Health", "debug operation ongoing: incompatible with health checks"), nil
		}
	}

	t.setProbesValues(e)

	return true, nil, nil
}

func (t *healthTrait) setProbesValues(e *Environment) {
	if t.LivenessProbe == "" {
		if e.CamelCatalog.Runtime.Capabilities["health"].Metadata != nil {
			t.LivenessProbe = e.CamelCatalog.Runtime.Capabilities["health"].Metadata["defaultLivenessProbePath"]
		} else {
			// Deprecated: to be removed
			t.LivenessProbe = defaultLivenessProbePath
		}
	}
	if t.ReadinessProbe == "" {
		if e.CamelCatalog.Runtime.Capabilities["health"].Metadata != nil {
			t.ReadinessProbe = e.CamelCatalog.Runtime.Capabilities["health"].Metadata["defaultReadinessProbePath"]
		} else {
			// Deprecated: to be removed
			t.ReadinessProbe = defaultReadinessProbePath
		}
	}
	if t.StartupProbe == "" {
		if e.CamelCatalog.Runtime.Capabilities["health"].Metadata != nil {
			t.StartupProbe = e.CamelCatalog.Runtime.Capabilities["health"].Metadata["defaultStartupProbePath"]
		} else {
			// Deprecated: to be removed
			t.StartupProbe = defaultStartupProbePath
		}
	}
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

	if !ptr.Deref(t.LivenessProbeEnabled, false) && !ptr.Deref(t.ReadinessProbeEnabled, true) && !ptr.Deref(t.StartupProbeEnabled, false) {
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

	return t.setProbes(container, port)
}

func (t *healthTrait) setProbes(container *corev1.Container, port *intstr.IntOrString) error {
	if ptr.Deref(t.LivenessProbeEnabled, false) {
		if t.LivenessProbe == "" {
			return fmt.Errorf("you need to configure a liveness probe explicitly or in your catalog")
		}
		container.LivenessProbe = t.newLivenessProbe(port, t.LivenessProbe)
	}
	if ptr.Deref(t.ReadinessProbeEnabled, true) {
		if t.ReadinessProbe == "" {
			return fmt.Errorf("you need to configure a readiness probe explicitly or in your catalog")
		}
		container.ReadinessProbe = t.newReadinessProbe(port, t.ReadinessProbe)
	}
	if ptr.Deref(t.StartupProbeEnabled, false) {
		if t.StartupProbe == "" {
			return fmt.Errorf("you need to configure a startup probe explicitly or in your catalog")
		}
		container.StartupProbe = t.newStartupProbe(port, t.StartupProbe)
	}

	return nil
}

func (t *healthTrait) getLivenessScheme() corev1.URIScheme {
	if t.LivenessScheme == "" {
		return corev1.URISchemeHTTP
	}

	return corev1.URIScheme(t.LivenessScheme)
}

func (t *healthTrait) getReadinessScheme() corev1.URIScheme {
	if t.ReadinessScheme == "" {
		return corev1.URISchemeHTTP
	}

	return corev1.URIScheme(t.ReadinessScheme)
}

func (t *healthTrait) getStartupScheme() corev1.URIScheme {
	if t.StartupScheme == "" {
		return corev1.URISchemeHTTP
	}

	return corev1.URIScheme(t.StartupScheme)
}

func (t *healthTrait) newLivenessProbe(port *intstr.IntOrString, path string) *corev1.Probe {
	p := corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   path,
				Scheme: t.getLivenessScheme(),
				Port:   *t.getLivenessPort(port),
			},
		},
		InitialDelaySeconds: t.LivenessInitialDelay,
		TimeoutSeconds:      t.LivenessTimeout,
		PeriodSeconds:       t.LivenessPeriod,
		SuccessThreshold:    t.LivenessSuccessThreshold,
		FailureThreshold:    t.LivenessFailureThreshold,
	}

	return &p
}

func (t *healthTrait) newReadinessProbe(port *intstr.IntOrString, path string) *corev1.Probe {
	p := corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   path,
				Scheme: t.getReadinessScheme(),
				Port:   *t.getReadinessPort(port),
			},
		},
		InitialDelaySeconds: t.ReadinessInitialDelay,
		TimeoutSeconds:      t.ReadinessTimeout,
		PeriodSeconds:       t.ReadinessPeriod,
		SuccessThreshold:    t.ReadinessSuccessThreshold,
		FailureThreshold:    t.ReadinessFailureThreshold,
	}

	return &p
}

func (t *healthTrait) newStartupProbe(port *intstr.IntOrString, path string) *corev1.Probe {
	p := corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   path,
				Scheme: t.getStartupScheme(),
				Port:   *t.getStartupPort(port),
			},
		},
		InitialDelaySeconds: t.StartupInitialDelay,
		TimeoutSeconds:      t.StartupTimeout,
		PeriodSeconds:       t.StartupPeriod,
		SuccessThreshold:    t.StartupSuccessThreshold,
		FailureThreshold:    t.StartupFailureThreshold,
	}

	return &p
}

func (t *healthTrait) getLivenessPort(port *intstr.IntOrString) *intstr.IntOrString {
	if t.LivenessPort != 0 {
		livenessPort := intstr.FromInt32(t.LivenessPort)

		return &livenessPort
	}

	return port
}

func (t *healthTrait) getReadinessPort(port *intstr.IntOrString) *intstr.IntOrString {
	if t.ReadinessPort != 0 {
		readinessPort := intstr.FromInt32(t.ReadinessPort)

		return &readinessPort
	}

	return port
}

func (t *healthTrait) getStartupPort(port *intstr.IntOrString) *intstr.IntOrString {
	if t.StartupPort != 0 {
		startupPort := intstr.FromInt32(t.StartupPort)

		return &startupPort
	}

	return port
}

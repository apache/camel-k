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
	"errors"
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

	defaultQuarkusBasePath           = "/q/health"
	defaultQuarkusLivenessProbePath  = defaultQuarkusBasePath + "/live"
	defaultQuarkusReadinessProbePath = defaultQuarkusBasePath + "/ready"
	defaultQuarkusStartupProbePath   = defaultQuarkusBasePath + "/started"
	defaultQuarkusHealthPort         = int32(8080)

	defaultObsSvcBasePath           = "/observe/health"
	defaultObsSvcLivenessProbePath  = defaultObsSvcBasePath + "/live"
	defaultObsSvcReadinessProbePath = defaultObsSvcBasePath + "/ready"
	defaultObsSvcStartupProbePath   = defaultObsSvcBasePath + ""
	defaultObsSvcHealthPort         = int32(9876)

	// We need to be a bit more generous with startup
	// failure. A jvm mode app can take more than 30 secs
	// to be started up.
	defaultStartupFailureThreshold = 6
)

type healthTrait struct {
	BaseTrait
	traitv1.HealthTrait `property:",squash"`

	// Probes port and base path can be also configured via user properties
	probesPort          int32
	probesLivenessPath  string
	probesReadinessPath string
	probesStartupPath   string
}

func newHealthTrait() Trait {
	return &healthTrait{
		BaseTrait: NewBaseTrait(healthTraitID, healthTraitOrder),
	}
}

func (t *healthTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil ||
		(!e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases()) ||
		!ptr.Deref(t.Enabled, true) {
		return false, nil, nil
	}

	// The trait used to be disabled by default in older Camel K runtime
	if e.CamelCatalog != nil && e.CamelCatalog.Runtime.Provider == v1.RuntimeProviderQuarkus && !ptr.Deref(t.Enabled, false) {
		return false, nil, nil
	}

	// Skip if self managed build, unless enabled on purpose
	if !e.Integration.IsManagedBuild() && !ptr.Deref(t.Enabled, false) {
		return false, NewIntegrationConditionPlatformDisabledWithMessage("Health",
			"self managed build: container probes disabled, you can turn explicitly on"), nil
	}

	// Skip if built from git, unless enabled on purpose
	if e.Integration.IsGitBuild() && !ptr.Deref(t.Enabled, false) {
		return false, NewIntegrationConditionPlatformDisabledWithMessage("Health",
			"build from Git: container probes disabled, you can turn explicitly on"), nil
	}

	// The trait must be disabled if a debug operation is ongoing
	if jt := e.Catalog.GetTrait(jvmTraitID); jt != nil {
		if jvm, ok := jt.(*jvmTrait); ok && ptr.Deref(jvm.Debug, false) {
			return false, NewIntegrationConditionPlatformDisabledWithMessage("Health",
				"debug operation ongoing: incompatible with health checks"), nil
		}
	}

	t.setProbesValues(e)

	return ptr.Deref(t.Enabled, true), nil, nil
}

func (t *healthTrait) setProbesValues(e *Environment) {
	// values are taken by trait configuration as priority
	t.probesLivenessPath = t.LivenessProbe
	t.probesReadinessPath = t.ReadinessProbe
	t.probesStartupPath = t.StartupProbe
	// Default is 8080 anyway
	t.probesPort = defaultQuarkusHealthPort

	if e.CamelCatalog == nil {
		// Likely it is a self managed build integration
		return
	}
	isOlderCamelKRuntime := e.CamelCatalog.Runtime.Provider == v1.RuntimeProviderQuarkus
	t.probesPort = defaultObsSvcHealthPort
	if isOlderCamelKRuntime {
		t.probesPort = defaultQuarkusHealthPort
	}

	if t.probesLivenessPath == "" {
		if isOlderCamelKRuntime {
			t.probesLivenessPath = defaultQuarkusLivenessProbePath
		} else {
			t.probesLivenessPath = defaultObsSvcLivenessProbePath
		}
	}

	if t.probesReadinessPath == "" {
		if isOlderCamelKRuntime {
			t.probesReadinessPath = defaultQuarkusReadinessProbePath
		} else {
			t.probesReadinessPath = defaultObsSvcReadinessProbePath
		}
	}

	if t.probesStartupPath == "" {
		if isOlderCamelKRuntime {
			t.probesStartupPath = defaultQuarkusStartupProbePath
		} else {
			t.probesStartupPath = defaultObsSvcStartupProbePath
		}
	}
}

func (t *healthTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		// Execute this only for the old deprecated provider in order to
		// maintain backward compatibility
		if e.CamelCatalog != nil && e.CamelCatalog.Runtime.Provider == v1.RuntimeProviderQuarkus {
			addCamelHealthCapabilityDependency(e)
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

	p := intstr.FromInt32(t.probesPort)

	return t.setProbes(container, &p)
}

func addCamelHealthCapabilityDependency(e *Environment) {
	if capability, ok := e.CamelCatalog.Runtime.Capabilities[v1.CapabilityHealth]; ok {
		for _, dependency := range capability.Dependencies {
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, dependency.GetDependencyID())
		}
		// sort the dependencies to get always the same list if they don't change
		sort.Strings(e.Integration.Status.Dependencies)
	}
}

func (t *healthTrait) setProbes(container *corev1.Container, port *intstr.IntOrString) error {
	if ptr.Deref(t.LivenessProbeEnabled, false) {
		if t.probesLivenessPath == "" {
			return errors.New("you need to configure a liveness probe explicitly or in your catalog")
		}
		container.LivenessProbe = t.newLivenessProbe(port, t.probesLivenessPath)
	}
	if ptr.Deref(t.ReadinessProbeEnabled, true) {
		if t.probesReadinessPath == "" {
			return errors.New("you need to configure a readiness probe explicitly or in your catalog")
		}
		container.ReadinessProbe = t.newReadinessProbe(port, t.probesReadinessPath)
	}
	if ptr.Deref(t.StartupProbeEnabled, false) {
		if t.probesStartupPath == "" {
			return errors.New("you need to configure a startup probe explicitly or in your catalog")
		}
		container.StartupProbe = t.newStartupProbe(port, t.probesStartupPath)
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
	return newProbe(
		path,
		t.getLivenessScheme(),
		t.getLivenessPort(port),
		t.LivenessInitialDelay,
		t.LivenessTimeout,
		t.LivenessPeriod,
		t.LivenessSuccessThreshold,
		t.LivenessFailureThreshold,
	)
}

func (t *healthTrait) newReadinessProbe(port *intstr.IntOrString, path string) *corev1.Probe {
	return newProbe(
		path,
		t.getReadinessScheme(),
		t.getReadinessPort(port),
		t.ReadinessInitialDelay,
		t.ReadinessTimeout,
		t.ReadinessPeriod,
		t.ReadinessSuccessThreshold,
		t.ReadinessFailureThreshold,
	)
}

func (t *healthTrait) newStartupProbe(port *intstr.IntOrString, path string) *corev1.Probe {
	startupFailureThreshold := t.StartupFailureThreshold
	if startupFailureThreshold == 0 {
		startupFailureThreshold = defaultStartupFailureThreshold
	}

	return newProbe(
		path,
		t.getStartupScheme(),
		t.getStartupPort(port),
		t.StartupInitialDelay,
		t.StartupTimeout,
		t.StartupPeriod,
		t.StartupSuccessThreshold,
		startupFailureThreshold,
	)
}

func newProbe(
	path string,
	scheme corev1.URIScheme,
	resolverPort *intstr.IntOrString,
	initialDelay, timeout, period, success, failure int32,
) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   path,
				Scheme: scheme,
				Port:   *resolverPort,
			},
		},
		InitialDelaySeconds: initialDelay,
		TimeoutSeconds:      timeout,
		PeriodSeconds:       period,
		SuccessThreshold:    success,
		FailureThreshold:    failure,
	}
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

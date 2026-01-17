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

// The health trait is responsible for configuring the health probes on the integration container.
//
// NOTE: this trait is disabled by default.
//
// +camel-k:trait=health.
//
//nolint:godoclint
type HealthTrait struct {
	Trait `json:",inline" property:",squash"`

	// Configures the liveness probe for the integration container (default `false`).
	LivenessProbeEnabled *bool `json:"livenessProbeEnabled,omitempty" property:"liveness-probe-enabled"`
	// Scheme to use when connecting to the liveness probe (default `HTTP`).
	LivenessScheme string `json:"livenessScheme,omitempty" property:"liveness-scheme"`
	// Number of seconds after the container has started before the liveness probe is initiated.
	LivenessInitialDelay int32 `json:"livenessInitialDelay,omitempty" property:"liveness-initial-delay"`
	// Number of seconds after which the liveness probe times out.
	LivenessTimeout int32 `json:"livenessTimeout,omitempty" property:"liveness-timeout"`
	// How often to perform the liveness probe.
	LivenessPeriod int32 `json:"livenessPeriod,omitempty" property:"liveness-period"`
	// Minimum consecutive successes for the liveness probe to be considered successful after having failed.
	LivenessSuccessThreshold int32 `json:"livenessSuccessThreshold,omitempty" property:"liveness-success-threshold"`
	// Minimum consecutive failures for the liveness probe to be considered failed after having succeeded.
	LivenessFailureThreshold int32 `json:"livenessFailureThreshold,omitempty" property:"liveness-failure-threshold"`
	// The liveness probe path to use (default provided by the Catalog runtime used).
	LivenessProbe string `json:"livenessProbe,omitempty" property:"liveness-probe"`
	// The liveness port to use (default 8080).
	LivenessPort int32 `json:"livenessPort,omitempty" property:"liveness-port"`

	// Configures the readiness probe for the integration container (default `true`).
	ReadinessProbeEnabled *bool `json:"readinessProbeEnabled,omitempty" property:"readiness-probe-enabled"`
	// Scheme to use when connecting to the readiness probe (default `HTTP`).
	ReadinessScheme string `json:"readinessScheme,omitempty" property:"readiness-scheme"`
	// Number of seconds after the container has started before the readiness probe is initiated.
	ReadinessInitialDelay int32 `json:"readinessInitialDelay,omitempty" property:"readiness-initial-delay"`
	// Number of seconds after which the readiness probe times out.
	ReadinessTimeout int32 `json:"readinessTimeout,omitempty" property:"readiness-timeout"`
	// How often to perform the readiness probe.
	ReadinessPeriod int32 `json:"readinessPeriod,omitempty" property:"readiness-period"`
	// Minimum consecutive successes for the readiness probe to be considered successful after having failed.
	ReadinessSuccessThreshold int32 `json:"readinessSuccessThreshold,omitempty" property:"readiness-success-threshold"`
	// Minimum consecutive failures for the readiness probe to be considered failed after having succeeded.
	ReadinessFailureThreshold int32 `json:"readinessFailureThreshold,omitempty" property:"readiness-failure-threshold"`
	// The readiness probe path to use (default provided by the Catalog runtime used).
	ReadinessProbe string `json:"readinessProbe,omitempty" property:"readiness-probe"`
	// The readiness port to use (default 8080).
	ReadinessPort int32 `json:"readinessPort,omitempty" property:"readiness-port"`

	// Configures the startup probe for the integration container (default `false`).
	StartupProbeEnabled *bool `json:"startupProbeEnabled,omitempty" property:"startup-probe-enabled"`
	// Scheme to use when connecting to the startup probe (default `HTTP`).
	StartupScheme string `json:"startupScheme,omitempty" property:"startup-scheme"`
	// Number of seconds after the container has started before the startup probe is initiated.
	StartupInitialDelay int32 `json:"startupInitialDelay,omitempty" property:"startup-initial-delay"`
	// Number of seconds after which the startup probe times out.
	StartupTimeout int32 `json:"startupTimeout,omitempty" property:"startup-timeout"`
	// How often to perform the startup probe.
	StartupPeriod int32 `json:"startupPeriod,omitempty" property:"startup-period"`
	// Minimum consecutive successes for the startup probe to be considered successful after having failed.
	StartupSuccessThreshold int32 `json:"startupSuccessThreshold,omitempty" property:"startup-success-threshold"`
	// Minimum consecutive failures for the startup probe to be considered failed after having succeeded.
	StartupFailureThreshold int32 `json:"startupFailureThreshold,omitempty" property:"startup-failure-threshold"`
	// The startup probe path to use (default provided by the Catalog runtime used).
	StartupProbe string `json:"startupProbe,omitempty" property:"startup-probe"`
	// The startup port to use (default 8080).
	StartupPort int32 `json:"startupPort,omitempty" property:"startup-port"`
}

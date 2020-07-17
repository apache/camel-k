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

package v1

// The Container trait can be used to configure properties of the container where the integration will run.
//
// It also provides configuration for Services associated to the container.
//
// +camel-k:trait=container
type ContainerTrait struct {
	Trait `property:",squash" json:",inline"`
	// The main container name. It's named `integration` by default.
	Name string `property:"name" json:"name,omitempty"`
	Auto *bool  `property:"auto" json:"auto,omitempty"`

	// The minimum amount of CPU required.
	RequestCPU string `property:"request-cpu" json:"requestCPU,omitempty"`
	// The minimum amount of memory required.
	RequestMemory string `property:"request-memory" json:"requestMemory,omitempty"`
	// The maximum amount of CPU required.
	LimitCPU string `property:"limit-cpu" json:"limitCPU,omitempty"`
	// The maximum amount of memory required.
	LimitMemory string `property:"limit-memory" json:"limitMemory,omitempty"`

	// Can be used to enable/disable exposure via kubernetes Service.
	Expose *bool `property:"expose" json:"expose,omitempty"`
	// To configure a different port exposed by the container (default `8080`).
	Port int `property:"port" json:"port,omitempty"`
	// To configure a different port name for the port exposed by the container (default `http`).
	PortName string `property:"port-name" json:"portName,omitempty"`
	// To configure under which service port the container port is to be exposed (default `80`).
	ServicePort int `property:"service-port" json:"servicePort,omitempty"`
	// To configure under which service port name the container port is to be exposed (default `http`).
	ServicePortName string `property:"service-port-name" json:"servicePortName,omitempty"`

	// ProbesEnabled enable/disable probes on the container (default `false`)
	ProbesEnabled bool `property:"probes-enabled" json:"probesEnabled,omitempty"`
	// Path to access on the probe ( default `/health`). Note that this property is not supported
	// on quarkus runtime and setting it will result in the integration failing to start.
	ProbePath string `property:"probe-path" json:"probePath,omitempty"`
	// Number of seconds after the container has started before liveness probes are initiated.
	LivenessInitialDelay int32 `property:"liveness-initial-delay" json:"livenessInitialDelay,omitempty"`
	// Number of seconds after which the probe times out. Applies to the liveness probe.
	LivenessTimeout int32 `property:"liveness-timeout" json:"livenessTimeout,omitempty"`
	// How often to perform the probe. Applies to the liveness probe.
	LivenessPeriod int32 `property:"liveness-period" json:"livenessPeriod,omitempty"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	// Applies to the liveness probe.
	LivenessSuccessThreshold int32 `property:"liveness-success-threshold" json:"livenessSuccessThreshold,omitempty"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	// Applies to the liveness probe.
	LivenessFailureThreshold int32 `property:"liveness-failure-threshold" json:"livenessFailureThreshold,omitempty"`
	// Number of seconds after the container has started before readiness probes are initiated.
	ReadinessInitialDelay int32 `property:"readiness-initial-delay" json:"readinessInitialDelay,omitempty"`
	// Number of seconds after which the probe times out. Applies to the readiness probe.
	ReadinessTimeout int32 `property:"readiness-timeout" json:"readinessTimeout,omitempty"`
	// How often to perform the probe. Applies to the readiness probe.
	ReadinessPeriod int32 `property:"readiness-period" json:"readinessPeriod,omitempty"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	// Applies to the readiness probe.
	ReadinessSuccessThreshold int32 `property:"readiness-success-threshold" json:"readinessSuccessThreshold,omitempty"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	// Applies to the readiness probe.
	ReadinessFailureThreshold int32 `property:"readiness-failure-threshold" json:"readinessFailureThreshold,omitempty"`
}

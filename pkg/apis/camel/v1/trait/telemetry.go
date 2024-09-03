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

// The Telemetry trait can be used to automatically publish tracing information to an OTLP compatible collector.
//
// The trait is able to automatically discover the telemetry OTLP endpoint available in the namespace (supports **Jaerger** in version 1.35+).
//
// The Telemetry trait is disabled by default.
//
// WARNING: The Telemetry trait can't be enabled at the same time as the Tracing trait.
//
// +camel-k:trait=telemetry.
type TelemetryTrait struct {
	Trait `property:",squash" json:",inline"`
	// Enables automatic configuration of the trait, including automatic discovery of the telemetry endpoint.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// The name of the service that publishes telemetry data (defaults to the integration name)
	ServiceName string `property:"service-name" json:"serviceName,omitempty"`
	// The target endpoint of the Telemetry service (automatically discovered by default)
	Endpoint string `property:"endpoint" json:"endpoint,omitempty"`
	// The sampler of the telemetry used for tracing (default "on")
	Sampler string `property:"sampler" json:"sampler,omitempty"`
	// The sampler ratio of the telemetry used for tracing
	SamplerRatio string `property:"sampler-ratio" json:"sampler-ratio,omitempty"`
	// The sampler of the telemetry used for tracing is parent based (default "true")
	SamplerParentBased *bool `property:"sampler-parent-based" json:"sampler-parent-based,omitempty"`
}

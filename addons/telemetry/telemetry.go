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

package telemetry

import (
	"k8s.io/utils/pointer"

	"github.com/apache/camel-k/addons/telemetry/discovery"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
)

// The Telemetry trait can be used to automatically publish tracing information to an OTLP compatible collector.
//
// The trait is able to automatically discover the telemetry OTLP endpoint available in the namespace (supports **Jaerger** in version 1.35+).
//
// The Telemetry trait is disabled by default.
//
// WARNING: The Telemetry trait can't be enabled at the same time as the Tracing trait.
//
// +camel-k:trait=telemetry.
type Trait struct {
	traitv1.Trait `property:",squash" json:",inline"`
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

type telemetryTrait struct {
	trait.BaseTrait
	Trait `property:",squash"`
}

const (
	propEnabled            = "propEnabled"
	propEndpoint           = "propEndpoint"
	propServiceName        = "propServiceName"
	propSampler            = "propSampler"
	propSamplerRatio       = "propSamplerRatio"
	propSamplerParentBased = "propSamplerParentBased"
)

var (
	telemetryProperties = map[v1.RuntimeProvider]map[string]string{
		v1.RuntimeProviderQuarkus: {
			propEndpoint:           "quarkus.opentelemetry.tracer.exporter.otlp.endpoint",
			propServiceName:        "quarkus.opentelemetry.tracer.resource-attributes",
			propSampler:            "quarkus.opentelemetry.tracer.sampler",
			propSamplerRatio:       "quarkus.opentelemetry.tracer.sampler.ratio",
			propSamplerParentBased: "quarkus.opentelemetry.tracer.sampler.parent-based",
		},
	}
)

// NewTelemetryTrait instance the telemetry trait as a BaseTrait capable to inject quarkus properties.
func NewTelemetryTrait() trait.Trait {
	return &telemetryTrait{
		BaseTrait: trait.NewBaseTrait("telemetry", trait.TraitOrderBeforeControllerCreation),
	}
}

func (t *telemetryTrait) Configure(e *trait.Environment) (bool, error) {
	if e.Integration == nil || !pointer.BoolDeref(t.Enabled, false) {
		return false, nil
	}

	if pointer.BoolDeref(t.Auto, true) {
		if t.Endpoint == "" {
			for _, locator := range discovery.TelemetryLocators {
				endpoint, err := locator.FindEndpoint(e.Ctx, t.Client, t.L, e)
				if err != nil {
					return false, err
				}
				if endpoint != "" {
					t.L.Infof("Using tracing endpoint: %s", endpoint)
					t.Endpoint = endpoint
					break
				}
			}
		}

		if t.ServiceName == "" {
			t.ServiceName = e.Integration.Name
		}

		if t.Sampler == "" {
			t.Sampler = "on"
		}
	}

	return true, nil
}

func (t *telemetryTrait) Apply(e *trait.Environment) error {
	util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityTelemetry)

	if e.CamelCatalog != nil {
		provider := e.CamelCatalog.CamelCatalogSpec.Runtime.Provider
		properties := telemetryProperties[provider]

		if appPropEnabled := properties[propEnabled]; appPropEnabled != "" {
			e.ApplicationProperties[appPropEnabled] = "true"
		}

		if appPropEndpoint := properties[propEndpoint]; appPropEndpoint != "" && t.Endpoint != "" {
			e.ApplicationProperties[appPropEndpoint] = t.Endpoint
		}

		if appPropServiceName := properties[propServiceName]; appPropServiceName != "" && t.ServiceName != "" {
			e.ApplicationProperties[appPropServiceName] = "service.name=" + t.ServiceName
		}

		if appPropSampler := properties[propSampler]; appPropSampler != "" && t.Sampler != "" {
			e.ApplicationProperties[appPropSampler] = t.Sampler
		}

		if appPropSamplerRatio := properties[propSamplerRatio]; appPropSamplerRatio != "" && t.SamplerRatio != "" {
			e.ApplicationProperties[appPropSamplerRatio] = t.SamplerRatio
		}

		if appPropSamplerParentBased := properties[propSamplerParentBased]; appPropSamplerParentBased != "" {
			if pointer.BoolDeref(t.SamplerParentBased, true) {
				e.ApplicationProperties[appPropSamplerParentBased] = "true"
			} else {
				e.ApplicationProperties[appPropSamplerParentBased] = "false"
			}
		}

	}

	return nil
}

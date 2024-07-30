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
	"fmt"

	"github.com/apache/camel-k/v2/pkg/util/boolean"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/apache/camel-k/v2/addons/telemetry/discovery"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util"
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

func (t *telemetryTrait) isForcefullyEnabled() bool {
	return ptr.Deref(t.Enabled, false) && !ptr.Deref(t.Auto, true)
}

func (t *telemetryTrait) Configure(e *trait.Environment) (bool, *trait.TraitCondition, error) {
	if e.Integration == nil || !ptr.Deref(t.Enabled, false) {
		return false, nil, nil
	}
	if e.CamelCatalog == nil && !t.isForcefullyEnabled() {
		return false, trait.NewIntegrationConditionPlatformDisabledCatalogMissing(), nil
	}

	if !ptr.Deref(t.Auto, true) {
		return true, nil, nil
	}

	var condition *trait.TraitCondition

	if t.Endpoint == "" {
		for _, locator := range discovery.TelemetryLocators {
			endpoint, err := locator.FindEndpoint(e.Ctx, t.Client, t.L, e)
			if err != nil {
				return false, nil, err
			}
			if endpoint != "" {
				t.L.Infof("Using tracing endpoint: %s", endpoint)
				condition = trait.NewIntegrationCondition(
					"Telemetry",
					v1.IntegrationConditionTraitInfo,
					corev1.ConditionTrue,
					"TracingEndpoint",
					endpoint,
				)
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

	return true, condition, nil
}

func (t *telemetryTrait) Apply(e *trait.Environment) error {
	util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityTelemetry)

	if t.isForcefullyEnabled() || e.CamelCatalog.Runtime.Capabilities["telemetry"].RuntimeProperties != nil {
		t.setCatalogConfiguration(e)
	} else {
		t.setRuntimeProviderProperties(e)
	}

	return nil
}

func (t *telemetryTrait) setCatalogConfiguration(e *trait.Environment) {
	if e.ApplicationProperties == nil {
		e.ApplicationProperties = make(map[string]string)
	}

	if t.Endpoint != "" {
		e.ApplicationProperties["camel.k.telemetry.endpoint"] = t.Endpoint
	}
	if t.ServiceName != "" {
		e.ApplicationProperties["camel.k.telemetry.serviceName"] = fmt.Sprintf("service.name=%s", t.ServiceName)
	}
	if t.Sampler != "" {
		e.ApplicationProperties["camel.k.telemetry.sampler"] = t.Sampler
	}
	if t.SamplerRatio != "" {
		e.ApplicationProperties["camel.k.telemetry.samplerRatio"] = t.SamplerRatio
	}
	if ptr.Deref(t.SamplerParentBased, true) {
		e.ApplicationProperties["camel.k.telemetry.samplerParentBased"] = boolean.TrueString
	} else {
		e.ApplicationProperties["camel.k.telemetry.samplerParentBased"] = boolean.FalseString
	}

	if e.CamelCatalog != nil && e.CamelCatalog.Runtime.Capabilities["telemetry"].RuntimeProperties != nil {
		for _, cp := range e.CamelCatalog.Runtime.Capabilities["telemetry"].RuntimeProperties {
			e.ApplicationProperties[trait.CapabilityPropertyKey(cp.Key, e.ApplicationProperties)] = cp.Value
		}
	}
}

// Deprecated: to be removed in future release in favor of func setCatalogConfiguration().
func (t *telemetryTrait) setRuntimeProviderProperties(e *trait.Environment) {
	if e.CamelCatalog == nil {
		return
	}

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
		if ptr.Deref(t.SamplerParentBased, true) {
			e.ApplicationProperties[appPropSamplerParentBased] = "true"
		} else {
			e.ApplicationProperties[appPropSamplerParentBased] = boolean.FalseString
		}
	}
}

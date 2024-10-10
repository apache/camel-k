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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/trait/discovery"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

type telemetryTrait struct {
	BaseTrait
	traitv1.TelemetryTrait `property:",squash"`
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
func NewTelemetryTrait() Trait {
	return &telemetryTrait{
		BaseTrait: NewBaseTrait("telemetry", TraitOrderBeforeControllerCreation),
	}
}

func (t *telemetryTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil || !ptr.Deref(t.Enabled, false) {
		return false, nil, nil
	}

	var condition *TraitCondition

	// Deprecated
	if _, isAddon := e.Integration.Spec.Traits.Addons["telemetry"]; isAddon {
		condition = NewIntegrationCondition(
			"Telemetry",
			v1.IntegrationConditionTraitInfo,
			corev1.ConditionTrue,
			traitConfigurationReason,
			"Telemetry addon configuration is deprecated and may be removed in future releases. Make sure to use Telemetry trait configuration instead.",
		)
	}

	if !ptr.Deref(t.Auto, true) {
		return true, condition, nil
	}

	if t.Endpoint == "" {
		for _, locator := range discovery.TelemetryLocators {
			endpoint, err := locator.FindEndpoint(e.Ctx, t.Client, t.L, e.Integration.Namespace)
			if err != nil {
				return false, nil, err
			}
			if endpoint != "" {
				t.L.Infof("Using tracing endpoint: %s", endpoint)
				conditionMessage := "TracingEndpoint"
				if condition != nil {
					conditionMessage = conditionMessage + ";" + condition.message
				}
				condition = NewIntegrationCondition(
					"Telemetry",
					v1.IntegrationConditionTraitInfo,
					corev1.ConditionTrue,
					conditionMessage,
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

func (t *telemetryTrait) Apply(e *Environment) error {
	util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityTelemetry)

	// Hack for camel-k-runtime 3.15.0
	if e.CamelCatalog.CamelCatalogSpec.Runtime.Provider == v1.RuntimeProviderQuarkus &&
		e.CamelCatalog.CamelCatalogSpec.Runtime.Version == "3.15.0" {
		t.setRuntimeProviderQuarkus315Properties(e)
	} else if e.CamelCatalog.Runtime.Capabilities["telemetry"].RuntimeProperties != nil {
		t.setCatalogConfiguration(e)
	} else {
		t.setRuntimeProviderProperties(e)
	}

	return nil
}

func (t *telemetryTrait) setCatalogConfiguration(e *Environment) {
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

	if e.CamelCatalog.Runtime.Capabilities["telemetry"].RuntimeProperties != nil {
		for _, cp := range e.CamelCatalog.Runtime.Capabilities["telemetry"].RuntimeProperties {
			e.ApplicationProperties[CapabilityPropertyKey(cp.Key, e.ApplicationProperties)] = cp.Value
		}
	}
}

// Deprecated: to be removed in future release in favor of func setCatalogConfiguration().
func (t *telemetryTrait) setRuntimeProviderProperties(e *Environment) {
	provider := e.CamelCatalog.CamelCatalogSpec.Runtime.Provider
	properties := telemetryProperties[provider]
	if appPropEnabled := properties[propEnabled]; appPropEnabled != "" {
		e.ApplicationProperties[appPropEnabled] = boolean.TrueString
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
			e.ApplicationProperties[appPropSamplerParentBased] = boolean.TrueString
		} else {
			e.ApplicationProperties[appPropSamplerParentBased] = boolean.FalseString
		}
	}
}

// Hack for camel-k-runtime 3.15.0
// Otel quarkus properties name breaking change not present in camel-k-runtime.
func (t *telemetryTrait) setRuntimeProviderQuarkus315Properties(e *Environment) {
	propEndpoint := "quarkus.otel.exporter.otlp.traces.endpoint"
	propServiceName := "quarkus.otel.resource.attributes"
	propSampler := "quarkus.otel.traces.sampler"
	propSamplerRatio := "quarkus.otel.traces.sampler.ratio"
	propSamplerParentBased := "quarkus.otel.traces.sampler.parent-based"

	if t.Endpoint != "" {
		e.ApplicationProperties["camel.k.telemetry.endpoint"] = t.Endpoint
		e.ApplicationProperties[propEndpoint] = t.Endpoint
	}
	if t.ServiceName != "" {
		e.ApplicationProperties["camel.k.telemetry.serviceName"] = fmt.Sprintf("service.name=%s", t.ServiceName)
		e.ApplicationProperties[propServiceName] = fmt.Sprintf("service.name=%s", t.ServiceName)
	}
	if t.Sampler != "" {
		e.ApplicationProperties["camel.k.telemetry.sampler"] = t.Sampler
		e.ApplicationProperties[propSampler] = t.Sampler
	}
	if t.SamplerRatio != "" {
		e.ApplicationProperties["camel.k.telemetry.samplerRatio"] = t.SamplerRatio
		e.ApplicationProperties[propSamplerRatio] = t.SamplerRatio
	}
	if ptr.Deref(t.SamplerParentBased, true) {
		e.ApplicationProperties["camel.k.telemetry.samplerParentBased"] = boolean.TrueString
		e.ApplicationProperties[propSamplerParentBased] = boolean.TrueString
	} else {
		e.ApplicationProperties["camel.k.telemetry.samplerParentBased"] = boolean.FalseString
		e.ApplicationProperties[propSamplerParentBased] = boolean.FalseString
	}

	if e.CamelCatalog.Runtime.Capabilities["telemetry"].RuntimeProperties != nil {
		for _, cp := range e.CamelCatalog.Runtime.Capabilities["telemetry"].RuntimeProperties {
			e.ApplicationProperties[CapabilityPropertyKey(cp.Key, e.ApplicationProperties)] = cp.Value
		}
	}
}

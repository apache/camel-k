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

package tracing

import (
	"github.com/apache/camel-k/v2/addons/tracing/discovery"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"k8s.io/utils/ptr"
)

// WARNING: The Tracing trait has been **deprecated** in favor of the xref:traits:telemetry.adoc[Telemetry] trait.
//
// The Tracing trait can be used to automatically publish tracing information to an OpenTracing compatible collector.
//
// The trait is able to automatically discover the tracing endpoint available in the namespace (supports **Jaeger**).
//
// The Tracing trait is disabled by default.
//
// WARNING: The Tracing trait can't be enabled at the same time as the Telemetry trait.
//
// +camel-k:trait=tracing.
// +camel-k:deprecated=1.12.0.
type Trait struct {
	traitv1.Trait `property:",squash" json:",inline"`
	// Enables automatic configuration of the trait, including automatic discovery of the tracing endpoint.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// The name of the service that publishes tracing data (defaults to the integration name)
	ServiceName string `property:"service-name" json:"serviceName,omitempty"`
	// The target endpoint of the OpenTracing service (automatically discovered by default)
	Endpoint string `property:"endpoint" json:"endpoint,omitempty"`
	// The sampler type (default "const")
	SamplerType *string `property:"sampler-type" json:"samplerType,omitempty"`
	// The sampler specific param (default "1")
	SamplerParam *string `property:"sampler-param" json:"samplerParam,omitempty"`
}

type tracingTrait struct {
	trait.BaseTrait
	Trait `property:",squash"`
}

const (
	propEnabled      = "propEnabled"
	propEndpoint     = "propEndpoint"
	propServiceName  = "propServiceName"
	propSamplerType  = "propSamplerType"
	propSamplerParam = "propSamplerParam"
)

var (
	tracingProperties = map[v1.RuntimeProvider]map[string]string{
		v1.RuntimeProviderQuarkus: {
			propEndpoint:     "quarkus.jaeger.endpoint",
			propServiceName:  "quarkus.jaeger.service-name",
			propSamplerType:  "quarkus.jaeger.sampler-type",
			propSamplerParam: "quarkus.jaeger.sampler-param",
		},
	}

	defaultSamplerType  = "const"
	defaultSamplerParam = "1"
)

// NewTracingTrait --.
func NewTracingTrait() trait.Trait {
	return &tracingTrait{
		BaseTrait: trait.NewBaseTrait("tracing", trait.TraitOrderBeforeControllerCreation),
	}
}

func (t *tracingTrait) Configure(e *trait.Environment) (bool, *trait.TraitCondition, error) {
	if e.Integration == nil || !ptr.Deref(t.Enabled, false) {
		return false, nil, nil
	}
	if e.CamelCatalog == nil {
		return false, trait.NewIntegrationConditionPlatformDisabledCatalogMissing(), nil
	}

	if !ptr.Deref(t.Auto, true) {
		return true, nil, nil
	}

	if t.Endpoint == "" {
		for _, locator := range discovery.TracingLocators {
			endpoint, err := locator.FindEndpoint(e.Ctx, t.Client, t.L, e)
			if err != nil {
				return false, nil, err
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

	if t.SamplerType == nil {
		t.SamplerType = &defaultSamplerType
	}

	if t.SamplerParam == nil {
		t.SamplerParam = &defaultSamplerParam
	}

	return true, nil, nil
}

func (t *tracingTrait) Apply(e *trait.Environment) error {
	util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityTracing)

	provider := e.CamelCatalog.CamelCatalogSpec.Runtime.Provider
	properties := tracingProperties[provider]

	if appPropEnabled := properties[propEnabled]; appPropEnabled != "" {
		e.ApplicationProperties[appPropEnabled] = "true"
	}
	if appPropEndpoint := properties[propEndpoint]; appPropEndpoint != "" && t.Endpoint != "" {
		e.ApplicationProperties[appPropEndpoint] = t.Endpoint
	}
	if appPropServiceName := properties[propServiceName]; appPropServiceName != "" && t.ServiceName != "" {
		e.ApplicationProperties[appPropServiceName] = t.ServiceName
	}
	if appPropSamplerType := properties[propSamplerType]; appPropSamplerType != "" && t.SamplerType != nil {
		e.ApplicationProperties[appPropSamplerType] = *t.SamplerType
	}
	if appPropSamplerParam := properties[propSamplerParam]; appPropSamplerParam != "" && t.SamplerParam != nil {
		e.ApplicationProperties[appPropSamplerParam] = *t.SamplerParam
	}

	return nil
}

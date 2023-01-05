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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/camel"

	"github.com/stretchr/testify/assert"
)

func TestTelemetryTraitOnDefaultQuarkus(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog)
	telemetry := NewTelemetryTrait()
	tt, _ := telemetry.(*telemetryTrait)
	tt.Enabled = pointer.Bool(true)
	tt.Endpoint = "http://endpoint3"
	ok, err := telemetry.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = telemetry.Apply(e)
	assert.Nil(t, err)

	assert.Empty(t, e.ApplicationProperties["quarkus.opentelemetry.enabled"])
	assert.Equal(t, "http://endpoint3", e.ApplicationProperties["quarkus.opentelemetry.tracer.exporter.otlp.endpoint"])
	assert.Equal(t, "service.name=test", e.ApplicationProperties["quarkus.opentelemetry.tracer.resource-attributes"])
	assert.Equal(t, "on", e.ApplicationProperties["quarkus.opentelemetry.tracer.sampler"])
	assert.Empty(t, e.ApplicationProperties["quarkus.opentelemetry.tracer.sampler.ratio"])
	assert.Equal(t, "true", e.ApplicationProperties["quarkus.opentelemetry.tracer.sampler.parent-based"])
}

func TestTelemetryTraitWithValues(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog)
	telemetry := NewTelemetryTrait()
	tt, _ := telemetry.(*telemetryTrait)
	tt.Enabled = pointer.Bool(true)
	tt.Endpoint = "http://endpoint3"
	tt.ServiceName = "Test"
	tt.Sampler = "ratio"
	tt.SamplerRatio = "0.001"
	tt.SamplerParentBased = pointer.Bool(false)
	ok, err := telemetry.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = telemetry.Apply(e)
	assert.Nil(t, err)

	assert.Empty(t, e.ApplicationProperties["quarkus.opentelemetry.enabled"])
	assert.Equal(t, "http://endpoint3", e.ApplicationProperties["quarkus.opentelemetry.tracer.exporter.otlp.endpoint"])
	assert.Equal(t, "service.name=Test", e.ApplicationProperties["quarkus.opentelemetry.tracer.resource-attributes"])
	assert.Equal(t, "ratio", e.ApplicationProperties["quarkus.opentelemetry.tracer.sampler"])
	assert.Equal(t, "0.001", e.ApplicationProperties["quarkus.opentelemetry.tracer.sampler.ratio"])
	assert.Equal(t, "false", e.ApplicationProperties["quarkus.opentelemetry.tracer.sampler.parent-based"])
}

func createEnvironment(t *testing.T, catalogGen func() (*camel.RuntimeCatalog, error)) *trait.Environment {
	t.Helper()

	catalog, err := catalogGen()
	assert.Nil(t, err)

	e := trait.Environment{
		CamelCatalog:          catalog,
		ApplicationProperties: make(map[string]string),
	}

	it := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseDeploying,
		},
	}
	e.Integration = &it
	return &e
}

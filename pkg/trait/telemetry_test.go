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
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestTelemetryTraitOnDefaultQuarkus(t *testing.T) {
	e := createTelemetryEnvironment(t, camel.QuarkusCatalog)
	telemetry := NewTelemetryTrait()
	tt, _ := telemetry.(*telemetryTrait)
	tt.Enabled = ptr.To(true)
	tt.Endpoint = "http://endpoint3"
	ok, condition, err := telemetry.Configure(e)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	err = telemetry.Apply(e)
	require.NoError(t, err)

	assert.Empty(t, e.ApplicationProperties["quarkus.opentelemetry.enabled"])
	assert.Equal(t, "http://endpoint3", e.ApplicationProperties["camel.k.telemetry.endpoint"])
	assert.Equal(t, "service.name=test", e.ApplicationProperties["camel.k.telemetry.serviceName"])
	assert.Equal(t, "on", e.ApplicationProperties["camel.k.telemetry.sampler"])
	assert.Equal(t, "", e.ApplicationProperties["camel.k.telemetry.samplerRatio"])
	assert.Equal(t, "true", e.ApplicationProperties["camel.k.telemetry.samplerParentBased"])
	assert.Equal(t, "${camel.k.telemetry.endpoint}", e.ApplicationProperties["quarkus.opentelemetry.tracer.exporter.otlp.endpoint"])
	assert.Equal(t, "${camel.k.telemetry.serviceName}", e.ApplicationProperties["quarkus.opentelemetry.tracer.resource-attributes"])
	assert.Equal(t, "${camel.k.telemetry.sampler}", e.ApplicationProperties["quarkus.opentelemetry.tracer.sampler"])
	assert.Equal(t, "${camel.k.telemetry.samplerRatio}", e.ApplicationProperties["quarkus.opentelemetry.tracer.sampler.ratio"])
	assert.Equal(t, "${camel.k.telemetry.samplerParentBased}", e.ApplicationProperties["quarkus.opentelemetry.tracer.sampler.parent-based"])
}

func TestTelemetryTraitWithValues(t *testing.T) {
	e := createTelemetryEnvironment(t, camel.QuarkusCatalog)
	telemetry := NewTelemetryTrait()
	tt, _ := telemetry.(*telemetryTrait)
	tt.Enabled = ptr.To(true)
	tt.Endpoint = "http://endpoint3"
	tt.ServiceName = "Test"
	tt.Sampler = "ratio"
	tt.SamplerRatio = "0.001"
	tt.SamplerParentBased = ptr.To(false)
	ok, condition, err := telemetry.Configure(e)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	err = telemetry.Apply(e)
	require.NoError(t, err)

	assert.Empty(t, e.ApplicationProperties["quarkus.opentelemetry.enabled"])
	assert.Equal(t, "http://endpoint3", e.ApplicationProperties["camel.k.telemetry.endpoint"])
	assert.Equal(t, "service.name=Test", e.ApplicationProperties["camel.k.telemetry.serviceName"])
	assert.Equal(t, "ratio", e.ApplicationProperties["camel.k.telemetry.sampler"])
	assert.Equal(t, "0.001", e.ApplicationProperties["camel.k.telemetry.samplerRatio"])
	assert.Equal(t, boolean.FalseString, e.ApplicationProperties["camel.k.telemetry.samplerParentBased"])
	assert.Equal(t, "${camel.k.telemetry.endpoint}", e.ApplicationProperties["quarkus.opentelemetry.tracer.exporter.otlp.endpoint"])
	assert.Equal(t, "${camel.k.telemetry.serviceName}", e.ApplicationProperties["quarkus.opentelemetry.tracer.resource-attributes"])
	assert.Equal(t, "${camel.k.telemetry.sampler}", e.ApplicationProperties["quarkus.opentelemetry.tracer.sampler"])
	assert.Equal(t, "${camel.k.telemetry.samplerRatio}", e.ApplicationProperties["quarkus.opentelemetry.tracer.sampler.ratio"])
	assert.Equal(t, "${camel.k.telemetry.samplerParentBased}", e.ApplicationProperties["quarkus.opentelemetry.tracer.sampler.parent-based"])
}

func TestTelemetryForSelfManagedBuild(t *testing.T) {
	e := createTelemetryEnvironment(t, camel.QuarkusCatalog)
	telemetry := NewTelemetryTrait()
	tt, _ := telemetry.(*telemetryTrait)
	tt.Enabled = ptr.To(true)
	tt.Auto = ptr.To(false)
	tt.Endpoint = "http://endpoint3"
	tt.ServiceName = "Test"
	tt.Sampler = "ratio"
	tt.SamplerRatio = "0.001"
	tt.SamplerParentBased = ptr.To(false)

	ok, condition, err := telemetry.Configure(e)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	err = telemetry.Apply(e)
	require.NoError(t, err)

	assert.Empty(t, e.ApplicationProperties["quarkus.opentelemetry.enabled"])
	assert.Equal(t, "http://endpoint3", e.ApplicationProperties["camel.k.telemetry.endpoint"])
	assert.Equal(t, "service.name=Test", e.ApplicationProperties["camel.k.telemetry.serviceName"])
	assert.Equal(t, "ratio", e.ApplicationProperties["camel.k.telemetry.sampler"])
	assert.Equal(t, "0.001", e.ApplicationProperties["camel.k.telemetry.samplerRatio"])
	assert.Equal(t, boolean.FalseString, e.ApplicationProperties["camel.k.telemetry.samplerParentBased"])
}

func createTelemetryEnvironment(t *testing.T, catalogGen func() (*camel.RuntimeCatalog, error)) *Environment {
	t.Helper()

	catalog, err := catalogGen()
	require.NoError(t, err)

	e := Environment{
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

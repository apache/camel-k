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
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTracingTrait(t *testing.T) {
	e := createEnvironment(t, camel.DefaultCatalog)
	tracing := NewTracingTrait()
	enabled := true
	tracing.(*tracingTrait).Enabled = &enabled
	tracing.(*tracingTrait).Endpoint = "http://endpoint1"
	ok, err := tracing.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = tracing.Apply(e)
	assert.Nil(t, err)

	assert.Equal(t, "true", e.ApplicationProperties["camel.k.customizer.tracing.enabled"])
	assert.Equal(t, "http://endpoint1", e.ApplicationProperties["camel.k.customizer.tracing.reporter.sender.endpoint"])
	assert.Equal(t, "test", e.ApplicationProperties["camel.k.customizer.tracing.service-name"])
	assert.Equal(t, "const", e.ApplicationProperties["camel.k.customizer.tracing.sampler.type"])
	assert.Equal(t, "1", e.ApplicationProperties["camel.k.customizer.tracing.sampler.param"])
}

func TestTracingTraitFullConfig(t *testing.T) {
	e := createEnvironment(t, camel.DefaultCatalog)
	tracing := NewTracingTrait()
	enabled := true
	tracing.(*tracingTrait).Enabled = &enabled
	tracing.(*tracingTrait).Endpoint = "http://endpoint2"
	samplerParam := "2"
	tracing.(*tracingTrait).SamplerParam = &samplerParam
	samplerType := "buh"
	tracing.(*tracingTrait).SamplerType = &samplerType
	tracing.(*tracingTrait).ServiceName = "myservice"
	ok, err := tracing.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = tracing.Apply(e)
	assert.Nil(t, err)

	assert.Equal(t, "true", e.ApplicationProperties["camel.k.customizer.tracing.enabled"])
	assert.Equal(t, "http://endpoint2", e.ApplicationProperties["camel.k.customizer.tracing.reporter.sender.endpoint"])
	assert.Equal(t, "myservice", e.ApplicationProperties["camel.k.customizer.tracing.service-name"])
	assert.Equal(t, "buh", e.ApplicationProperties["camel.k.customizer.tracing.sampler.type"])
	assert.Equal(t, "2", e.ApplicationProperties["camel.k.customizer.tracing.sampler.param"])
}

func TestTracingTraitOnQuarkus(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog)
	tracing := NewTracingTrait()
	enabled := true
	tracing.(*tracingTrait).Enabled = &enabled
	tracing.(*tracingTrait).Endpoint = "http://endpoint3"
	ok, err := tracing.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = tracing.Apply(e)
	assert.Nil(t, err)

	assert.Empty(t, e.ApplicationProperties["quarkus.jaeger.enabled"])
	assert.Equal(t, "http://endpoint3", e.ApplicationProperties["quarkus.jaeger.endpoint"])
	assert.Equal(t, "test", e.ApplicationProperties["quarkus.jaeger.service-name"])
	assert.Equal(t, "const", e.ApplicationProperties["quarkus.jaeger.sampler-type"])
	assert.Equal(t, "1", e.ApplicationProperties["quarkus.jaeger.sampler-param"])
}

func createEnvironment(t *testing.T, catalogGen func() (*camel.RuntimeCatalog, error)) *trait.Environment {
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

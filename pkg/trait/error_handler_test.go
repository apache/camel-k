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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

func TestErrorHandlerConfigureFromIntegrationProperty(t *testing.T) {
	e := &Environment{
		Catalog:     NewEnvironmentTestCatalog(),
		Integration: &v1.Integration{},
	}
	e.Integration.Spec.AddConfigurationProperty(fmt.Sprintf("%v = %s", v1.ErrorHandlerRefName, "defaultErrorHandler"))

	trait := newErrorHandlerTrait()
	enabled, condition, err := trait.Configure(e)
	require.NoError(t, err)
	assert.False(t, enabled)
	assert.Nil(t, condition)

	e.Integration.Status.Phase = v1.IntegrationPhaseNone
	enabled, condition, err = trait.Configure(e)
	require.NoError(t, err)
	assert.False(t, enabled)
	assert.Nil(t, condition)

	e.Integration.Status.Phase = v1.IntegrationPhaseInitialization
	enabled, condition, err = trait.Configure(e)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)

}

func TestErrorHandlerApplySource(t *testing.T) {
	e := &Environment{
		Catalog:     NewEnvironmentTestCatalog(),
		Integration: &v1.Integration{},
	}
	e.Integration.Spec.AddConfigurationProperty(fmt.Sprintf("%v = %s", v1.ErrorHandlerRefName, "defaultErrorHandler"))
	e.Integration.Status.Phase = v1.IntegrationPhaseInitialization

	trait := newErrorHandlerTrait()
	enabled, condition, err := trait.Configure(e)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)

	err = trait.Apply(e)
	require.NoError(t, err)
	assert.Equal(t, `- error-handler:
    ref-error-handler: defaultErrorHandler
`, e.Integration.Status.GeneratedSources[0].Content)
}

func TestErrorHandlerApplyDependency(t *testing.T) {
	c, err := camel.DefaultCatalog()
	require.NoError(t, err)
	e := &Environment{
		Catalog:      NewEnvironmentTestCatalog(),
		CamelCatalog: c,
		Integration:  &v1.Integration{},
	}
	e.Integration.Spec.AddConfigurationProperty("camel.beans.defaultErrorHandler = #class:org.apache.camel.builder.DeadLetterChannelBuilder")
	e.Integration.Spec.AddConfigurationProperty("camel.beans.defaultErrorHandler.deadLetterUri = log:info")
	e.Integration.Spec.AddConfigurationProperty(fmt.Sprintf("%v = %s", v1.ErrorHandlerRefName, "defaultErrorHandler"))
	e.Integration.Status.Phase = v1.IntegrationPhaseInitialization

	trait := newErrorHandlerTrait()
	enabled, condition, err := trait.Configure(e)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)

	err = trait.Apply(e)
	require.NoError(t, err)
	assert.Len(t, e.Integration.Spec.Configuration, 3)
	assert.Equal(t, "camel:log", e.Integration.Status.Dependencies[0])
}

func TestErrorHandlerDisableNoErrorHandler(t *testing.T) {
	c, err := camel.DefaultCatalog()
	require.NoError(t, err)
	e := &Environment{
		Catalog:      NewEnvironmentTestCatalog(),
		CamelCatalog: c,
		Integration:  &v1.Integration{},
	}
	e.Integration.Spec.AddConfigurationProperty("camel.beans.defaultErrorHandler = #class:org.apache.camel.builder.DeadLetterChannelBuilder")
	e.Integration.Spec.AddConfigurationProperty("camel.beans.defaultErrorHandler.deadLetterUri = log:info")
	e.Integration.Spec.AddConfigurationProperty(fmt.Sprintf("%v = %s", v1.ErrorHandlerRefName, "defaultErrorHandler"))
	e.Integration.Status.Phase = v1.IntegrationPhaseInitialization
	e.Integration.Status.RuntimeVersion = "3.8.0-SNAPSHOT"

	trait := newErrorHandlerTrait()
	enabled, condition, err := trait.Configure(e)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)

	err = trait.Apply(e)
	require.NoError(t, err)
	assert.Len(t, e.Integration.Spec.Configuration, 4)
	assert.Equal(t, "#class:org.apache.camel.builder.DeadLetterChannelBuilder", e.Integration.Spec.GetConfigurationProperty("camel.beans.defaultErrorHandler"))
	assert.Equal(t, "log:info", e.Integration.Spec.GetConfigurationProperty("camel.beans.defaultErrorHandler.deadLetterUri"))
	assert.Equal(t, "defaultErrorHandler", e.Integration.Spec.GetConfigurationProperty(v1.ErrorHandlerRefName))
	assert.Equal(t, "false", e.Integration.Spec.GetConfigurationProperty("camel.component.kamelet.noErrorHandler"))
}

func TestShouldDisableNoErrorHandler(t *testing.T) {
	assert.False(t, shouldHandleNoErrorHandler(&v1.Integration{}))
	assert.False(t, shouldHandleNoErrorHandler(&v1.Integration{
		Status: v1.IntegrationStatus{
			RuntimeVersion: defaults.DefaultRuntimeVersion,
		},
	}))
	assert.False(t, shouldHandleNoErrorHandler(&v1.Integration{
		Status: v1.IntegrationStatus{
			RuntimeVersion: "3.6.0",
		},
	}))
	assert.False(t, shouldHandleNoErrorHandler(&v1.Integration{
		Status: v1.IntegrationStatus{
			RuntimeVersion: "3.6.0-SNAPSHOT",
		},
	}))
	assert.False(t, shouldHandleNoErrorHandler(&v1.Integration{
		Status: v1.IntegrationStatus{
			RuntimeVersion: "3.6.0-ABCDEFG",
		},
	}))
	assert.True(t, shouldHandleNoErrorHandler(&v1.Integration{
		Status: v1.IntegrationStatus{
			RuntimeVersion: "3.8.0",
		},
	}))
	assert.True(t, shouldHandleNoErrorHandler(&v1.Integration{
		Status: v1.IntegrationStatus{
			RuntimeVersion: "3.8.0-SNAPSHOT",
		},
	}))
	assert.True(t, shouldHandleNoErrorHandler(&v1.Integration{
		Status: v1.IntegrationStatus{
			RuntimeVersion: "3.8.0-ABCDEFG",
		},
	}))
}

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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	"github.com/apache/camel-k/v2/pkg/util/camel"
)

func TestErrorHandlerConfigureFromIntegrationProperty(t *testing.T) {
	e := &Environment{
		Catalog:     NewEnvironmentTestCatalog(),
		Integration: &v1.Integration{},
	}
	e.Integration.Spec.AddConfiguration("property", fmt.Sprintf("%v = %s", v1.ErrorHandlerRefName, "defaultErrorHandler"))

	trait := newErrorHandlerTrait()
	enabled, condition, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)
	assert.Nil(t, condition)

	e.Integration.Status.Phase = v1.IntegrationPhaseNone
	enabled, condition, err = trait.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)
	assert.Nil(t, condition)

	e.Integration.Status.Phase = v1.IntegrationPhaseInitialization
	enabled, condition, err = trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)

}

func TestErrorHandlerApplySource(t *testing.T) {
	e := &Environment{
		Catalog:     NewEnvironmentTestCatalog(),
		Integration: &v1.Integration{},
	}
	e.Integration.Spec.AddConfiguration("property", fmt.Sprintf("%v = %s", v1.ErrorHandlerRefName, "defaultErrorHandler"))
	e.Integration.Status.Phase = v1.IntegrationPhaseInitialization

	trait := newErrorHandlerTrait()
	enabled, condition, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)

	err = trait.Apply(e)
	assert.Nil(t, err)
	assert.Equal(t, `- error-handler:
    ref-error-handler: defaultErrorHandler
`, e.Integration.Status.GeneratedSources[0].Content)
}

func TestErrorHandlerApplyDependency(t *testing.T) {
	c, err := camel.DefaultCatalog()
	assert.Nil(t, err)
	e := &Environment{
		Catalog:      NewEnvironmentTestCatalog(),
		CamelCatalog: c,
		Integration:  &v1.Integration{},
	}
	e.Integration.Spec.AddConfiguration("property", "camel.beans.defaultErrorHandler = #class:org.apache.camel.builder.DeadLetterChannelBuilder")
	e.Integration.Spec.AddConfiguration("property", "camel.beans.defaultErrorHandler.deadLetterUri = log:info")
	e.Integration.Spec.AddConfiguration("property", fmt.Sprintf("%v = %s", v1.ErrorHandlerRefName, "defaultErrorHandler"))
	e.Integration.Status.Phase = v1.IntegrationPhaseInitialization

	trait := newErrorHandlerTrait()
	enabled, condition, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)

	err = trait.Apply(e)
	assert.Nil(t, err)
	assert.Equal(t, "camel:log", e.Integration.Status.Dependencies[0])
}

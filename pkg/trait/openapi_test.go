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

	"github.com/apache/camel-k/v2/pkg/util/camel"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	"github.com/stretchr/testify/assert"
)

func TestRestDslTraitApplicability(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	e := &Environment{
		CamelCatalog: catalog,
	}

	trait, _ := newOpenAPITrait().(*openAPITrait)
	enabled, condition, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)
	assert.Nil(t, condition)

	e.Integration = &v1.Integration{
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseNone,
		},
	}
	enabled, condition, err = trait.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)
	assert.Nil(t, condition)

	trait.Configmaps = []string{"my-configmap"}

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

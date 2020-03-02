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

	"github.com/apache/camel-k/pkg/util/camel"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"

	"github.com/stretchr/testify/assert"
)

func TestRestDslTraitApplicability(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	e := &Environment{
		CamelCatalog: catalog,
	}

	trait := newRestDslTrait()
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)

	e.Integration = &v1.Integration{
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseNone,
		},
	}
	enabled, err = trait.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)

	resource := v1.ResourceSpec{
		Type: v1.ResourceTypeOpenAPI,
	}
	e.Integration.Spec.Resources = append(e.Integration.Spec.Resources, resource)
	enabled, err = trait.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)

	e.Integration.Status.Phase = v1.IntegrationPhaseInitialization
	enabled, err = trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)
}

func TestRestDslTraitDeps(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	e := &Environment{
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Resources: []v1.ResourceSpec{
					{Type: v1.ResourceTypeOpenAPI},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
	}

	trait := newRestDslTrait().(*restDslTrait)
	trait.computeDependencies(e)

	assert.Contains(t, e.Integration.Status.Dependencies, "mvn:org.apache.camel/camel-rest")
	assert.Contains(t, e.Integration.Status.Dependencies, "mvn:org.apache.camel/camel-undertow")
}

func TestRestDslTraitDepsQuarkus(t *testing.T) {
	catalog, err := camel.QuarkusCatalog()
	assert.Nil(t, err)

	e := &Environment{
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Resources: []v1.ResourceSpec{
					{Type: v1.ResourceTypeOpenAPI},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
	}

	trait := newRestDslTrait().(*restDslTrait)
	trait.computeDependencies(e)

	assert.Contains(t, e.Integration.Status.Dependencies, "mvn:org.apache.camel.quarkus/camel-quarkus-rest")
	assert.Contains(t, e.Integration.Status.Dependencies, "mvn:org.apache.camel.quarkus/camel-quarkus-platform-http")
}

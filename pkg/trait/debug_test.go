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

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/envvar"
)

func TestDebugTraitApplicability(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	env := Environment{
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Traits: map[string]v1.TraitSpec{
					"debug": {
						Configuration: map[string]string{
							"enabled": "true",
						},
					},
				},
			},
		},
		EnvVars: make([]corev1.EnvVar, 0),
	}

	trait := newDebugTrait()

	enabled, err := trait.Configure(&env)
	assert.Nil(t, err)
	assert.False(t, enabled)

	env.Integration.Status.Phase = v1.IntegrationPhaseRunning

	enabled, err = trait.Configure(&env)
	assert.Nil(t, err)
	assert.False(t, enabled)
}

func TestApplyDebugTrait(t *testing.T) {
	environment := Environment{
		Integration: &v1.Integration{
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Traits: map[string]v1.TraitSpec{
					"debug": {
						Configuration: map[string]string{
							"enabled": "true",
						},
					},
				},
			},
		},
		EnvVars: make([]corev1.EnvVar, 0),
	}

	trait := newDebugTrait()

	assert.Nil(t, trait.Apply(&environment))
	assert.NotNil(t, envvar.Get(environment.EnvVars, "JAVA_DEBUG"))
	assert.Equal(t, True, envvar.Get(environment.EnvVars, "JAVA_DEBUG").Value)
}

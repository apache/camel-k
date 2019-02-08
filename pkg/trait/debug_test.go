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

	"github.com/apache/camel-k/pkg/util/test"

	"github.com/apache/camel-k/pkg/util/envvar"

	"k8s.io/api/core/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestDebugTraitApplicability(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	env := Environment{
		CamelCatalog: catalog,
		Integration: &v1alpha1.Integration{
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
			Spec: v1alpha1.IntegrationSpec{
				Traits: map[string]v1alpha1.IntegrationTraitSpec{
					"debug": {
						Configuration: map[string]string{
							"enabled": "true",
						},
					},
				},
			},
		},
		EnvVars: make([]v1.EnvVar, 0),
	}

	trait := newDebugTrait()

	enabled, err := trait.Configure(&env)
	assert.Nil(t, err)
	assert.False(t, enabled)

	env.Integration.Status.Phase = v1alpha1.IntegrationPhaseRunning

	enabled, err = trait.Configure(&env)
	assert.Nil(t, err)
	assert.False(t, enabled)
}

func TestApplyDebugTrait(t *testing.T) {
	environment := Environment{
		Integration: &v1alpha1.Integration{
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
			Spec: v1alpha1.IntegrationSpec{
				Traits: map[string]v1alpha1.IntegrationTraitSpec{
					"debug": {
						Configuration: map[string]string{
							"enabled": "true",
						},
					},
				},
			},
		},
		EnvVars: make([]v1.EnvVar, 0),
	}

	trait := newDebugTrait()

	assert.Nil(t, trait.Apply(&environment))
	assert.NotNil(t, envvar.Get(environment.EnvVars, "JAVA_DEBUG"))
	assert.Equal(t, True, envvar.Get(environment.EnvVars, "JAVA_DEBUG").Value)
}

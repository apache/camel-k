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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/stretchr/testify/assert"
)

var (
	env = &Environment{
		Integration: &v1alpha1.Integration{
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
		},
		EnvVars: make(map[string]string)}

	trait = newDebugTrait()
)

func TestApplicability(t *testing.T) {
	assert.True(t, trait.appliesTo(env))

	env.Integration.Status.Phase = v1alpha1.IntegrationPhaseRunning
	assert.False(t, trait.appliesTo(env))
}

func TestApply(t *testing.T) {
	assert.Nil(t, trait.apply(env))
	assert.Equal(t, True, env.EnvVars["JAVA_DEBUG"])
}

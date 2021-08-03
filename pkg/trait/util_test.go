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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestCollectConfigurationValues(t *testing.T) {
	e := Environment{
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Configuration: []v1.ConfigurationSpec{
					{Type: "configmap", Value: "my-cm-integration"},
					{Type: "env", Value: "my-env-integration"},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Spec: v1.IntegrationKitSpec{
				Configuration: []v1.ConfigurationSpec{
					{Type: "configmap", Value: "my-cm-kit"},
					{Type: "property", Value: "my-p-kit"},
				},
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Configuration: []v1.ConfigurationSpec{
					{Type: "configmap", Value: "my-cm-platform"},
					{Type: "secret", Value: "my-secret-platform"},
					{Type: "property", Value: "my-p-platform"},
					{Type: "env", Value: "my-env-platform"},
				},
			},
		},
	}
	e.Platform.ResyncStatusFullConfig()

	assert.Contains(t, e.collectConfigurationValues("configmap"), "my-cm-integration")
	assert.Contains(t, e.collectConfigurationValues("secret"), "my-secret-platform")
	assert.Contains(t, e.collectConfigurationValues("property"), "my-p-kit")
	assert.Contains(t, e.collectConfigurationValues("env"), "my-env-integration")
}

func TestCollectConfigurationPairs(t *testing.T) {
	e := Environment{
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Configuration: []v1.ConfigurationSpec{
					{Type: "property", Value: "p1=integration"},
					{Type: "property", Value: "p4=integration"},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Spec: v1.IntegrationKitSpec{
				Configuration: []v1.ConfigurationSpec{
					{Type: "property", Value: "p1=kit"},
					{Type: "property", Value: "p2=kit"},
				},
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Configuration: []v1.ConfigurationSpec{
					{Type: "property", Value: "p1=platform"},
					{Type: "property", Value: "p2=platform"},
					{Type: "property", Value: "p3=platform"},
					{Type: "property", Value: "p4=platform"},
				},
			},
		},
	}
	e.Platform.ResyncStatusFullConfig()

	pairs := e.collectConfigurationPairs("property")
	assert.Equal(t, pairs, []variable{
		{Name: "p1", Value: "integration"},
		{Name: "p2", Value: "kit"},
		{Name: "p3", Value: "platform"},
		{Name: "p4", Value: "integration"},
	})
}

func TestBoolPointerFunctions(t *testing.T) {
	trueP := BoolP(true)
	falseP := BoolP(false)

	assert.True(t, IsTrue(trueP))
	assert.False(t, IsTrue(falseP))
	assert.False(t, IsTrue(nil))

	assert.True(t, IsNilOrTrue(trueP))
	assert.False(t, IsNilOrTrue(falseP))
	assert.True(t, IsNilOrTrue(nil))

	assert.False(t, IsFalse(trueP))
	assert.True(t, IsFalse(falseP))
	assert.False(t, IsFalse(nil))

	assert.False(t, IsNilOrFalse(trueP))
	assert.True(t, IsNilOrFalse(falseP))
	assert.True(t, IsNilOrFalse(nil))
}

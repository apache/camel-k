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

package addons

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/v2/addons/master"
	"github.com/apache/camel-k/v2/addons/telemetry"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/trait"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraitConfiguration(t *testing.T) {
	env := trait.Environment{
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
				Traits: v1.Traits{
					Addons: map[string]v1.AddonTrait{
						"master": trait.ToAddonTrait(t, map[string]interface{}{
							"enabled":      true,
						}),
						"telemetry": trait.ToAddonTrait(t, map[string]interface{}{
							"enabled": true,
						}),
					},
				},
			},
		},
	}
	c := trait.NewCatalog(nil)
	require.NoError(t, c.Configure(&env))

	require.NotNil(t, c.GetTrait("master"))
	master, ok := c.GetTrait("master").(*master.TestMasterTrait)
	require.True(t, ok)
	assert.True(t, *master.Enabled)

	require.NotNil(t, c.GetTrait("telemetry"))
	telemetry, ok := c.GetTrait("telemetry").(*telemetry.TestTelemetryTrait)
	require.True(t, ok)
	assert.True(t, *telemetry.Enabled)
}

func TestTraitConfigurationFromAnnotations(t *testing.T) {
	env := trait.Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"trait.camel.apache.org/master.enabled":       "true",
					"trait.camel.apache.org/telemetry.enabled":    "true",
				},
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
			},
		},
	}
	c := trait.NewCatalog(nil)
	require.NoError(t, c.Configure(&env))

	require.NotNil(t, c.GetTrait("master"))
	master, ok := c.GetTrait("master").(*master.TestMasterTrait)
	require.True(t, ok)
	assert.True(t, *master.Enabled)

	require.NotNil(t, c.GetTrait("telemetry"))
	telemetry, ok := c.GetTrait("telemetry").(*telemetry.TestTelemetryTrait)
	require.True(t, ok)
	assert.True(t, *telemetry.Enabled)
}

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
	"encoding/json"
	"testing"

	"github.com/apache/camel-k/v2/addons/master"
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
						"master": toAddonTrait(t, map[string]interface{}{
							"enabled":      true,
							"resourceName": "test-lock",
							"labelKey":     "test-label",
							"labelValue":   "test-value",
						}),
						"telemetry": toAddonTrait(t, map[string]interface{}{
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
	assert.Equal(t, "test-lock", *master.ResourceName)
	assert.Equal(t, "test-label", *master.LabelKey)
	assert.Equal(t, "test-value", *master.LabelValue)

}

func toAddonTrait(t *testing.T, config map[string]interface{}) v1.AddonTrait {
	t.Helper()

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var addon v1.AddonTrait
	err = json.Unmarshal(data, &addon)
	require.NoError(t, err)

	return addon
}

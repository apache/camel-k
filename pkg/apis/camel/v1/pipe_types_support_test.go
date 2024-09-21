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

package v1

import (
	"encoding/json"
	"testing"

	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestNumberConversion(t *testing.T) {
	props := map[string]interface{}{
		"string":  "str",
		"int32":   1000000,
		"int64":   int64(10000000000),
		"float32": float32(123.123),
		"float64": float64(1111123.123),
	}
	ser, err := json.Marshal(props)
	require.NoError(t, err)
	ep := EndpointProperties{
		RawMessage: ser,
	}
	res, err := ep.GetPropertyMap()
	require.NoError(t, err)
	assert.Equal(t, "str", res["string"])
	assert.Equal(t, "1000000", res["int32"])
	assert.Equal(t, "10000000000", res["int64"])
	assert.Equal(t, "123.123", res["float32"])
	assert.Equal(t, "1111123.123", res["float64"])
}

func TestSetTraits(t *testing.T) {
	traits := Traits{
		Affinity: &trait.AffinityTrait{
			Trait: trait.Trait{
				Enabled: ptr.To(true),
			},
			PodAffinity: ptr.To(true),
		},
		Addons: map[string]AddonTrait{
			"master": toAddonTrait(t, map[string]interface{}{
				"enabled":      true,
				"resourceName": "test-lock",
				"labelKey":     "test-label",
				"labelValue":   "test-value",
			}),
		},
		Knative: &trait.KnativeTrait{
			Trait: trait.Trait{
				Enabled: ptr.To(true),
			},
			ChannelSources: []string{
				"channel-a", "channel-b",
			},
		},
	}

	expectedAnnotations := map[string]string(map[string]string{
		"trait.camel.apache.org/affinity.enabled":        "true",
		"trait.camel.apache.org/affinity.pod-affinity":   "true",
		"trait.camel.apache.org/knative.channel-sources": "[channel-a channel-b]",
		"trait.camel.apache.org/knative.enabled":         "true",
		"trait.camel.apache.org/master.enabled":          "true",
		"trait.camel.apache.org/master.label-key":        "test-label",
		"trait.camel.apache.org/master.label-value":      "test-value",
		"trait.camel.apache.org/master.resource-name":    "test-lock",
	})

	pipe := NewPipe("my-pipe", "my-ns")
	err := pipe.SetTraits(&traits)
	assert.NoError(t, err)
	assert.Equal(t, expectedAnnotations, pipe.Annotations)
}

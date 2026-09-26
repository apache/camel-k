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

package apis

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"k8s.io/apimachinery/pkg/api/equality"
)

func TestURLJSON(t *testing.T) {
	u, err := ParseURL("http://my-service.my-ns.svc.cluster.local/path?query=1")
	require.NoError(t, err)

	data, err := json.Marshal(u)
	require.NoError(t, err)
	assert.Equal(t, `"http://my-service.my-ns.svc.cluster.local/path?query=1"`, string(data))

	var parsed URL
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Equal(t, *u, parsed)
	assert.Equal(t, "my-service.my-ns.svc.cluster.local", parsed.Host)
	assert.Equal(t, "/path", parsed.Path)

	require.NoError(t, json.Unmarshal([]byte(`""`), &parsed))
	assert.Equal(t, URL{}, parsed)
	require.Error(t, json.Unmarshal([]byte(`"://invalid"`), &parsed))
}

func TestParseURLEmpty(t *testing.T) {
	u, err := ParseURL("")
	require.NoError(t, err)
	assert.Nil(t, u)
	assert.Empty(t, u.String())
}

func TestURLSemanticEquality(t *testing.T) {
	a, err := ParseURL("http://my-host/a")
	require.NoError(t, err)
	b, err := ParseURL("http://my-host/a")
	require.NoError(t, err)
	c, err := ParseURL("http://my-host/c")
	require.NoError(t, err)

	assert.True(t, equality.Semantic.DeepEqual(a, b))
	assert.False(t, equality.Semantic.DeepEqual(a, c))
}

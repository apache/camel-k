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

package operator

import (
	"testing"

	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

func TestGetNamespacesSelector(t *testing.T) {
	tests := []struct {
		name              string
		operatorNamespace string
		watchNamespace    string
		expected          map[string]cache.Config
	}{
		{
			name:              "same namespace",
			operatorNamespace: "operator",
			watchNamespace:    "operator",
			expected: map[string]cache.Config{
				"operator": {},
			},
		},
		{
			name:              "different namespace",
			operatorNamespace: "operator",
			watchNamespace:    "tenant",
			expected: map[string]cache.Config{
				"operator": {},
				"tenant":   {},
			},
		},
		{
			name:              "csv namespaces",
			operatorNamespace: "operator",
			watchNamespace:    "tenant-a,tenant-b,tenant-c",
			expected: map[string]cache.Config{
				"operator": {},
				"tenant-a": {},
				"tenant-b": {},
				"tenant-c": {},
			},
		},
		{
			name:              "trim spaces",
			operatorNamespace: "operator",
			watchNamespace:    "tenant-a, tenant-b , tenant-c",
			expected: map[string]cache.Config{
				"operator": {},
				"tenant-a": {},
				"tenant-b": {},
				"tenant-c": {},
			},
		},
		{
			name:              "ignore duplicates",
			operatorNamespace: "operator",
			watchNamespace:    "tenant-a,tenant-a,tenant-b",
			expected: map[string]cache.Config{
				"operator": {},
				"tenant-a": {},
				"tenant-b": {},
			},
		},
		{
			name:              "ignore empty entries",
			operatorNamespace: "operator",
			watchNamespace:    "tenant-a,,tenant-b,",
			expected: map[string]cache.Config{
				"operator": {},
				"tenant-a": {},
				"tenant-b": {},
			},
		},
		{
			name:              "ignore operator namespace in csv",
			operatorNamespace: "operator",
			watchNamespace:    "tenant-a,operator,tenant-b",
			expected: map[string]cache.Config{
				"operator": {},
				"tenant-a": {},
				"tenant-b": {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := getNamespacesSelector(tt.operatorNamespace, tt.watchNamespace)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetWatchNamespace(t *testing.T) {
	t.Run("env variable set", func(t *testing.T) {
		t.Setenv(platform.OperatorWatchNamespaceEnvVariable, "tenant-a,tenant-b")

		ns, err := getWatchNamespace()

		require.NoError(t, err)
		assert.Equal(t, "tenant-a,tenant-b", ns)
	})

	t.Run("env variable not set", func(t *testing.T) {
		ns, err := getWatchNamespace()

		require.Error(t, err)
		assert.Empty(t, ns)
		assert.Contains(t, err.Error(), platform.OperatorWatchNamespaceEnvVariable)
	})
}

func TestGetOperatorImage(t *testing.T) {
	t.Run("env variable set", func(t *testing.T) {
		t.Setenv("CONTAINER_IMAGE", "quay.io/example/operator:latest")

		image := getOperatorImage()

		assert.Equal(t, "quay.io/example/operator:latest", image)
	})

	t.Run("env variable not set", func(t *testing.T) {
		t.Setenv("CONTAINER_IMAGE", "")

		image := getOperatorImage()

		assert.Empty(t, image)
	})
}

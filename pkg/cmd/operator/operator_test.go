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
	"context"
	"testing"

	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

func TestNewMetricsServerOptions(t *testing.T) {
	t.Run("HTTPS without authentication", func(t *testing.T) {
		options := newMetricsServerOptions(8080, true, false, "")

		assert.Equal(t, ":8080", options.BindAddress)
		assert.True(t, options.SecureServing)
		assert.Nil(t, options.FilterProvider)
		assert.Empty(t, options.CertDir)
	})

	t.Run("HTTPS with authentication", func(t *testing.T) {
		options := newMetricsServerOptions(9090, true, true, "/etc/camel-k/metrics-certs")

		assert.Equal(t, ":9090", options.BindAddress)
		assert.True(t, options.SecureServing)
		assert.NotNil(t, options.FilterProvider)
		assert.Equal(t, "/etc/camel-k/metrics-certs", options.CertDir)
	})

	t.Run("legacy HTTP", func(t *testing.T) {
		options := newMetricsServerOptions(7070, false, false, "")

		assert.Equal(t, ":7070", options.BindAddress)
		assert.False(t, options.SecureServing)
		assert.Nil(t, options.FilterProvider)
	})
}

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
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	t.Run("returns operator image", func(t *testing.T) {
		t.Setenv("POD_NAME", "operator-123")
		t.Setenv("NAMESPACE", "default")

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "operator-123",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "camel-k-operator",
						Image: "my-operator:v1.2.3",
					},
				},
			},
		}

		kubeClient, err := internal.NewFakeClient(pod)
		require.NoError(t, err)

		image, err := getOperatorImage(context.Background(), kubeClient)

		require.NoError(t, err)
		assert.Equal(t, "my-operator:v1.2.3", image)
	})

	t.Run("returns error when pod does not exist", func(t *testing.T) {
		t.Setenv("POD_NAME", "operator-123")
		t.Setenv("NAMESPACE", "default")

		kubeClient, err := internal.NewFakeClient()
		require.NoError(t, err)

		image, err := getOperatorImage(context.Background(), kubeClient)

		require.Error(t, err)
		assert.Empty(t, image)
	})

	t.Run("returns error when environment is missing", func(t *testing.T) {
		t.Setenv("POD_NAME", "")
		t.Setenv("NAMESPACE", "default")

		kubeClient, err := internal.NewFakeClient()
		require.NoError(t, err)

		image, err := getOperatorImage(context.Background(), kubeClient)

		require.Error(t, err)
		assert.Empty(t, image)
	})

	t.Run("returns error when operator container is missing", func(t *testing.T) {
		t.Setenv("POD_NAME", "operator-123")
		t.Setenv("NAMESPACE", "default")

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "operator-123",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "some-other-container",
						Image: "other:v1",
					},
				},
			},
		}
		kubeClient, err := internal.NewFakeClient(pod)
		require.NoError(t, err)

		image, err := getOperatorImage(context.Background(), kubeClient)

		require.Error(t, err)
		assert.Empty(t, image)
	})
}

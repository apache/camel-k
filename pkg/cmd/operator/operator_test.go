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
	"io"
	"net/http"
	"strings"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/platform"
	monitorv1alpha1 "github.com/camel-tooling/camel-monitor-operator/pkg/apis/camel/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

type managerRoundTripper func(*http.Request) (*http.Response, error)

func (f managerRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestManagerClientReadsCamelMonitorsWithoutStartingAnInformer(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, monitorv1alpha1.AddToScheme(scheme))
	require.NoError(t, v1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{monitorv1alpha1.SchemeGroupVersion, v1.SchemeGroupVersion})
	mapper.Add(monitorv1alpha1.SchemeGroupVersion.WithKind(monitorv1alpha1.AppKind), meta.RESTScopeNamespace)
	mapper.Add(v1.SchemeGroupVersion.WithKind(v1.IntegrationKind), meta.RESTScopeNamespace)
	integration := v1.NewIntegration("test", "sample")
	cached := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&integration).Build()
	requests := 0
	options := ctrl.Options{
		Scheme: scheme,
		Mapper: mapper,
		Cache:  &ctrl.CacheOptions{Reader: cached, DisableFor: []ctrl.Object{&corev1.Secret{}}},
		HTTPClient: &http.Client{Transport: managerRoundTripper(func(request *http.Request) (*http.Response, error) {
			requests++
			require.Equal(t, http.MethodGet, request.Method)
			require.Equal(t, "/apis/camel.apache.org/v1alpha1/namespaces/test/camelmonitors/sample", request.URL.Path)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": {"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"apiVersion":"camel.apache.org/v1alpha1","kind":"CamelMonitor","metadata":{"name":"sample","namespace":"test"}}`)),
			}, nil
		})},
	}
	c, err := newManagerClient(&rest.Config{Host: "http://api.invalid"}, options)
	require.NoError(t, err)
	monitor := monitorv1alpha1.NewCamelMonitor("test", "sample")
	require.NoError(t, c.Get(context.Background(), ctrl.ObjectKeyFromObject(&monitor), &monitor))
	require.Equal(t, 1, requests, "optional monitors must use a direct GET rather than the cache's list/watch")
	require.Contains(t, options.Cache.DisableFor, &corev1.Secret{}, "retain caller cache exclusions")
	// Other types still use the manager cache, without another API request.
	var fromCache v1.Integration
	require.NoError(t, c.Get(context.Background(), ctrl.ObjectKeyFromObject(&integration), &fromCache))
	require.Equal(t, integration.Name, fromCache.Name)
	require.Equal(t, 1, requests)
}

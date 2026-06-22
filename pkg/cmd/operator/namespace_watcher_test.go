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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func namespace(name string, labelSet map[string]string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labelSet},
	}
}

// newReviewer builds a fake client preloaded with the given namespaces. SelfSubjectAccessReview
// creations are answered locally: a namespace is "allowed" unless it appears in denied.
func newReviewer(namespaces []*corev1.Namespace, denied map[string]bool) ctrl.Client {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = authorizationv1.AddToScheme(scheme)

	objs := make([]ctrl.Object, 0, len(namespaces))
	for _, ns := range namespaces {
		objs = append(objs, ns)
	}

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c ctrl.WithWatch, obj ctrl.Object, opts ...ctrl.CreateOption) error {
				if ssar, ok := obj.(*authorizationv1.SelfSubjectAccessReview); ok {
					ns := ssar.Spec.ResourceAttributes.Namespace
					ssar.Status.Allowed = !denied[ns]

					return nil
				}

				return c.Create(ctx, obj, opts...)
			},
		}).
		Build()
}

func mustSelector(t *testing.T, s string) labels.Selector {
	t.Helper()
	sel, err := labels.Parse(s)
	require.NoError(t, err)

	return sel
}

func TestComputeWatchedNamespaces(t *testing.T) {
	enabled := map[string]string{"camel-k-enabled": "true"}

	t.Run("operator namespace is always included, even without an explicit SSAR allow", func(t *testing.T) {
		reviewer := newReviewer(nil, map[string]bool{"camel-k": true})
		got, err := computeWatchedNamespaces(context.Background(), reviewer, nil, nil, "camel-k", nil, nil)
		require.NoError(t, err)
		assert.Equal(t, map[string]bool{"camel-k": true}, got)
	})

	t.Run("static namespaces are included when accessible", func(t *testing.T) {
		reviewer := newReviewer(nil, nil)
		got, err := computeWatchedNamespaces(context.Background(), reviewer, nil, nil,
			"camel-k", []string{"team-a", "team-b"}, nil)
		require.NoError(t, err)
		assert.Equal(t, map[string]bool{"camel-k": true, "team-a": true, "team-b": true}, got)
	})

	t.Run("inaccessible namespaces are skipped, not failed", func(t *testing.T) {
		reviewer := newReviewer(nil, map[string]bool{"team-b": true})
		got, err := computeWatchedNamespaces(context.Background(), reviewer, nil, nil,
			"camel-k", []string{"team-a", "team-b"}, nil)
		require.NoError(t, err)
		assert.Equal(t, map[string]bool{"camel-k": true, "team-a": true}, got)
	})

	t.Run("selector discovers labelled and accessible namespaces", func(t *testing.T) {
		namespaces := []*corev1.Namespace{
			namespace("team-a", enabled),
			namespace("team-b", enabled),
			namespace("team-c", nil), // not labelled -> not discovered
		}
		reviewer := newReviewer(namespaces, map[string]bool{"team-b": true}) // team-b labelled but no RBAC
		got, err := computeWatchedNamespaces(context.Background(), reviewer, nil, nil,
			"camel-k", nil, mustSelector(t, "camel-k-enabled=true"))
		require.NoError(t, err)
		assert.Equal(t, map[string]bool{"camel-k": true, "team-a": true}, got)
	})

	t.Run("static list and selector are unioned", func(t *testing.T) {
		namespaces := []*corev1.Namespace{namespace("dyn", enabled)}
		reviewer := newReviewer(namespaces, nil)
		got, err := computeWatchedNamespaces(context.Background(), reviewer, nil, nil,
			"camel-k", []string{"static"}, mustSelector(t, "camel-k-enabled=true"))
		require.NoError(t, err)
		assert.Equal(t, map[string]bool{"camel-k": true, "static": true, "dyn": true}, got)
	})
}

func TestNamespaceWatcherChanged(t *testing.T) {
	enabled := map[string]string{"camel-k-enabled": "true"}
	selector := mustSelector(t, "camel-k-enabled=true")

	newWatcher := func(reviewer ctrl.Client, current map[string]bool) *namespaceWatcher {
		return &namespaceWatcher{
			reviewer:          reviewer,
			selector:          selector,
			operatorNamespace: "camel-k",
			current:           current,
		}
	}

	t.Run("no change when desired equals current", func(t *testing.T) {
		reviewer := newReviewer([]*corev1.Namespace{namespace("team-a", enabled)}, nil)
		w := newWatcher(reviewer, map[string]bool{"camel-k": true, "team-a": true})
		assert.False(t, w.changed(context.Background()))
	})

	t.Run("change when a new namespace is labelled", func(t *testing.T) {
		reviewer := newReviewer([]*corev1.Namespace{
			namespace("team-a", enabled),
			namespace("team-b", enabled),
		}, nil)
		w := newWatcher(reviewer, map[string]bool{"camel-k": true, "team-a": true})
		assert.True(t, w.changed(context.Background()))
	})

	t.Run("change when a watched namespace disappears", func(t *testing.T) {
		reviewer := newReviewer([]*corev1.Namespace{namespace("team-a", enabled)}, nil)
		w := newWatcher(reviewer, map[string]bool{"camel-k": true, "team-a": true, "team-gone": true})
		assert.True(t, w.changed(context.Background()))
	})
}

func TestToCacheNamespaces(t *testing.T) {
	assert.Empty(t, toCacheNamespaces(nil))
	got := toCacheNamespaces(map[string]bool{"a": true, "b": true})
	assert.Len(t, got, 2)
	assert.Contains(t, got, "a")
	assert.Contains(t, got, "b")
}

func TestSortedKeys(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c"}, sortedKeys(map[string]bool{"c": true, "a": true, "b": true}))
	assert.Empty(t, sortedKeys(nil))
}

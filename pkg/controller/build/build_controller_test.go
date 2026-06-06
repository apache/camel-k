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

package build

import (
	"context"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileBuild(t *testing.T) {
	ctx := context.TODO()

	build := &v1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-build",
			Namespace: "default",
		},
		Spec: v1.BuildSpec{
			Tasks: []v1.Task{
				{
					Builder: &v1.BuilderTask{
						BaseTask: v1.BaseTask{
							Name: "builder",
							Configuration: v1.BuildConfiguration{
								Strategy: v1.BuildStrategyRoutine,
							},
						},
					},
				},
			},
		},
	}

	c, err := internal.NewFakeClient(build)
	require.NoError(t, err)

	r := &reconcileBuild{
		client:   c,
		reader:   c,
		recorder: &internal.FakeRecorder{},
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-build",
			Namespace: "default",
		},
	}

	result, err := r.Reconcile(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: requeueAfterDuration}, result)

	var updated v1.Build
	err = c.Get(ctx, req.NamespacedName, &updated)

	require.NoError(t, err)
	assert.NotNil(t, updated.Status, "status should not be nil")
	assert.NotEmpty(t, updated.Status.Phase, "phase should be set")
}

// TestReconcileBuildDeletedDuringReconcile verifies that a Build deleted mid-reconcile does not
// surface as a reconcile error: a NotFound from the status write is benign (#6620).
func TestReconcileBuildDeletedDuringReconcile(t *testing.T) {
	ctx := context.TODO()

	build := &v1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-build",
			Namespace: "default",
		},
		Spec: v1.BuildSpec{
			Tasks: []v1.Task{
				{
					Builder: &v1.BuilderTask{
						BaseTask: v1.BaseTask{
							Name: "builder",
							Configuration: v1.BuildConfiguration{
								Strategy: v1.BuildStrategyRoutine,
							},
						},
					},
				},
			},
		},
	}

	c, err := internal.NewFakeClient(build)
	require.NoError(t, err)

	// Simulate the Build being deleted concurrently. The fake client routes a status patch
	// through Update, so the NotFound is injected there.
	fakeClient := c.(*internal.FakeClient)
	fakeClient.Intercept(&interceptor.Funcs{
		Update: func(ctx context.Context, cl ctrl.WithWatch, obj ctrl.Object, opts ...ctrl.UpdateOption) error {
			if _, ok := obj.(*v1.Build); ok {
				return k8serrors.NewNotFound(v1.Resource("builds"), obj.GetName())
			}

			return cl.Update(ctx, obj, opts...)
		},
	})

	r := &reconcileBuild{
		client:   c,
		reader:   c,
		recorder: &internal.FakeRecorder{},
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-build",
			Namespace: "default",
		},
	}

	result, err := r.Reconcile(ctx, req)

	// The NotFound must be swallowed: no error, and no requeue since the Build is gone.
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

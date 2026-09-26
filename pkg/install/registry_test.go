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

package install

import (
	"context"
	"testing"
	"time"

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func operatorDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "camel-k-operator",
			Namespace: "camel-k",
			UID:       types.UID("operator-uid"),
		},
	}
}

func TestGetOwnerRef(t *testing.T) {
	t.Run("returns owner reference", func(t *testing.T) {
		fakeClient, err := internal.NewFakeClient(operatorDeployment())
		require.NoError(t, err)

		ref, err := getOwnerRef(
			context.Background(),
			fakeClient,
			"camel-k-operator",
			"camel-k",
		)

		require.NoError(t, err)
		require.NotNil(t, ref)

		assert.Equal(t, "apps/v1", ref.APIVersion)
		assert.Equal(t, "Deployment", ref.Kind)
		assert.Equal(t, "camel-k-operator", ref.Name)
		assert.Equal(t, types.UID("operator-uid"), ref.UID)

		require.NotNil(t, ref.Controller)
		assert.True(t, *ref.Controller)

		require.NotNil(t, ref.BlockOwnerDeletion)
		assert.True(t, *ref.BlockOwnerDeletion)
	})

	t.Run("returns error when operator deployment does not exist", func(t *testing.T) {
		fakeClient, err := internal.NewFakeClient()
		require.NoError(t, err)

		_, err = getOwnerRef(
			context.Background(),
			fakeClient,
			"camel-k-operator",
			"camel-k",
		)

		require.Error(t, err)
		assert.True(t, apierrors.IsNotFound(err))
	})
}

func TestWaitForClusterIP(t *testing.T) {
	t.Run("returns cluster IP", func(t *testing.T) {
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "registry",
				Namespace: "camel-k",
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "10.96.0.20",
			},
		}

		fakeClient, err := internal.NewFakeClient(service)
		require.NoError(t, err)

		ip, err := waitForClusterIP(
			context.Background(),
			fakeClient,
			"camel-k",
			"registry",
			time.Second,
		)

		require.NoError(t, err)
		assert.Equal(t, "10.96.0.20", ip)
	})

	t.Run("returns get error", func(t *testing.T) {
		fakeClient, err := internal.NewFakeClient()
		require.NoError(t, err)

		_, err = waitForClusterIP(
			context.Background(),
			fakeClient,
			"camel-k",
			"registry",
			time.Second,
		)

		require.Error(t, err)
		assert.True(t, apierrors.IsNotFound(err))
	})

	t.Run("times out when ClusterIP is not assigned", func(t *testing.T) {
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "registry",
				Namespace: "camel-k",
			},
		}

		fakeClient, err := internal.NewFakeClient(service)
		require.NoError(t, err)

		_, err = waitForClusterIP(
			context.Background(),
			fakeClient,
			"camel-k",
			"registry",
			10*time.Millisecond,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "timed out waiting for Service")
	})

	t.Run("times out for headless service", func(t *testing.T) {
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "registry",
				Namespace: "camel-k",
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: corev1.ClusterIPNone,
			},
		}

		fakeClient, err := internal.NewFakeClient(service)
		require.NoError(t, err)

		_, err = waitForClusterIP(
			context.Background(),
			fakeClient,
			"camel-k",
			"registry",
			10*time.Millisecond,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "timed out waiting for Service")
	})
}

func TestReplace(t *testing.T) {
	ctx := context.Background()

	t.Run("creates object when it does not exist", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "camel-k",
			},
			Data: map[string][]byte{
				"foo": []byte("bar"),
			},
		}

		fakeClient, err := internal.NewFakeClient()
		require.NoError(t, err)

		err = replace(ctx, fakeClient, secret)
		require.NoError(t, err)

		var actual corev1.Secret
		require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{
			Name:      "test-secret",
			Namespace: "camel-k",
		}, &actual))

		assert.Equal(t, []byte("bar"), actual.Data["foo"])
	})

	t.Run("updates existing object", func(t *testing.T) {
		existing := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "camel-k",
			},
			Data: map[string][]byte{
				"foo": []byte("old"),
			},
		}

		fakeClient, err := internal.NewFakeClient(existing)
		require.NoError(t, err)

		updated := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "camel-k",
			},
			Data: map[string][]byte{
				"foo": []byte("new"),
			},
		}

		err = replace(ctx, fakeClient, updated)
		require.NoError(t, err)

		var actual corev1.Secret
		require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{
			Name:      "test-secret",
			Namespace: "camel-k",
		}, &actual))

		assert.Equal(t, []byte("new"), actual.Data["foo"])
	})
}

func TestOperatorStartupRegistry(t *testing.T) {
	t.Setenv("NAMESPACE", "camel-k")

	crtSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-secret",
			Namespace: "camel-k",
		},
		Data: map[string][]byte{
			"foo": []byte("bar"),
		},
	}

	fakeClient, err := internal.NewFakeClient(operatorDeployment(), crtSecret)
	require.NoError(t, err)

	// This is running in parallel, simulating the setting of an IP, when the Service is create by the
	// OperatorStartupRegistry func.
	setServiceClusterIP(context.Background(), fakeClient, "camel-k", "registry", "10.96.0.20")

	conf, err := OperatorStartupRegistry(
		context.Background(),
		fakeClient,
		false,
		crtSecret,
	)

	require.NoError(t, err)
	require.NotNil(t, conf)

	assert.Equal(t, "10.96.0.20", conf.clusterIP)
	assert.Equal(t, "false", conf.insecure)
	assert.Equal(t, "ck-dev-registry", conf.dockerRegistrySecretName)
}

// Useful to simulate Kubernetes IP assignment.
func setServiceClusterIP(ctx context.Context, c client.Client, namespace string, name string, clusterIP string) {
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			var service corev1.Service

			err := c.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, &service)

			if err == nil {
				service.Spec.ClusterIP = clusterIP
				_ = c.Update(ctx, &service)
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

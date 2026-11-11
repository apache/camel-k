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

package integration

import (
	"context"
	"errors"
	"testing"

	monitorv1alpha1 "github.com/camel-tooling/camel-monitor-operator/pkg/apis/camel/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/log"
)

func TestCamelMonitorCondition(t *testing.T) {
	ctx := context.Background()
	integration := v1.NewIntegration("test", "sample")
	integration.Status.Phase = v1.IntegrationPhaseRunning
	integration.SetReadyCondition(corev1.ConditionTrue, "Ready", "running")
	ready := integration.Status.GetCondition(v1.IntegrationConditionReady).DeepCopy()
	foreign := monitorv1alpha1.NewCamelMonitor("other", integration.Name)
	different := monitorv1alpha1.NewCamelMonitor(integration.Namespace, "different-name")
	different.Labels = map[string]string{v1.IntegrationLabel: integration.Name}
	c, err := internal.NewFakeClient()
	require.NoError(t, err)
	require.NoError(t, c.Create(ctx, &foreign))
	require.NoError(t, c.Create(ctx, &different))
	require.NoError(t, addCamelMonitorCondition(ctx, c, &integration))
	require.Nil(t, integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))

	monitor := monitorv1alpha1.NewCamelMonitor(integration.Namespace, integration.Name)
	monitor.Status.Conditions = []metav1.Condition{{Type: "Healthy", Status: metav1.ConditionFalse}}
	require.NoError(t, c.Create(ctx, &monitor))
	require.NoError(t, addCamelMonitorCondition(ctx, c, &integration))
	condition := integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable).DeepCopy()
	require.Equal(t, corev1.ConditionTrue, condition.Status)
	require.NoError(t, addCamelMonitorCondition(ctx, c, &integration))
	require.Equal(t, condition, integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	require.NoError(t, c.Delete(ctx, &monitor))
	require.NoError(t, addCamelMonitorCondition(ctx, c, &integration))
	require.Nil(t, integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	require.Equal(t, ready, integration.Status.GetCondition(v1.IntegrationConditionReady))
	require.Equal(t, v1.IntegrationPhaseRunning, integration.Status.Phase)
}

func TestCamelMonitorLookupFailure(t *testing.T) {
	integration := v1.NewIntegration("test", "sample")
	integration.Status.SetCondition(v1.IntegrationConditionCamelMonitorAvailable, corev1.ConditionTrue, "Available", "found")
	previous := integration.Status.DeepCopy()
	c, err := internal.NewFakeClient()
	require.NoError(t, err)
	lookupErr := errors.New("lookup unavailable")
	c.(*internal.FakeClient).Intercept(&interceptor.Funcs{Get: func(context.Context, ctrl.WithWatch, ctrl.ObjectKey, ctrl.Object, ...ctrl.GetOption) error {
		return lookupErr
	}})
	require.ErrorIs(t, addCamelMonitorCondition(context.Background(), c, &integration), lookupErr)
	require.Equal(t, previous, &integration.Status)
}

func enableCamelMonitorDiscovery(t *testing.T, c client.Client) *fake.FakeDiscovery {
	t.Helper()
	fakeClient, ok := c.(*internal.FakeClient)
	require.True(t, ok)
	discovery, ok := fakeClient.Interface.Discovery().(*fake.FakeDiscovery)
	require.True(t, ok)
	discovery.Resources = []*metav1.APIResourceList{{GroupVersion: monitorv1alpha1.SchemeGroupVersion.String(),
		APIResources: []metav1.APIResource{{Kind: monitorv1alpha1.AppKind, Name: "camelmonitors", Namespaced: true}}}}

	return discovery
}

func TestMonitorActionCachesCamelMonitorAvailability(t *testing.T) {
	for _, installed := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent", true: "installed"}[installed], func(t *testing.T) {
			c, integration, err := nominalEnvironment()
			require.NoError(t, err)
			if installed {
				enableCamelMonitorDiscovery(t, c)
			}
			monitor := monitorv1alpha1.NewCamelMonitor(integration.Namespace, integration.Name)
			require.NoError(t, c.Create(context.Background(), &monitor))
			action := NewMonitorAction(installed)
			action.InjectClient(c)
			action.InjectLogger(log.Log)
			if installed {
				c.(*internal.FakeClient).Interface.Discovery().(*fake.FakeDiscovery).Resources = nil
			} else {
				enableCamelMonitorDiscovery(t, c)
			}
			discoveryRequests := 0
			c.(*internal.FakeClient).Interface.Discovery().(*fake.FakeDiscovery).PrependReactor("get", "resource", func(k8stesting.Action) (bool, runtime.Object, error) {
				discoveryRequests++

				return true, nil, errors.New("unexpected discovery during reconciliation")
			})
			for range 2 {
				result, err := action.Handle(context.Background(), integration)
				require.NoError(t, err)
				require.NotNil(t, result)
				condition := result.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable)
				if installed {
					require.NotNil(t, condition)
					require.Equal(t, corev1.ConditionTrue, condition.Status)
				} else {
					require.Nil(t, condition)
				}
			}
			require.Zero(t, discoveryRequests)
		})
	}
}

func TestCamelMonitorOptionalWatch(t *testing.T) {
	c, err := internal.NewFakeClient()
	require.NoError(t, err)
	require.NoError(t, watchCamelMonitorResources(c, nil))
	discovery := enableCamelMonitorDiscovery(t, c)
	require.NoError(t, watchCamelMonitorResources(c, &builder.Builder{}))
	lookupErr := errors.New("discovery unavailable")
	discovery.PrependReactor("get", "resource", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, lookupErr
	})
	require.ErrorIs(t, watchCamelMonitorResources(c, &builder.Builder{}), lookupErr)
}

func TestAddCamelMonitorDiscoveryFailure(t *testing.T) {
	c, err := internal.NewFakeClient()
	require.NoError(t, err)
	discovery := enableCamelMonitorDiscovery(t, c)
	lookupErr := errors.New("discovery unavailable")
	discovery.PrependReactor("get", "resource", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, lookupErr
	})
	require.ErrorIs(t, Add(context.Background(), camelMonitorTestManager{}, c), lookupErr)
}

type camelMonitorTestManager struct {
	manager.Manager
}

func (camelMonitorTestManager) GetFieldIndexer() ctrl.FieldIndexer {
	return camelMonitorTestFieldIndexer{}
}

type camelMonitorTestFieldIndexer struct{}

func (camelMonitorTestFieldIndexer) IndexField(context.Context, ctrl.Object, string, ctrl.IndexerFunc) error {
	return nil
}

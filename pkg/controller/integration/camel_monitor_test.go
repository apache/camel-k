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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	servingv1 "github.com/apache/camel-k/v2/pkg/apis/duck/knative/serving/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
	monitorv1alpha1 "github.com/camel-tooling/camel-monitor-operator/pkg/apis/camel/v1alpha1"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestMonitorActionCamelMonitorLifecycle(t *testing.T) {
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "camel.apache.org/monitor")
	ctx := context.Background()
	c, it, err := nominalEnvironment()
	require.NoError(t, err)
	enableCamelMonitorDiscovery(t, c)
	action := monitorAction{}
	action.InjectLogger(log.Log)
	action.InjectClient(c)
	result, err := action.Handle(ctx, it)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Nil(t, it.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	ready, phase := *it.Status.GetCondition(v1.IntegrationConditionReady), it.Status.Phase
	monitors := []monitorv1alpha1.CamelMonitor{
		monitorv1alpha1.NewCamelMonitor(it.Namespace, "first"),
		monitorv1alpha1.NewCamelMonitor(it.Namespace, "second"),
		monitorv1alpha1.NewCamelMonitor("other", "foreign"),
	}
	for i := range monitors {
		monitors[i].Labels = kubernetes.DeploymentLabels(it.Name)
		require.NoError(t, c.Create(ctx, &monitors[i]))
	}
	result, err = action.Handle(ctx, it)
	require.NoError(t, err)
	condition := *result.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable)
	require.Equal(t, corev1.ConditionTrue, condition.Status)
	_, err = action.Handle(ctx, it)
	require.NoError(t, err)
	require.Equal(t, condition, *it.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	require.NoError(t, c.Delete(ctx, &monitors[0]))
	_, err = action.Handle(ctx, it)
	require.NoError(t, err)
	require.Equal(t, corev1.ConditionTrue, it.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable).Status)
	require.NoError(t, c.Delete(ctx, &monitors[1]))
	_, err = action.Handle(ctx, it)
	require.NoError(t, err)
	require.Nil(t, it.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	require.Equal(t, ready, *it.Status.GetCondition(v1.IntegrationConditionReady))
	require.Equal(t, phase, it.Status.Phase)
}

func TestCamelMonitorSyntheticAssociationChanges(t *testing.T) {
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "example.org/monitor")
	ctx := context.Background()
	integration := v1.NewIntegration("test", "sample")
	cron := &batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "source",
		Labels: map[string]string{v1.IntegrationLabel: integration.Name, "example.org/monitor": "custom-app"}}}
	monitor := monitorv1alpha1.NewCamelMonitor("test", "different-name")
	monitor.Annotations = map[string]string{monitorv1alpha1.MonitorImportedKindLabel: "CronJob", monitorv1alpha1.MonitorImportedNameLabel: cron.Name}
	c, err := internal.NewFakeClient(cron)
	require.NoError(t, err)
	require.NoError(t, c.Create(ctx, &monitor))
	require.NoError(t, updateCamelMonitorCondition(ctx, c, &integration))
	require.Equal(t, corev1.ConditionTrue, integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable).Status)
	// The monitor still imports the same source when its app identifier changes.
	cron.Labels["example.org/monitor"] = "new-app"
	require.NoError(t, c.Update(ctx, cron))
	require.NoError(t, updateCamelMonitorCondition(ctx, c, &integration))
	require.NotNil(t, integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	cron.Labels[v1.IntegrationLabel] = "another"
	require.NoError(t, c.Update(ctx, cron))
	require.NoError(t, updateCamelMonitorCondition(ctx, c, &integration))
	require.Nil(t, integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	other := v1.NewIntegration("test", "another")
	require.NoError(t, updateCamelMonitorCondition(ctx, c, &other))
	require.NotNil(t, other.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	delete(cron.Labels, "example.org/monitor")
	require.NoError(t, c.Update(ctx, cron))
	require.NoError(t, updateCamelMonitorCondition(ctx, c, &other))
	require.Nil(t, other.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	cron.Labels["example.org/monitor"] = "enabled"
	require.NoError(t, c.Update(ctx, cron))
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "")
	require.NoError(t, updateCamelMonitorCondition(ctx, c, &other))
	require.Nil(t, other.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "example.org/monitor")
	require.NoError(t, c.Delete(ctx, cron))
	require.NoError(t, updateCamelMonitorCondition(ctx, c, &other))
	require.Nil(t, other.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
}

func interceptMonitorLists(t *testing.T, c client.Client, lookupErr error) *int {
	t.Helper()
	fakeClient, ok := c.(*internal.FakeClient)
	require.True(t, ok)
	calls := new(int)
	fakeClient.Intercept(&interceptor.Funcs{List: func(ctx context.Context, c ctrl.WithWatch, list ctrl.ObjectList, opts ...ctrl.ListOption) error {
		if _, ok := list.(*monitorv1alpha1.CamelMonitorList); ok {
			*calls++

			if lookupErr != nil {
				return lookupErr
			}
		}

		return c.List(ctx, list, opts...)
	}})

	return calls
}

func TestMonitorActionCamelMonitorLookupFailure(t *testing.T) {
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "camel.apache.org/monitor")
	for _, lookupErr := range []error{errors.New("connection reset"), k8serrors.NewForbidden(schema.GroupResource{Group: "camel.apache.org", Resource: "camelmonitors"}, "", errors.New("denied"))} {
		t.Run(lookupErr.Error(), func(t *testing.T) {
			c, it, err := nominalEnvironment()
			require.NoError(t, err)
			interceptMonitorLists(t, c, lookupErr)
			it.Status.Phase = v1.IntegrationPhaseDeploying
			it.Status.SetCondition(v1.IntegrationConditionReady, corev1.ConditionFalse, "Pending", "waiting")
			enableCamelMonitorDiscovery(t, c)
			action := monitorAction{}
			action.InjectClient(c)
			action.InjectLogger(log.Log)
			result, err := action.Handle(context.Background(), it)
			require.ErrorIs(t, err, lookupErr)
			require.NotNil(t, result)
			require.Equal(t, corev1.ConditionUnknown, result.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable).Status)
			require.Equal(t, corev1.ConditionTrue, result.Status.GetCondition(v1.IntegrationConditionReady).Status)
			require.Equal(t, v1.IntegrationPhaseRunning, result.Status.Phase)
		})
	}
}

func TestCamelMonitorSourceLookupFailure(t *testing.T) {
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "example.org/monitor")
	ctx := context.Background()
	integration := v1.NewIntegration("test", "sample")
	monitor := monitorv1alpha1.NewCamelMonitor("test", "custom-app")
	monitor.Annotations = map[string]string{monitorv1alpha1.MonitorImportedKindLabel: "Deployment", monitorv1alpha1.MonitorImportedNameLabel: "source"}
	c, err := internal.NewFakeClient()
	require.NoError(t, err)
	require.NoError(t, c.Create(ctx, &monitor))
	fakeClient, ok := c.(*internal.FakeClient)
	require.True(t, ok)
	lookupErr := errors.New("source lookup failed")
	fakeClient.Intercept(&interceptor.Funcs{Get: func(context.Context, ctrl.WithWatch, ctrl.ObjectKey, ctrl.Object, ...ctrl.GetOption) error {
		return lookupErr
	}})
	require.ErrorIs(t, updateCamelMonitorCondition(ctx, c, &integration), lookupErr)
	require.Equal(t, corev1.ConditionUnknown, integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable).Status)
	// A positively identified monitor is enough even when another source cannot be read.
	other := monitorv1alpha1.NewCamelMonitor("test", "second")
	other.Labels = map[string]string{"example.org/monitor": integration.Name}
	require.NoError(t, c.Create(ctx, &other))
	require.NoError(t, updateCamelMonitorCondition(ctx, c, &integration))
	require.Equal(t, corev1.ConditionTrue, integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable).Status)
}

func TestMonitorActionCamelMonitorDisabledAndEarlyExit(t *testing.T) {
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "camel.apache.org/monitor")
	c, it, err := nominalEnvironment()
	require.NoError(t, err)
	calls := interceptMonitorLists(t, c, errors.New("must not read optional monitors"))
	it.Status.SetCondition(v1.IntegrationConditionCamelMonitorAvailable, corev1.ConditionTrue, "CamelMonitorAvailable", "")
	action := monitorAction{}
	action.InjectClient(c)
	action.InjectLogger(log.Log)
	_, err = action.Handle(context.Background(), it)
	require.NoError(t, err)
	require.Zero(t, *calls)
	require.Nil(t, it.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))

	c, it, err = nominalEnvironment()
	require.NoError(t, err)
	it.Status.Phase = v1.IntegrationPhaseError
	it.Status.SetCondition(v1.IntegrationConditionReady, corev1.ConditionFalse, v1.IntegrationConditionInitializationFailedReason, "failed")
	ready := *it.Status.GetCondition(v1.IntegrationConditionReady)
	enableCamelMonitorDiscovery(t, c)
	action = monitorAction{}
	action.InjectClient(c)
	action.InjectLogger(log.Log)
	result, err := action.Handle(context.Background(), it)
	require.NoError(t, err)
	require.Nil(t, result)
	monitor := monitorv1alpha1.NewCamelMonitor(it.Namespace, "sample")
	monitor.Labels = kubernetes.DeploymentLabels(it.Name)
	require.NoError(t, c.Create(context.Background(), &monitor))
	result, err = action.Handle(context.Background(), it)
	require.NoError(t, err)
	require.Same(t, it, result)
	result, err = action.Handle(context.Background(), it)
	require.NoError(t, err)
	require.Nil(t, result)
	require.NoError(t, c.Delete(context.Background(), &monitor))
	result, err = action.Handle(context.Background(), it)
	require.NoError(t, err)
	require.Same(t, it, result)
	require.Nil(t, it.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	require.Equal(t, ready, *it.Status.GetCondition(v1.IntegrationConditionReady))
}

func TestCamelMonitorEvents(t *testing.T) {
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "example.org/monitor")
	t.Setenv("WATCH_NAMESPACE", "test")
	t.Setenv("NAMESPACE", "test")
	first, second := v1.NewIntegration("test", "first"), v1.NewIntegration("test", "second")
	first.Status.Phase, second.Status.Phase = v1.IntegrationPhaseRunning, v1.IntegrationPhaseError
	foreign := v1.NewIntegration("other", "first")
	foreign.Status.Phase = v1.IntegrationPhaseRunning
	otherOperator := v1.NewIntegration("test", "assigned")
	otherOperator.Status.Phase = v1.IntegrationPhaseRunning
	otherOperator.Annotations = map[string]string{v1.OperatorIDAnnotation: "another-operator"}
	pending := v1.NewIntegration("test", "building")
	pending.Status.Phase = v1.IntegrationPhaseBuildingKit
	c, err := internal.NewFakeClient(&first, &second, &foreign, &otherOperator, &pending)
	require.NoError(t, err)
	monitor := monitorv1alpha1.NewCamelMonitor("test", "different-name")
	monitor.Labels = map[string]string{"example.org/monitor": first.Name}
	ctx := context.Background()
	for _, name := range []string{otherOperator.Name, pending.Name, "missing"} {
		unrelated := monitor.DeepCopy()
		unrelated.Labels["example.org/monitor"] = name
		require.Empty(t, camelMonitorEnqueueRequests(ctx, c, unrelated))
	}
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
	defer queue.ShutDown()
	handler := camelMonitorEventHandler(c)
	assertRequests := func(names ...string) {
		t.Helper()
		var requests []string
		for queue.Len() > 0 {
			request, shutdown := queue.Get()
			require.False(t, shutdown)
			require.Equal(t, "test", request.Namespace)
			requests = append(requests, request.Name)
			queue.Done(request)
		}
		require.ElementsMatch(t, names, requests)
	}
	handler.Create(ctx, event.CreateEvent{Object: &monitor}, queue)
	assertRequests(first.Name)
	handler.Delete(ctx, event.DeleteEvent{Object: &monitor}, queue)
	assertRequests(first.Name)
	changed := monitor.DeepCopy()
	changed.Labels["example.org/monitor"] = second.Name
	update := event.UpdateEvent{ObjectOld: &monitor, ObjectNew: changed}
	predicate := camelMonitorAssociationChangedPredicate()
	require.True(t, predicate.Update(update))
	handler.Update(ctx, update, queue)
	assertRequests(first.Name, second.Name)
	removed := monitor.DeepCopy()
	delete(removed.Labels, "example.org/monitor")
	update = event.UpdateEvent{ObjectOld: &monitor, ObjectNew: removed}
	require.True(t, predicate.Update(update))
	handler.Update(ctx, update, queue)
	assertRequests(first.Name)
	status := monitor.DeepCopy()
	status.Status.Phase = monitorv1alpha1.CamelMonitorPhaseRunning
	require.False(t, predicate.Update(event.UpdateEvent{ObjectOld: &monitor, ObjectNew: status}))
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

func TestCamelMonitorOptionalWatch(t *testing.T) {
	c, err := internal.NewFakeClient()
	require.NoError(t, err)
	// Without the API no source should be registered on the builder.
	require.NoError(t, watchCamelMonitorResources(c, nil))
	discovery := enableCamelMonitorDiscovery(t, c)
	require.NoError(t, watchCamelMonitorResources(c, &builder.Builder{}))
	lookupErr := errors.New("discovery unavailable")
	discovery.PrependReactor("get", "resource", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, lookupErr
	})
	require.ErrorIs(t, watchCamelMonitorResources(c, &builder.Builder{}), lookupErr)
}

func TestMonitorActionCamelMonitorDiscoversAPIOnEachReconcile(t *testing.T) {
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "camel.apache.org/monitor")
	ctx := context.Background()
	c, it, err := nominalEnvironment()
	require.NoError(t, err)
	calls := interceptMonitorLists(t, c, nil)
	monitor := monitorv1alpha1.NewCamelMonitor(it.Namespace, "sample")
	monitor.Labels = kubernetes.DeploymentLabels(it.Name)
	require.NoError(t, c.Create(ctx, &monitor))
	action := NewMonitorAction()
	action.InjectClient(c)
	action.InjectLogger(log.Log)
	_, err = action.Handle(ctx, it)
	require.NoError(t, err)
	require.Zero(t, *calls)
	require.Nil(t, it.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	discovery := enableCamelMonitorDiscovery(t, c)
	_, err = action.Handle(ctx, it)
	require.NoError(t, err)
	require.Equal(t, 1, *calls)
	require.Equal(t, corev1.ConditionTrue, it.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable).Status)
	lookupErr := errors.New("discovery unavailable")
	discovery.PrependReactor("get", "resource", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, lookupErr
	})
	_, err = action.Handle(ctx, it)
	require.ErrorIs(t, err, lookupErr)
	require.Equal(t, 1, *calls)
	require.Equal(t, corev1.ConditionUnknown, it.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable).Status)
	require.Equal(t, corev1.ConditionTrue, it.Status.GetCondition(v1.IntegrationConditionReady).Status)
}

func TestCamelMonitorKnativeService(t *testing.T) {
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "example.org/monitor")
	t.Setenv("WATCH_NAMESPACE", "test")
	t.Setenv("NAMESPACE", "test")
	ctx := context.Background()
	integration := v1.NewIntegration("test", "sample")
	integration.Status.Phase = v1.IntegrationPhaseRunning
	service := &servingv1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: servingv1.SchemeGroupVersion.String(), Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "sample-service",
			Labels: map[string]string{v1.IntegrationLabel: integration.Name, "example.org/monitor": "custom-app"}},
	}
	c, err := internal.NewFakeClient(&integration, service)
	require.NoError(t, err)
	monitor := monitorv1alpha1.NewCamelMonitor("test", "custom-app-revision")
	monitor.Annotations = map[string]string{monitorv1alpha1.MonitorImportedKindLabel: "Service", monitorv1alpha1.MonitorImportedNameLabel: service.Name}
	require.NoError(t, c.Create(ctx, &monitor))
	require.NoError(t, updateCamelMonitorCondition(ctx, c, &integration))
	require.Equal(t, corev1.ConditionTrue, integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable).Status)
	require.Equal(t, []reconcile.Request{{NamespacedName: ctrl.ObjectKeyFromObject(&integration)}}, camelMonitorEnqueueRequests(ctx, c, &monitor))
	require.NoError(t, c.Delete(ctx, service))
	require.NoError(t, updateCamelMonitorCondition(ctx, c, &integration))
	require.Nil(t, integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
}

func TestCamelMonitorLabeledResourceWithDifferentName(t *testing.T) {
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "example.org/monitor")
	integration := v1.NewIntegration("test", "sample")
	monitor := monitorv1alpha1.NewCamelMonitor("test", "custom-monitor")
	monitor.Labels = map[string]string{"example.org/monitor": integration.Name}
	c, err := internal.NewFakeClient()
	require.NoError(t, err)
	require.NoError(t, c.Create(context.Background(), &monitor))

	require.NoError(t, updateCamelMonitorCondition(context.Background(), c, &integration))
	condition := integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable)
	require.NotNil(t, condition)
	require.Equal(t, corev1.ConditionTrue, condition.Status)
}

func TestCamelMonitorSyntheticResourceUsesDiscoveryLabel(t *testing.T) {
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "example.org/monitor")
	integration := v1.NewIntegration("test", "sample")
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Namespace: integration.Namespace, Name: "sample-revision",
		Labels: map[string]string{v1.IntegrationLabel: integration.Name, "example.org/monitor": "custom-app"},
	}}
	// The official monitor operator keeps source annotations, without copying labels.
	monitor := monitorv1alpha1.NewCamelMonitor("test", "custom-app-sample-revision")
	monitor.Annotations = map[string]string{
		monitorv1alpha1.MonitorImportedKindLabel: "Deployment",
		monitorv1alpha1.MonitorImportedNameLabel: deployment.Name,
	}
	c, err := internal.NewFakeClient(deployment)
	require.NoError(t, err)
	require.NoError(t, c.Create(context.Background(), &monitor))

	require.NoError(t, updateCamelMonitorCondition(context.Background(), c, &integration))
	condition := integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable)
	require.NotNil(t, condition)
	require.Equal(t, corev1.ConditionTrue, condition.Status)
}

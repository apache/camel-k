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
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util/log"
	monitorv1alpha1 "github.com/camel-tooling/camel-monitor-operator/pkg/apis/camel/v1alpha1"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	authclient "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func monitorOwner(apiVersion, kind, name, uid string) metav1.OwnerReference {
	return metav1.OwnerReference{APIVersion: apiVersion, Kind: kind, Name: name, UID: types.UID(uid)}
}

func monitorResource(name, namespace, uid string, owners ...metav1.OwnerReference) *monitorv1alpha1.CamelMonitor {
	resource := monitorv1alpha1.NewCamelMonitor(namespace, name)
	resource.UID = types.UID(uid)
	resource.OwnerReferences = owners
	return &resource
}

func monitorWorkload(gvk schema.GroupVersionKind, name, namespace, uid string, owners ...metav1.OwnerReference) ctrl.Object {
	metadata := metav1.ObjectMeta{Name: name, Namespace: namespace, UID: types.UID(uid), OwnerReferences: owners}
	if gvk.Kind == "CronJob" {
		return &batchv1.CronJob{ObjectMeta: metadata}
	}
	return &appsv1.Deployment{ObjectMeta: metadata}
}

func monitorScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, monitorv1alpha1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))
	return scheme
}

func TestCamelMonitorAssociation(t *testing.T) {
	integration := v1.NewIntegration("test", "sample")
	it := &integration
	it.UID = "integration-uid"
	integrationOwner := monitorOwner("camel.apache.org/v1", "Integration", it.Name, string(it.UID))
	cases := []struct {
		name     string
		monitor  *monitorv1alpha1.CamelMonitor
		workload ctrl.Object
		want     bool
	}{
		{name: "direct", monitor: monitorResource(it.Name, it.Namespace, "monitor", integrationOwner), want: true},
		{name: "direct after missing workload", monitor: monitorResource(it.Name, it.Namespace, "monitor", monitorOwner("apps/v1", "Deployment", it.Name, "missing"), integrationOwner), want: true},
		{name: "no owner", monitor: monitorResource(it.Name, it.Namespace, "monitor")},
		{name: "different namespace", monitor: monitorResource(it.Name, "other", "monitor", integrationOwner)},
		{name: "different name", monitor: monitorResource("other", it.Namespace, "monitor", integrationOwner)},
		{name: "stale Integration UID", monitor: monitorResource(it.Name, it.Namespace, "monitor", monitorOwner("camel.apache.org/v1", "Integration", it.Name, "old"))},
		{name: "foreign group", monitor: monitorResource(it.Name, it.Namespace, "monitor", monitorOwner("other/v1", "Integration", it.Name, string(it.UID)))},
		{name: "foreign kind", monitor: monitorResource(it.Name, it.Namespace, "monitor", monitorOwner("camel.apache.org/v1", "Pipe", it.Name, string(it.UID)))},
		{name: "foreign owner name", monitor: monitorResource(it.Name, it.Namespace, "monitor", monitorOwner("camel.apache.org/v1", "Integration", "other", string(it.UID)))},
	}
	for _, workloadGVK := range []schema.GroupVersionKind{{Group: "apps", Version: "v1", Kind: "Deployment"}, {Group: "batch", Version: "v1", Kind: "CronJob"}} {
		workload := monitorWorkload(workloadGVK, it.Name, it.Namespace, "workload", integrationOwner)
		monitor := monitorResource(it.Name, it.Namespace, "monitor", monitorOwner(workloadGVK.GroupVersion().String(), workloadGVK.Kind, it.Name, "workload"))
		cases = append(cases, struct {
			name     string
			monitor  *monitorv1alpha1.CamelMonitor
			workload ctrl.Object
			want     bool
		}{workloadGVK.Kind, monitor, workload, true})
		stale := monitor.DeepCopy()
		stale.SetOwnerReferences([]metav1.OwnerReference{monitorOwner(workloadGVK.GroupVersion().String(), workloadGVK.Kind, it.Name, "stale")})
		cases = append(cases, struct {
			name     string
			monitor  *monitorv1alpha1.CamelMonitor
			workload ctrl.Object
			want     bool
		}{workloadGVK.Kind + " stale UID", stale, workload, false})
		foreign := monitorWorkload(workloadGVK, it.Name, it.Namespace, "workload")
		foreign.SetOwnerReferences(nil)
		cases = append(cases, struct {
			name     string
			monitor  *monitorv1alpha1.CamelMonitor
			workload ctrl.Object
			want     bool
		}{workloadGVK.Kind + " foreign workload", monitor, foreign, false})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().WithScheme(monitorScheme(t))
			if tc.workload != nil {
				builder.WithObjects(tc.workload)
			}
			got, err := camelMonitorOwnedByIntegration(context.Background(), builder.Build(), tc.monitor, it)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

type monitorErrorReader struct {
	ctrl.Reader
	err error
}

func (c monitorErrorReader) Get(context.Context, ctrl.ObjectKey, ctrl.Object, ...ctrl.GetOption) error {
	return c.err
}

func TestCamelMonitorLookupErrors(t *testing.T) {
	cases := []struct {
		name  string
		err   error
		want  corev1.ConditionStatus
		retry bool
	}{
		{"missing", k8serrors.NewNotFound(schema.GroupResource{Group: "camel.apache.org", Resource: "camelmonitors"}, "sample"), corev1.ConditionStatus(""), false},
		{"absent CRD", &meta.NoKindMatchError{GroupKind: camelMonitorGVK.GroupKind()}, corev1.ConditionStatus(""), false},
		{"forbidden", k8serrors.NewForbidden(schema.GroupResource{Group: "camel.apache.org", Resource: "camelmonitors"}, "sample", errors.New("denied")), corev1.ConditionUnknown, true},
		{"transient", errors.New("connection reset"), corev1.ConditionUnknown, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			integration := v1.NewIntegration("test", "sample")
			it := &integration
			it.Status.Phase = v1.IntegrationPhaseRunning
			it.Status.SetCondition(v1.IntegrationConditionReady, corev1.ConditionTrue, "Ready", "ready")
			ready := *it.Status.GetCondition(v1.IntegrationConditionReady)
			err := updateCamelMonitorCondition(context.Background(), monitorErrorReader{err: tc.err}, it)
			if tc.retry {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
			condition := it.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable)
			if tc.want == "" {
				require.Nil(t, condition)
			} else {
				require.Equal(t, tc.want, condition.Status)
			}
			require.Equal(t, ready, *it.Status.GetCondition(v1.IntegrationConditionReady))
			require.Equal(t, v1.IntegrationPhaseRunning, it.Status.Phase)
		})
	}
}

type monitorActionClient struct {
	client.Client
	monitorReader ctrl.Reader
}

func (c monitorActionClient) Get(ctx context.Context, key ctrl.ObjectKey, obj ctrl.Object, opts ...ctrl.GetOption) error {
	if _, ok := obj.(*monitorv1alpha1.CamelMonitor); ok {
		return c.monitorReader.Get(ctx, key, obj, opts...)
	}
	return c.Client.Get(ctx, key, obj, opts...)
}

func TestMonitorActionCamelMonitorLifecycle(t *testing.T) {
	ctx := context.Background()
	c, it, err := nominalEnvironment()
	require.NoError(t, err)
	it.UID = "integration"
	backing := fake.NewClientBuilder().WithScheme(monitorScheme(t)).Build()
	action := monitorAction{}
	action.InjectLogger(log.Log)
	action.InjectClient(monitorActionClient{Client: c, monitorReader: backing})
	result, err := action.Handle(ctx, it)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Nil(t, result.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	ready := *it.Status.GetCondition(v1.IntegrationConditionReady)
	phase := it.Status.Phase
	monitor := monitorResource(it.Name, it.Namespace, "monitor", monitorOwner("camel.apache.org/v1", "Integration", it.Name, string(it.UID)))
	require.NoError(t, backing.Create(ctx, monitor))
	result, err = action.Handle(ctx, it)
	require.NoError(t, err)
	condition := *result.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable)
	require.Equal(t, corev1.ConditionTrue, condition.Status)
	result, err = action.Handle(ctx, it)
	require.NoError(t, err)
	require.Equal(t, condition, *result.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	monitor.SetOwnerReferences(nil)
	require.NoError(t, backing.Update(ctx, monitor))
	result, err = action.Handle(ctx, it)
	require.NoError(t, err)
	require.Nil(t, result.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	monitor.SetOwnerReferences([]metav1.OwnerReference{monitorOwner("camel.apache.org/v1", "Integration", it.Name, string(it.UID))})
	require.NoError(t, backing.Update(ctx, monitor))
	result, err = action.Handle(ctx, it)
	require.NoError(t, err)
	require.Equal(t, corev1.ConditionTrue, result.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable).Status)
	require.NoError(t, backing.Delete(ctx, monitor))
	result, err = action.Handle(ctx, it)
	require.NoError(t, err)
	require.Nil(t, result.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	require.Equal(t, ready, *result.Status.GetCondition(v1.IntegrationConditionReady))
	require.Equal(t, phase, result.Status.Phase)
	for _, lookupErr := range []error{errors.New("connection reset"), k8serrors.NewForbidden(schema.GroupResource{Group: "camel.apache.org", Resource: "camelmonitors"}, it.Name, errors.New("denied"))} {
		action.InjectClient(monitorActionClient{Client: c, monitorReader: monitorErrorReader{err: lookupErr}})
		it.Status.Phase = v1.IntegrationPhaseDeploying
		it.Status.SetCondition(v1.IntegrationConditionReady, corev1.ConditionFalse, "Pending", "waiting")
		result, err = action.Handle(ctx, it)
		require.ErrorIs(t, err, lookupErr)
		require.NotNil(t, result)
		require.Equal(t, corev1.ConditionUnknown, result.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable).Status)
		require.Equal(t, ready.Status, result.Status.GetCondition(v1.IntegrationConditionReady).Status)
		require.Equal(t, ready.Reason, result.Status.GetCondition(v1.IntegrationConditionReady).Reason)
		require.Equal(t, phase, result.Status.Phase)
	}
	action.InjectClient(monitorActionClient{Client: c, monitorReader: backing})
	result, err = action.Handle(ctx, it)
	require.NoError(t, err)
	require.Nil(t, result.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
}

func TestMonitorActionCamelMonitorEarlyExit(t *testing.T) {
	ctx := context.Background()
	c, it, err := nominalEnvironment()
	require.NoError(t, err)
	it.UID = "integration"
	it.Status.Phase = v1.IntegrationPhaseError
	it.Status.SetCondition(v1.IntegrationConditionReady, corev1.ConditionFalse, v1.IntegrationConditionInitializationFailedReason, "failed")
	ready := *it.Status.GetCondition(v1.IntegrationConditionReady)
	backing := fake.NewClientBuilder().WithScheme(monitorScheme(t)).Build()
	action := monitorAction{}
	action.InjectLogger(log.Log)
	action.InjectClient(monitorActionClient{Client: c, monitorReader: backing})
	result, err := action.Handle(ctx, it)
	require.NoError(t, err)
	require.Nil(t, result, "unchanged early exit preserves nil result")
	monitor := monitorResource(it.Name, it.Namespace, "monitor", monitorOwner("camel.apache.org/v1", "Integration", it.Name, string(it.UID)))
	require.NoError(t, backing.Create(ctx, monitor))
	result, err = action.Handle(ctx, it)
	require.NoError(t, err)
	require.Same(t, it, result)
	require.Equal(t, corev1.ConditionTrue, result.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable).Status)
	result, err = action.Handle(ctx, it)
	require.NoError(t, err)
	require.Nil(t, result)
	require.NoError(t, backing.Delete(ctx, monitor))
	result, err = action.Handle(ctx, it)
	require.NoError(t, err)
	require.Same(t, it, result)
	require.Nil(t, result.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable))
	require.Equal(t, ready, *it.Status.GetCondition(v1.IntegrationConditionReady))
	require.Equal(t, v1.IntegrationPhaseError, it.Status.Phase)
}

func TestCamelMonitorEvents(t *testing.T) {
	ctx := context.Background()
	monitor := monitorResource("sample", "test", "uid", monitorOwner("apps/v1", "Deployment", "sample", "workload"))
	changed := monitor.DeepCopy()
	changed.SetOwnerReferences(nil)
	predicate := camelMonitorOwnerChangedPredicate()
	require.True(t, predicate.Create(event.CreateEvent{Object: monitor}))
	require.True(t, predicate.Delete(event.DeleteEvent{Object: monitor}))
	require.True(t, predicate.Update(event.UpdateEvent{ObjectOld: monitor, ObjectNew: changed}))
	statusChanged := monitor.DeepCopy()
	statusChanged.Status.Phase = monitorv1alpha1.CamelMonitorPhaseRunning
	require.False(t, predicate.Update(event.UpdateEvent{ObjectOld: monitor, ObjectNew: statusChanged}))
	h := handler.EnqueueRequestsFromMapFunc(camelMonitorEnqueueRequests)
	for _, action := range []string{"create", "delete", "owner change"} {
		t.Run(action, func(t *testing.T) {
			queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
			defer queue.ShutDown()
			switch action {
			case "create":
				h.Create(ctx, event.CreateEvent{Object: monitor}, queue)
			case "delete":
				h.Delete(ctx, event.DeleteEvent{Object: monitor}, queue)
			case "owner change":
				h.Update(ctx, event.UpdateEvent{ObjectOld: monitor, ObjectNew: changed}, queue)
			}
			require.Equal(t, 1, queue.Len())
			request, shutdown := queue.Get()
			require.False(t, shutdown)
			require.Equal(t, ctrl.ObjectKeyFromObject(monitor), request.NamespacedName)
			queue.Done(request)
		})
	}
	// A workload owner change must pass the watch predicate even without a status
	// change, including when its old Integration reference is not a controller.
	oldWorkload := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "sample", Namespace: "test", OwnerReferences: []metav1.OwnerReference{monitorOwner("camel.apache.org/v1", "Integration", "sample", "integration")}}}
	newWorkload := oldWorkload.DeepCopy()
	newWorkload.SetOwnerReferences(nil)
	require.True(t, camelMonitorOwnerChangedPredicate().Update(event.UpdateEvent{ObjectOld: oldWorkload, ObjectNew: newWorkload}))
	require.False(t, camelMonitorOwnerChangedPredicate().Update(event.UpdateEvent{ObjectOld: oldWorkload, ObjectNew: oldWorkload.DeepCopy()}))
}

func TestCamelMonitorWatchPermissionsNamespaces(t *testing.T) {
	for _, watch := range []string{" a, b,a ", ""} {
		t.Run(watch, func(t *testing.T) {
			t.Setenv("WATCH_NAMESPACE", watch)
			t.Setenv("NAMESPACE", "operator")
			c := k8sfake.NewClientset()
			observed := map[string][]string{}
			c.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
				create, ok := action.(k8stesting.CreateAction)
				require.True(t, ok)
				review, ok := create.GetObject().(*authorizationv1.SelfSubjectAccessReview)
				require.True(t, ok)
				attributes := review.Spec.ResourceAttributes
				require.Equal(t, "camel.apache.org", attributes.Group)
				require.Equal(t, "camelmonitors", attributes.Resource)
				require.NotContains(t, attributes.Namespace, ",")
				observed[attributes.Namespace] = append(observed[attributes.Namespace], attributes.Verb)
				review.Status.Allowed = attributes.Namespace == "a" || attributes.Namespace == "b" || attributes.Namespace == "operator" || attributes.Namespace == ""
				return true, review, nil
			})
			allowed, err := camelMonitorWatchPermissions(context.Background(), c)
			require.NoError(t, err)
			require.True(t, allowed)
			if watch == "" {
				require.Equal(t, map[string][]string{"": {"list", "watch"}}, observed)
			} else {
				require.Equal(t, map[string][]string{"operator": {"list", "watch"}, "a": {"list", "watch"}, "b": {"list", "watch"}}, observed)
			}
		})
	}
}

func TestCamelMonitorWorkloadOwnerEvents(t *testing.T) {
	ctx := context.Background()
	h := handler.EnqueueRequestsFromMapFunc(camelMonitorEnqueueRequests)
	for _, controller := range []bool{false, true} {
		for _, kind := range []string{"Deployment", "CronJob"} {
			t.Run(kind+" controller="+strconv.FormatBool(controller), func(t *testing.T) {
				group := "apps"
				if kind == "CronJob" {
					group = "batch"
				}
				owner := monitorOwner("camel.apache.org/v1", "Integration", "sample", "integration")
				owner.Controller = &controller
				old := monitorWorkload(schema.GroupVersionKind{Group: group, Version: "v1", Kind: kind}, "sample", "test", "workload", owner)
				changed := monitorWorkload(schema.GroupVersionKind{Group: group, Version: "v1", Kind: kind}, "sample", "test", "workload", owner)
				changed.SetOwnerReferences(nil)
				for _, deleted := range []bool{false, true} {
					queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
					if deleted {
						h.Delete(ctx, event.DeleteEvent{Object: old}, queue)
					} else {
						h.Update(ctx, event.UpdateEvent{ObjectOld: old, ObjectNew: changed}, queue)
					}
					require.Equal(t, 1, queue.Len())
					request, shutdown := queue.Get()
					require.False(t, shutdown)
					require.Equal(t, ctrl.ObjectKey{Namespace: "test", Name: "sample"}, request.NamespacedName)
					queue.Done(request)
					queue.ShutDown()
				}
			})
		}
	}
}

type monitorWorkloadErrorReader struct {
	ctrl.Reader
	err error
}

func (c monitorWorkloadErrorReader) Get(ctx context.Context, key ctrl.ObjectKey, obj ctrl.Object, opts ...ctrl.GetOption) error {
	if obj.GetObjectKind().GroupVersionKind().Kind == "Deployment" {
		return c.err
	}
	return c.Reader.Get(ctx, key, obj, opts...)
}

func TestCamelMonitorMultipleWorkloadOwners(t *testing.T) {
	integration := v1.NewIntegration("test", "sample")
	it := &integration
	it.UID = "integration"
	monitor := monitorResource(it.Name, it.Namespace, "monitor",
		monitorOwner("apps/v1", "Deployment", it.Name, "deployment"),
		monitorOwner("batch/v1", "CronJob", it.Name, "cron"))
	cron := monitorWorkload(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "CronJob"}, it.Name, it.Namespace, "cron",
		monitorOwner("camel.apache.org/v1", "Integration", it.Name, string(it.UID)))
	backing := fake.NewClientBuilder().WithScheme(monitorScheme(t)).WithObjects(cron).Build()
	lookupErr := errors.New("temporary Deployment lookup failure")
	reader := monitorWorkloadErrorReader{Reader: backing, err: lookupErr}
	available, err := camelMonitorOwnedByIntegration(context.Background(), reader, monitor, it)
	require.NoError(t, err)
	require.True(t, available, "a failed candidate must not hide a valid ownership chain")
	cron.SetOwnerReferences(nil)
	require.NoError(t, backing.Update(context.Background(), cron))
	available, err = camelMonitorOwnedByIntegration(context.Background(), reader, monitor, it)
	require.ErrorIs(t, err, lookupErr)
	require.False(t, available, "retain lookup error when no candidate establishes ownership")
}

type monitorWatchClient struct {
	client.Client
	cluster *k8sfake.Clientset
}

func (c monitorWatchClient) Discovery() discovery.DiscoveryInterfaces { return c.cluster.Discovery() }
func (c monitorWatchClient) AuthorizationV1() authclient.AuthorizationV1Interface {
	return c.cluster.AuthorizationV1()
}

func TestCamelMonitorOptionalWatch(t *testing.T) {
	t.Setenv("WATCH_NAMESPACE", "test")
	t.Setenv("NAMESPACE", "test")
	for _, installed := range []bool{false, true} {
		t.Run(strconv.FormatBool(installed), func(t *testing.T) {
			c := k8sfake.NewClientset()
			c.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
				create, ok := action.(k8stesting.CreateAction)
				require.True(t, ok)
				review, ok := create.GetObject().(*authorizationv1.SelfSubjectAccessReview)
				require.True(t, ok)
				review.Status.Allowed = false
				return true, review, nil
			})
			if installed {
				discovery, ok := c.Discovery().(*fakediscovery.FakeDiscovery)
				require.True(t, ok)
				discovery.Resources = []*metav1.APIResourceList{{GroupVersion: camelMonitorGVK.GroupVersion().String(), APIResources: []metav1.APIResource{{Kind: camelMonitorGVK.Kind, Name: "camelmonitors", Namespaced: true}}}}
			}
			// With no CRD, or without permission to watch an installed CRD, startup
			// must not register a source which would prevent the cache from syncing.
			require.False(t, camelMonitorWatchAvailable(context.Background(), monitorWatchClient{cluster: c}))
			allowed, err := camelMonitorWatchPermissions(context.Background(), c)
			require.NoError(t, err)
			require.False(t, allowed)
		})
	}
}

type monitorRoundTripper func(*http.Request) (*http.Response, error)

func (f monitorRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestCamelMonitorAssociationIncludesWorkloadsOutsideTheLabelFilteredCache(t *testing.T) {
	integration := v1.NewIntegration("test", "sample")
	integration.UID = "integration"
	monitor := monitorResource("sample", "test", "monitor", monitorOwner("apps/v1", "Deployment", "sample", "deployment"))
	scheme := monitorScheme(t)
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{appsv1.SchemeGroupVersion})
	mapper.Add(appsv1.SchemeGroupVersion.WithKind("Deployment"), meta.RESTScopeNamespace)
	// An empty cache represents a Deployment excluded by the operator's Integration-label selector.
	cached := fake.NewClientBuilder().WithScheme(scheme).Build()
	requests := 0
	c, err := ctrl.New(&rest.Config{Host: "http://api.invalid"}, ctrl.Options{
		Scheme: scheme,
		Mapper: mapper,
		Cache:  &ctrl.CacheOptions{Reader: cached},
		HTTPClient: &http.Client{Transport: monitorRoundTripper(func(request *http.Request) (*http.Response, error) {
			requests++
			require.Equal(t, http.MethodGet, request.Method)
			require.Equal(t, "/apis/apps/v1/namespaces/test/deployments/sample", request.URL.Path)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": {"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"sample","namespace":"test","uid":"deployment","ownerReferences":[{"apiVersion":"camel.apache.org/v1","kind":"Integration","name":"sample","uid":"integration"}]}}`)),
			}, nil
		})},
	})
	require.NoError(t, err)
	associated, err := camelMonitorOwnedByIntegration(context.Background(), c, monitor, &integration)
	require.NoError(t, err)
	require.True(t, associated)
	require.Equal(t, 1, requests, "ownership lookup must read metadata even when the workload is outside the cache")
}

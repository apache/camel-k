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
	"fmt"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sclient "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

var camelMonitorGVK = schema.GroupVersionKind{Group: "camel.apache.org", Version: "v1alpha1", Kind: "CamelMonitor"}

const (
	camelMonitorDiscoveryTimeout = 10 * time.Second
)

// updateCamelMonitorCondition reports resource discovery, independently of runtime readiness.
// Lookup failures use the existing action error/retry path.
func updateCamelMonitorCondition(ctx context.Context, c ctrl.Reader, integration *v1.Integration) error {
	monitor, err := kubernetes.GetUnstructured(ctx, c, camelMonitorGVK, integration.Name, integration.Namespace)
	available := false
	if err == nil {
		available, err = camelMonitorOwnedByIntegration(ctx, c, monitor, integration)
	}
	switch {
	case k8serrors.IsNotFound(err), meta.IsNoMatchError(err):
		integration.Status.RemoveCondition(v1.IntegrationConditionCamelMonitorAvailable)
	case err != nil:
		integration.Status.SetCondition(v1.IntegrationConditionCamelMonitorAvailable, corev1.ConditionUnknown,
			"CamelMonitorLookupFailed", err.Error())

		return err
	case available:
		integration.Status.SetCondition(v1.IntegrationConditionCamelMonitorAvailable, corev1.ConditionTrue,
			"CamelMonitorAvailable", "An associated CamelMonitor resource is available")
	default:
		integration.Status.RemoveCondition(v1.IntegrationConditionCamelMonitorAvailable)
	}

	return nil
}

// A monitor can belong directly to the Integration or to its Deployment/CronJob.
// Only these two hops are followed, in the Integration namespace, with UID checks
// at both edges. Names and labels alone do not establish ownership.
func camelMonitorOwnedByIntegration(ctx context.Context, c ctrl.Reader, monitor *unstructured.Unstructured, integration *v1.Integration) (bool, error) {
	if monitor.GetNamespace() != integration.Namespace || monitor.GetName() != integration.Name {
		return false, nil
	}
	for _, owner := range monitor.GetOwnerReferences() {
		if ownerMatchesIntegration(owner, integration) {
			return true, nil
		}
	}
	var lookupErr error
	for _, owner := range monitor.GetOwnerReferences() {
		gv, err := schema.ParseGroupVersion(owner.APIVersion)
		if err != nil || owner.UID == "" || owner.Name != integration.Name {
			continue
		}
		supportedWorkload := (gv.Group == "apps" && owner.Kind == "Deployment") || (gv.Group == "batch" && owner.Kind == "CronJob")
		if !supportedWorkload {
			continue
		}
		workload, err := kubernetes.GetUnstructured(ctx, c, gv.WithKind(owner.Kind), owner.Name, integration.Namespace)
		if err != nil {
			if !k8serrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
				lookupErr = err
			}

			continue
		}
		if workload.GetUID() != owner.UID {
			continue
		}
		for _, reference := range workload.GetOwnerReferences() {
			if ownerMatchesIntegration(reference, integration) {
				return true, nil
			}
		}
	}

	return false, lookupErr
}

func ownerMatchesIntegration(owner metav1.OwnerReference, integration *v1.Integration) bool {
	gv, err := schema.ParseGroupVersion(owner.APIVersion)

	return err == nil && gv.Group == v1.SchemeGroupVersion.Group && owner.Kind == v1.IntegrationKind &&
		owner.Name == integration.Name && owner.UID != "" && owner.UID == integration.UID
}

// Enqueue the same-name Integration even if the new owner no longer matches:
// deletes and ownership changes must also clear a previously true condition.
// Association is checked when reconciling, never inferred from the name alone.
func camelMonitorEnqueueRequests(_ context.Context, object ctrl.Object) []reconcile.Request {
	return []reconcile.Request{{NamespacedName: ctrl.ObjectKeyFromObject(object)}}
}

func watchCamelMonitorResources(ctx context.Context, c client.Client, b *builder.Builder) {
	if !camelMonitorWatchAvailable(ctx, c) {
		return
	}
	monitor := &unstructured.Unstructured{}
	monitor.SetGroupVersionKind(camelMonitorGVK)
	b.Watches(monitor, handler.EnqueueRequestsFromMapFunc(camelMonitorEnqueueRequests))
}

func camelMonitorWatchAvailable(ctx context.Context, c client.Client) bool {
	installed, err := kubernetes.IsAPIResourceInstalled(c, camelMonitorGVK.GroupVersion().String(), camelMonitorGVK.Kind)
	if err != nil || !installed {
		Log.Info(fmt.Sprintf("CamelMonitor resources are unavailable (%v). Install the CRD and grant get/list/watch permissions, then restart the operator to enable discovery events.", err))

		return false
	}
	ctx, cancel := context.WithTimeout(ctx, camelMonitorDiscoveryTimeout)
	defer cancel()
	allowed, permissionErr := camelMonitorWatchPermissions(ctx, c)
	if permissionErr != nil || !allowed {
		Log.Info("CamelMonitor watch permissions are unavailable. Grant list/watch permissions and restart the operator to enable discovery events.")

		return false
	}

	return true
}

// Workload ownership changes can invalidate a transitive association without a
// status change. A same-name mapper also handles non-controller references.
func camelMonitorWorkloadPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld != nil && e.ObjectNew != nil &&
				!equality.Semantic.DeepEqual(e.ObjectOld.GetOwnerReferences(), e.ObjectNew.GetOwnerReferences())
		},
	}
}

// Match the cache scope configured by operator.getNamespacesSelector: a local
// cache includes the operator namespace as well as every watched namespace.
func camelMonitorWatchPermissions(ctx context.Context, c k8sclient.Interface) (bool, error) {
	namespaces := []string{""}
	if !platform.IsCurrentOperatorGlobal() {
		namespaces = []string{platform.GetOperatorNamespace()}
		for namespace := range strings.SplitSeq(platform.GetOperatorWatchNamespace(), ",") {
			namespace = strings.TrimSpace(namespace)
			if namespace != "" && !slices.Contains(namespaces, namespace) {
				namespaces = append(namespaces, namespace)
			}
		}
	}
	for _, namespace := range namespaces {
		for _, verb := range []string{"list", "watch"} {
			allowed, err := kubernetes.CheckSelfPermission(ctx, c, camelMonitorGVK.Group, "camelmonitors", namespace, "", verb)
			if err != nil || !allowed {
				return false, err
			}
		}
	}

	return true, nil
}

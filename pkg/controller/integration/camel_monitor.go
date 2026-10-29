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
	"os"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	monitorv1alpha1 "github.com/camel-tooling/camel-monitor-operator/pkg/apis/camel/v1alpha1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	servingv1 "github.com/apache/camel-k/v2/pkg/apis/duck/knative/serving/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
)

// updateCamelMonitorCondition reports resource discovery, independently of runtime readiness.
// The monitor action checks that the optional API is installed before calling it.
func updateCamelMonitorCondition(ctx context.Context, c ctrl.Reader, integration *v1.Integration) error {
	monitors := &monitorv1alpha1.CamelMonitorList{}
	err := c.List(ctx, monitors, ctrl.InNamespace(integration.Namespace))
	available := false
	if err == nil {
		for i := range monitors.Items {
			name, lookupErr := camelMonitorIntegrationName(ctx, c, &monitors.Items[i])
			if lookupErr != nil {
				err = lookupErr
			}
			if name == integration.Name {
				available = true

				break
			}
		}
	}
	switch {
	case available:
		integration.Status.SetCondition(v1.IntegrationConditionCamelMonitorAvailable, corev1.ConditionTrue,
			"CamelMonitorAvailable", "An associated CamelMonitor resource is available")
	case k8serrors.IsNotFound(err), meta.IsNoMatchError(err):
		integration.Status.RemoveCondition(v1.IntegrationConditionCamelMonitorAvailable)
	case err != nil:
		integration.Status.SetCondition(v1.IntegrationConditionCamelMonitorAvailable, corev1.ConditionUnknown,
			"CamelMonitorLookupFailed", err.Error())

		return err
	default:
		integration.Status.RemoveCondition(v1.IntegrationConditionCamelMonitorAvailable)
	}

	return nil
}

// Synthetic monitors in the official API retain their source as annotations rather
// than copying its discovery label. Resolve that source in the same namespace and
// use its existing Integration label; monitor names need not equal Integration names.
func camelMonitorIntegrationName(ctx context.Context, c ctrl.Reader, monitor *monitorv1alpha1.CamelMonitor) (string, error) {
	label := os.Getenv(kubernetes.CamelMonitorOperatorLabelEnvVar)
	if label == "" {
		return "", nil
	}
	if name := monitor.Labels[label]; name != "" {
		return name, nil
	}
	workload, err := camelMonitorWorkload(ctx, c, monitor)
	if err != nil || workload == nil {
		return "", err
	}
	if workload.GetLabels()[label] == "" {
		return "", nil
	}

	return workload.GetLabels()[v1.IntegrationLabel], nil
}

func camelMonitorWorkload(ctx context.Context, c ctrl.Reader, monitor *monitorv1alpha1.CamelMonitor) (ctrl.Object, error) {
	name := monitor.Annotations[monitorv1alpha1.MonitorImportedNameLabel]
	if name == "" {
		return nil, nil
	}
	var workload ctrl.Object
	switch monitor.Annotations[monitorv1alpha1.MonitorImportedKindLabel] {
	case "Deployment":
		workload = &appsv1.Deployment{}
	case "CronJob":
		workload = &batchv1.CronJob{}
	case "Service":
		workload = &servingv1.Service{}
	default:
		return nil, nil
	}
	if err := c.Get(ctx, ctrl.ObjectKey{Namespace: monitor.Namespace, Name: name}, workload); err != nil {
		if k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			return nil, nil
		}

		return nil, err
	}

	return workload, nil
}

func camelMonitorEnqueueRequests(ctx context.Context, c client.Client, object ctrl.Object) []reconcile.Request {
	monitor, ok := object.(*monitorv1alpha1.CamelMonitor)
	if !ok {
		return nil
	}
	name, err := camelMonitorIntegrationName(ctx, c, monitor)
	if err != nil {
		log.Error(err, "Failed to retrieve the monitored workload")

		return nil
	}
	if name == "" {
		return nil
	}
	integrations := &v1.IntegrationList{}
	if err := c.List(ctx, integrations, ctrl.InNamespace(object.GetNamespace())); err != nil {
		log.Error(err, "Failed to list integrations")

		return nil
	}
	for i := range integrations.Items {
		integration := &integrations.Items[i]
		if integration.Name == name && (&monitorAction{}).CanHandle(integration) &&
			platform.IsOperatorHandlerConsideringLock(ctx, c, integration.Namespace, integration) {
			return []reconcile.Request{{NamespacedName: ctrl.ObjectKeyFromObject(integration)}}
		}
	}

	return nil
}

func camelMonitorEventHandler(c client.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object ctrl.Object) []reconcile.Request {
		return camelMonitorEnqueueRequests(ctx, c, object)
	})
}

// Creation, deletion and association changes affect availability, unlike runtime status updates.
// The map handler processes both the old and new objects on updates.
func camelMonitorAssociationChangedPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}
			label := os.Getenv(kubernetes.CamelMonitorOperatorLabelEnvVar)

			return e.ObjectOld.GetLabels()[label] != e.ObjectNew.GetLabels()[label] ||
				e.ObjectOld.GetAnnotations()[monitorv1alpha1.MonitorImportedNameLabel] != e.ObjectNew.GetAnnotations()[monitorv1alpha1.MonitorImportedNameLabel] ||
				e.ObjectOld.GetAnnotations()[monitorv1alpha1.MonitorImportedKindLabel] != e.ObjectNew.GetAnnotations()[monitorv1alpha1.MonitorImportedKindLabel]
		},
	}
}

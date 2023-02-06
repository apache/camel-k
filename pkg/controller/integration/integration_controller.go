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
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	camelevent "github.com/apache/camel-k/pkg/event"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/monitoring"
)

func Add(mgr manager.Manager, c client.Client) error {
	return add(mgr, c, newReconciler(mgr, c))
}

func newReconciler(mgr manager.Manager, c client.Client) reconcile.Reconciler {
	return monitoring.NewInstrumentedReconciler(
		&reconcileIntegration{
			client:   c,
			scheme:   mgr.GetScheme(),
			recorder: mgr.GetEventRecorderFor("camel-k-integration-controller"),
		},
		schema.GroupVersionKind{
			Group:   v1.SchemeGroupVersion.Group,
			Version: v1.SchemeGroupVersion.Version,
			Kind:    v1.IntegrationKind,
		},
	)
}

func integrationUpdateFunc(old *v1.Integration, it *v1.Integration) bool {
	// Observe the time to first readiness metric
	previous := old.Status.GetCondition(v1.IntegrationConditionReady)
	next := it.Status.GetCondition(v1.IntegrationConditionReady)
	if isIntegrationUpdated(it, previous, next) {
		duration := next.FirstTruthyTime.Time.Sub(it.Status.InitializationTimestamp.Time)
		Log.WithValues("request-namespace", it.Namespace, "request-name", it.Name, "ready-after", duration.Seconds()).
			ForIntegration(it).Infof("First readiness after %s", duration)
		timeToFirstReadiness.Observe(duration.Seconds())
	}

	// If traits have changed, the reconciliation loop must kick in as
	// traits may have impact
	sameTraits, err := trait.IntegrationsHaveSameTraits(old, it)
	if err != nil {
		Log.ForIntegration(it).Error(
			err,
			"unable to determine if old and new resource have the same traits")
	}
	if !sameTraits {
		return true
	}

	// Ignore updates to the integration status in which case metadata.Generation does not change,
	// or except when the integration phase changes as it's used to transition from one phase
	// to another.
	return old.Generation != it.Generation ||
		old.Status.Phase != it.Status.Phase
}

func isIntegrationUpdated(it *v1.Integration, previous, next *v1.IntegrationCondition) bool {
	if previous == nil || previous.Status != corev1.ConditionTrue && (previous.FirstTruthyTime == nil || previous.FirstTruthyTime.IsZero()) {
		if next != nil && next.Status == corev1.ConditionTrue && next.FirstTruthyTime != nil && !next.FirstTruthyTime.IsZero() {
			return it.Status.InitializationTimestamp != nil
		}
	}

	return false
}

func integrationKitEnqueueRequestsFromMapFunc(c client.Client, kit *v1.IntegrationKit) []reconcile.Request {
	var requests []reconcile.Request
	if kit.Status.Phase != v1.IntegrationKitPhaseReady && kit.Status.Phase != v1.IntegrationKitPhaseError {
		return requests
	}

	list := &v1.IntegrationList{}
	// Do global search in case of global operator (it may be using a global platform)
	var opts []ctrl.ListOption
	if !platform.IsCurrentOperatorGlobal() {
		opts = append(opts, ctrl.InNamespace(kit.Namespace))
	}
	if err := c.List(context.Background(), list, opts...); err != nil {
		log.Error(err, "Failed to retrieve integration list")
		return requests
	}

	for i := range list.Items {
		integration := &list.Items[i]
		Log.Debug("Integration Controller: Assessing integration", "integration", integration.Name, "namespace", integration.Namespace)

		match, err := sameOrMatch(kit, integration)
		if err != nil {
			Log.ForIntegration(integration).Errorf(err, "Error matching integration %q with kit %q", integration.Name, kit.Name)
			continue
		}
		if !match {
			continue
		}

		if integration.Status.Phase == v1.IntegrationPhaseBuildingKit ||
			integration.Status.Phase == v1.IntegrationPhaseRunning {
			log.Infof("Kit %s ready, notify integration: %s", kit.Name, integration.Name)
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: integration.Namespace,
					Name:      integration.Name,
				},
			})
		}
	}

	return requests
}

func integrationPlatformEnqueueRequestsFromMapFunc(c client.Client, p *v1.IntegrationPlatform) []reconcile.Request {
	var requests []reconcile.Request

	if p.Status.Phase == v1.IntegrationPlatformPhaseReady {
		list := &v1.IntegrationList{}

		// Do global search in case of global operator (it may be using a global platform)
		var opts []ctrl.ListOption
		if !platform.IsCurrentOperatorGlobal() {
			opts = append(opts, ctrl.InNamespace(p.Namespace))
		}

		if err := c.List(context.Background(), list, opts...); err != nil {
			log.Error(err, "Failed to list integrations")
			return requests
		}

		for _, integration := range list.Items {
			if integration.Status.Phase == v1.IntegrationPhaseWaitingForPlatform {
				log.Infof("Platform %s ready, wake-up integration: %s", p.Name, integration.Name)
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: integration.Namespace,
						Name:      integration.Name,
					},
				})
			}
		}
	}

	return requests
}

func add(mgr manager.Manager, c client.Client, r reconcile.Reconciler) error {
	b := builder.ControllerManagedBy(mgr).
		Named("integration-controller").
		// Watch for changes to primary resource Integration
		For(&v1.Integration{}, builder.WithPredicates(
			platform.FilteringFuncs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					old, ok := e.ObjectOld.(*v1.Integration)
					if !ok {
						return false
					}
					it, ok := e.ObjectNew.(*v1.Integration)
					if !ok {
						return false
					}

					return integrationUpdateFunc(old, it)
				},
				DeleteFunc: func(e event.DeleteEvent) bool {
					// Evaluates to false if the object has been confirmed deleted
					return !e.DeleteStateUnknown
				},
			})).
		// Watch for IntegrationKit phase transitioning to ready or error, and
		// enqueue requests for any integration that matches the kit, in building
		// or running phase.
		Watches(&source.Kind{Type: &v1.IntegrationKit{}},
			handler.EnqueueRequestsFromMapFunc(func(a ctrl.Object) []reconcile.Request {
				kit, ok := a.(*v1.IntegrationKit)
				if !ok {
					log.Error(fmt.Errorf("type assertion failed: %v", a), "Failed to retrieve integration list")
					return []reconcile.Request{}
				}

				return integrationKitEnqueueRequestsFromMapFunc(c, kit)
			})).
		// Watch for IntegrationPlatform phase transitioning to ready and enqueue
		// requests for any integrations that are in phase waiting for platform
		Watches(&source.Kind{Type: &v1.IntegrationPlatform{}},
			handler.EnqueueRequestsFromMapFunc(func(a ctrl.Object) []reconcile.Request {
				p, ok := a.(*v1.IntegrationPlatform)
				if !ok {
					log.Error(fmt.Errorf("type assertion failed: %v", a), "Failed to list integrations")
					return []reconcile.Request{}
				}

				return integrationPlatformEnqueueRequestsFromMapFunc(c, p)
			})).
		// Watch for the owned Deployments
		Owns(&appsv1.Deployment{}, builder.WithPredicates(StatusChangedPredicate{})).
		// Watch for the Integration Pods
		Watches(&source.Kind{Type: &corev1.Pod{}},
			handler.EnqueueRequestsFromMapFunc(func(a ctrl.Object) []reconcile.Request {
				pod, ok := a.(*corev1.Pod)
				if !ok {
					log.Error(fmt.Errorf("type assertion failed: %v", a), "Failed to list integration pods")
					return []reconcile.Request{}
				}
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Namespace: pod.GetNamespace(),
							Name:      pod.Labels[v1.IntegrationLabel],
						},
					},
				}
			}))

	if ok, err := kubernetes.IsAPIResourceInstalled(c, batchv1.SchemeGroupVersion.String(), reflect.TypeOf(batchv1.CronJob{}).Name()); ok && err == nil {
		// Watch for the owned CronJobs
		b.Owns(&batchv1.CronJob{}, builder.WithPredicates(StatusChangedPredicate{}))
	}

	// Watch for the owned Knative Services conditionally
	if ok, err := kubernetes.IsAPIResourceInstalled(c, servingv1.SchemeGroupVersion.String(), reflect.TypeOf(servingv1.Service{}).Name()); err != nil {
		return err
	} else if ok {
		// Check for permission to watch the ConsoleCLIDownload resource
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if ok, err = kubernetes.CheckPermission(ctx, c, serving.GroupName, "services", platform.GetOperatorWatchNamespace(), "", "watch"); err != nil {
			return err
		} else if ok {
			b.Owns(&servingv1.Service{}, builder.WithPredicates(StatusChangedPredicate{}))
		}
	}

	return b.Complete(r)
}

var _ reconcile.Reconciler = &reconcileIntegration{}

// reconcileIntegration reconciles an Integration object.
type reconcileIntegration struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the API server
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for an Integration object and makes changes based on the state read
// and what is in the Integration.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *reconcileIntegration) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	rlog := Log.WithValues("request-namespace", request.Namespace, "request-name", request.Name)
	rlog.Info("Reconciling Integration")

	// Make sure the operator is allowed to act on namespace
	if ok, err := platform.IsOperatorAllowedOnNamespace(ctx, r.client, request.Namespace); err != nil {
		return reconcile.Result{}, err
	} else if !ok {
		rlog.Info("Ignoring request because namespace is locked")
		return reconcile.Result{}, nil
	}

	// Fetch the Integration instance
	var instance v1.Integration

	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Only process resources assigned to the operator
	if !platform.IsOperatorHandlerConsideringLock(ctx, r.client, request.Namespace, &instance) {
		rlog.Info("Ignoring request because resource is not assigned to current operator")
		return reconcile.Result{}, nil
	}

	target := instance.DeepCopy()
	targetLog := rlog.ForIntegration(target)

	actions := []Action{
		NewPlatformSetupAction(),
		NewInitializeAction(),
		newBuildKitAction(),
		NewMonitorAction(),
	}

	for _, a := range actions {
		a.InjectClient(r.client)
		a.InjectLogger(targetLog)

		if a.CanHandle(target) {
			targetLog.Infof("Invoking action %s", a.Name())

			newTarget, err := a.Handle(ctx, target)
			if err != nil {
				camelevent.NotifyIntegrationError(ctx, r.client, r.recorder, &instance, newTarget, err)
				// Update the integration (mostly just to update its phase) if the new instance is returned
				if newTarget != nil {
					_ = r.update(ctx, &instance, newTarget, &targetLog)
				}
				return reconcile.Result{}, err
			}

			if newTarget != nil {
				if err := r.update(ctx, &instance, newTarget, &targetLog); err != nil {
					camelevent.NotifyIntegrationError(ctx, r.client, r.recorder, &instance, newTarget, err)
					return reconcile.Result{}, err
				}
			}

			// handle one action at time so the resource
			// is always at its latest state
			camelevent.NotifyIntegrationUpdated(ctx, r.client, r.recorder, &instance, newTarget)
			break
		}
	}

	return reconcile.Result{}, nil
}

func (r *reconcileIntegration) update(ctx context.Context, base *v1.Integration, target *v1.Integration, log *log.Logger) error {
	d, err := digest.ComputeForIntegration(target)
	if err != nil {
		return err
	}

	target.Status.Digest = d
	target.Status.ObservedGeneration = base.Generation

	if err := r.client.Status().Patch(ctx, target, ctrl.MergeFrom(base)); err != nil {
		return err
	}

	if target.Status.Phase != base.Status.Phase {
		log.Info(
			"state transition",
			"phase-from", base.Status.Phase,
			"phase-to", target.Status.Phase,
		)
	}

	return nil
}

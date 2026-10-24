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

package integrationkit

import (
	"context"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/events"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	camelevent "github.com/apache/camel-k/v2/pkg/event"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/digest"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/monitoring"
)

const (
	requeueAfterDuration = 2 * time.Second
)

// Add creates a new IntegrationKit Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, mgr manager.Manager, c client.Client) error {
	return add(ctx, mgr, newReconciler(mgr, c))
}

func newReconciler(mgr manager.Manager, c client.Client) reconcile.Reconciler {
	return monitoring.NewInstrumentedReconciler(
		&reconcileIntegrationKit{
			client:   c,
			scheme:   mgr.GetScheme(),
			recorder: mgr.GetEventRecorder("camel-k-integration-kit-controller"),
			actions: []Action{
				NewInitializeAction(),
				NewBuildAction(),
				NewMonitorAction(),
				NewErrorAction(),
			},
		},
		schema.GroupVersionKind{
			Group:   v1.SchemeGroupVersion.Group,
			Version: v1.SchemeGroupVersion.Version,
			Kind:    v1.IntegrationKitKind,
		},
	)
}

func add(_ context.Context, mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("integrationkit-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource IntegrationKit
	err = c.Watch(
		source.Kind(
			mgr.GetCache(),
			&v1.IntegrationKit{},
			&handler.TypedEnqueueRequestForObject[*v1.IntegrationKit]{},
			platform.FilteringFuncs[*v1.IntegrationKit]{
				UpdateFunc: func(e event.TypedUpdateEvent[*v1.IntegrationKit]) bool {
					// Ignore updates to the integration kit status in which case metadata.Generation
					// does not change, or except when the integration kit phase changes as it's used
					// to transition from one phase to another
					return e.ObjectOld.Generation != e.ObjectNew.Generation ||
						e.ObjectOld.Status.Phase != e.ObjectNew.Status.Phase
				},
				DeleteFunc: func(e event.TypedDeleteEvent[*v1.IntegrationKit]) bool {
					// Evaluates to false if the object has been confirmed deleted
					return !e.DeleteStateUnknown
				},
			},
		),
	)
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Builds and requeue the owner IntegrationKit
	err = c.Watch(
		source.Kind(mgr.GetCache(),
			&v1.Build{},
			handler.TypedEnqueueRequestForOwner[*v1.Build](
				mgr.GetScheme(),
				mgr.GetRESTMapper(),
				&v1.IntegrationKit{},
				handler.OnlyControllerOwner(),
			),
			platform.FilteringFuncs[*v1.Build]{
				UpdateFunc: func(e event.TypedUpdateEvent[*v1.Build]) bool {
					// Ignore updates to the build CR except when the build phase changes
					// as it's used to transition the integration kit from one phase
					// to another during the image build
					return e.ObjectOld.Status.Phase != e.ObjectNew.Status.Phase
				},
			},
		),
	)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &reconcileIntegrationKit{}

// reconcileIntegrationKit reconciles a IntegrationKit object.
type reconcileIntegrationKit struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the API server
	client   client.Client
	scheme   *runtime.Scheme
	recorder events.EventRecorder
	actions  []Action
}

// Reconcile reads that state of the cluster for a IntegrationKit object and makes changes based on the state read
// and what is in the IntegrationKit.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *reconcileIntegrationKit) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	rlog := Log.WithValues("request-namespace", request.Namespace, "request-name", request.Name)
	rlog.Debug("Reconciling IntegrationKit")

	// Make sure the operator is allowed to act on namespace
	if ok, err := platform.IsOperatorAllowedOnNamespace(ctx, r.client, request.Namespace); err != nil {
		log.Debugf("Error occurred when checking whether operator is allowed in namespace %s: %v", request.Namespace, err)

		return reconcile.Result{}, err
	} else if !ok {
		rlog.Info("Ignoring request because namespace is locked")

		return reconcile.Result{}, nil
	}

	var instance v1.IntegrationKit

	// Fetch the IntegrationKit instance
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
	targetLog := rlog.ForIntegrationKit(target)

	if target.Status.Phase == v1.IntegrationKitPhaseNone || target.Status.Phase == v1.IntegrationKitPhaseWaitingForPlatform {
		rlog.Debug("Preparing to shift integration kit phase")
		if target.IsExternal() {
			target.Status.Phase = v1.IntegrationKitPhaseInitialization

			return r.update(ctx, &instance, target)
		}

		target.Status.Phase = v1.IntegrationKitPhaseInitialization

		if instance.Status.Phase != target.Status.Phase {
			return r.update(ctx, &instance, target)
		}

		return reconcile.Result{}, nil
	}

	targetPhase := instance.Status.Phase

	for _, a := range r.actions {
		a.InjectClient(r.client)
		a.InjectLogger(targetLog)

		if !a.CanHandle(target) {
			continue
		}

		targetLog.Infof("Invoking action %s", a.Name())

		newTarget, err := a.Handle(ctx, target)

		if err != nil {
			camelevent.NotifyError(r.recorder, &instance, target, instance.Name, instance.Kind, err)

			return reconcile.Result{}, err
		}

		if newTarget != nil {
			if res, err := r.update(ctx, &instance, newTarget); err != nil {
				camelevent.NotifyError(r.recorder, &instance, target, target.Name, target.Kind, err)

				return res, err
			}

			targetPhase = newTarget.Status.Phase

			if targetPhase != instance.Status.Phase {
				targetLog.Info(
					"State transition",
					"phase-from", instance.Status.Phase,
					"phase-to", targetPhase,
				)
			}
		}

		// handle one action at time so the resource
		// is always at its latest state
		camelevent.NotifyIntegrationKitUpdated(ctx, r.client, r.recorder, &instance, newTarget)

		break
	}

	if targetPhase == v1.IntegrationKitPhaseWaitingForCatalog {
		// Requeue
		return reconcile.Result{
			RequeueAfter: requeueAfterDuration,
		}, nil
	}

	return reconcile.Result{}, nil
}

func (r *reconcileIntegrationKit) update(ctx context.Context, base *v1.IntegrationKit, target *v1.IntegrationKit) (reconcile.Result, error) {
	dgst, err := digest.ComputeForIntegrationKit(target)
	if err != nil {
		return reconcile.Result{}, err
	}

	target.Status.Digest = dgst
	target.Status.ObservedGeneration = base.Generation
	err = r.client.Status().Patch(ctx, target, ctrl.MergeFrom(base))

	return reconcile.Result{}, err
}

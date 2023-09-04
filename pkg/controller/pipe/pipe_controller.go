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

package pipe

import (
	"context"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/trait"

	camelevent "github.com/apache/camel-k/v2/pkg/event"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/monitoring"
)

// Add creates a new Pipe Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, mgr manager.Manager, c client.Client) error {
	return add(mgr, newReconciler(mgr, c))
}

func newReconciler(mgr manager.Manager, c client.Client) reconcile.Reconciler {
	return monitoring.NewInstrumentedReconciler(
		&ReconcilePipe{
			client:   c,
			scheme:   mgr.GetScheme(),
			recorder: mgr.GetEventRecorderFor("camel-k-pipe-controller"),
		},
		schema.GroupVersionKind{
			Group:   v1.SchemeGroupVersion.Group,
			Version: v1.SchemeGroupVersion.Version,
			Kind:    v1.PipeKind,
		},
	)
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("kamelet-binding-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Pipe
	err = c.Watch(source.Kind(mgr.GetCache(), &v1.Pipe{}),
		&handler.EnqueueRequestForObject{},
		platform.FilteringFuncs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldPipe, ok := e.ObjectOld.(*v1.Pipe)
				if !ok {
					return false
				}
				newPipe, ok := e.ObjectNew.(*v1.Pipe)
				if !ok {
					return false
				}

				// If traits have changed, the reconciliation loop must kick in as
				// traits may have impact
				sameTraits, err := trait.PipesHaveSameTraits(oldPipe, newPipe)
				if err != nil {
					Log.ForPipe(newPipe).Error(
						err,
						"unable to determine if old and new resource have the same traits")
				}
				if !sameTraits {
					return true
				}

				// Ignore updates to the binding status in which case metadata.Generation
				// does not change, or except when the binding phase changes as it's used
				// to transition from one phase to another
				return oldPipe.Generation != newPipe.Generation ||
					oldPipe.Status.Phase != newPipe.Status.Phase
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				// Evaluates to false if the object has been confirmed deleted
				return !e.DeleteStateUnknown
			},
		},
	)
	if err != nil {
		return err
	}

	// Watch Integration to propagate changes downstream
	err = c.Watch(source.Kind(mgr.GetCache(), &v1.Integration{}),
		handler.EnqueueRequestForOwner(
			mgr.GetScheme(),
			mgr.GetRESTMapper(),
			&v1.Pipe{},
			handler.OnlyControllerOwner(),
		),
	)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePipe{}

// ReconcilePipe reconciles a Pipe object.
type ReconcilePipe struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the API server
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a Pipe object and makes changes based
// on the state read and what is in the Pipe.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePipe) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	rlog := Log.WithValues("request-namespace", request.Namespace, "request-name", request.Name)
	rlog.Debug("Reconciling Pipe")

	// Make sure the operator is allowed to act on namespace
	if ok, err := platform.IsOperatorAllowedOnNamespace(ctx, r.client, request.Namespace); err != nil {
		return reconcile.Result{}, err
	} else if !ok {
		rlog.Info("Ignoring request because namespace is locked")
		return reconcile.Result{}, nil
	}

	// Fetch the Pipe instance
	var instance v1.Pipe

	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup
			// logic use finalizers.

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

	actions := []Action{
		NewInitializeAction(),
		NewMonitorAction(),
	}

	var err error

	target := instance.DeepCopy()
	targetLog := rlog.ForPipe(target)

	for _, a := range actions {
		a.InjectClient(r.client)
		a.InjectLogger(targetLog)

		if a.CanHandle(target) {
			targetLog.Debugf("Invoking action %s", a.Name())

			target, err = a.Handle(ctx, target)
			if err != nil {
				camelevent.NotifyPipeError(ctx, r.client, r.recorder, &instance, target, err)
				// Update the binding (mostly just to update its phase) if the new instance is returned
				if target != nil {
					_ = r.update(ctx, &instance, target, &targetLog)
				}
				return reconcile.Result{}, err
			}

			if target != nil {
				if err := r.update(ctx, &instance, target, &targetLog); err != nil {
					camelevent.NotifyPipeError(ctx, r.client, r.recorder, &instance, target, err)
					return reconcile.Result{}, err
				}
			}

			// handle one action at time so the resource
			// is always at its latest state
			camelevent.NotifyPipeUpdated(ctx, r.client, r.recorder, &instance, target)
			break
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcilePipe) update(ctx context.Context, base *v1.Pipe, target *v1.Pipe, log *log.Logger) error {
	target.Status.ObservedGeneration = base.Generation

	if err := r.client.Status().Patch(ctx, target, ctrl.MergeFrom(base)); err != nil {
		camelevent.NotifyPipeError(ctx, r.client, r.recorder, base, target, err)
		return err
	}

	if target.Status.Phase != base.Status.Phase {
		log.Info(
			"State transition",
			"phase-from", base.Status.Phase,
			"phase-to", target.Status.Phase,
		)

		if target.Status.Phase == v1.PipePhaseError {
			if cond := target.Status.GetCondition(v1.PipeIntegrationConditionError); cond != nil {
				log.Info(
					"Integration error",
					"reason", cond.GetReason(),
					"error-message", cond.GetMessage())
			}
		}
	}

	return nil
}

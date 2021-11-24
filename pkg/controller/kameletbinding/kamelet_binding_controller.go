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

package kameletbinding

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	camelevent "github.com/apache/camel-k/pkg/event"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/monitoring"
)

// Add creates a new KameletBinding Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	c, err := client.FromManager(mgr)
	if err != nil {
		return err
	}
	return add(mgr, newReconciler(mgr, c))
}

func newReconciler(mgr manager.Manager, c client.Client) reconcile.Reconciler {
	return monitoring.NewInstrumentedReconciler(
		&ReconcileKameletBinding{
			client:   c,
			scheme:   mgr.GetScheme(),
			recorder: mgr.GetEventRecorderFor("camel-k-kamelet-binding-controller"),
		},
		schema.GroupVersionKind{
			Group:   v1alpha1.SchemeGroupVersion.Group,
			Version: v1alpha1.SchemeGroupVersion.Version,
			Kind:    v1alpha1.KameletBindingKind,
		},
	)
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kamelet-binding-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KameletBinding
	err = c.Watch(&source.Kind{Type: &v1alpha1.KameletBinding{}},
		&handler.EnqueueRequestForObject{},
		platform.FilteringFuncs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldKameletBinding, ok := e.ObjectOld.(*v1alpha1.KameletBinding)
				if !ok {
					return false
				}
				newKameletBinding, ok := e.ObjectNew.(*v1alpha1.KameletBinding)
				if !ok {
					return false
				}
				// Ignore updates to the kameletBinding status in which case metadata.Generation
				// does not change, or except when the kameletBinding phase changes as it's used
				// to transition from one phase to another
				return oldKameletBinding.Generation != newKameletBinding.Generation ||
					oldKameletBinding.Status.Phase != newKameletBinding.Status.Phase
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
	err = c.Watch(&source.Kind{Type: &v1.Integration{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &v1alpha1.KameletBinding{},
		IsController: false,
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileKameletBinding{}

// ReconcileKameletBinding reconciles a KameletBinding object.
type ReconcileKameletBinding struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the API server
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a KameletBinding object and makes changes based
// on the state read and what is in the KameletBinding.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKameletBinding) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	rlog := Log.WithValues("request-namespace", request.Namespace, "request-name", request.Name)
	rlog.Info("Reconciling KameletBinding")

	// Make sure the operator is allowed to act on namespace
	if ok, err := platform.IsOperatorAllowedOnNamespace(ctx, r.client, request.Namespace); err != nil {
		return reconcile.Result{}, err
	} else if !ok {
		rlog.Info("Ignoring request because namespace is locked")
		return reconcile.Result{}, nil
	}

	// Fetch the KameletBinding instance
	var instance v1alpha1.KameletBinding

	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if errors.IsNotFound(err) {
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
	if !platform.IsOperatorHandler(&instance) {
		rlog.Info("Ignoring request because resource is not assigned to current operator")
		return reconcile.Result{}, nil
	}

	actions := []Action{
		NewInitializeAction(),
		NewMonitorAction(),
	}

	var targetPhase v1alpha1.KameletBindingPhase
	var err error

	target := instance.DeepCopy()
	targetLog := rlog.ForKameletBinding(target)

	for _, a := range actions {
		a.InjectClient(r.client)
		a.InjectLogger(targetLog)

		if a.CanHandle(target) {
			targetLog.Infof("Invoking action %s", a.Name())

			phaseFrom := target.Status.Phase

			target, err = a.Handle(ctx, target)
			if err != nil {
				camelevent.NotifyKameletBindingError(ctx, r.client, r.recorder, &instance, target, err)
				return reconcile.Result{}, err
			}

			if target != nil {
				if err := r.client.Status().Patch(ctx, target, ctrl.MergeFrom(&instance)); err != nil {
					camelevent.NotifyKameletBindingError(ctx, r.client, r.recorder, &instance, target, err)
					return reconcile.Result{}, err
				}

				targetPhase = target.Status.Phase

				if targetPhase != phaseFrom {
					targetLog.Info(
						"state transition",
						"phase-from", phaseFrom,
						"phase-to", target.Status.Phase,
					)
				}
			}

			// handle one action at time so the resource
			// is always at its latest state
			camelevent.NotifyKameletBindingUpdated(ctx, r.client, r.recorder, &instance, target)
			break
		}
	}

	if targetPhase == v1alpha1.KameletBindingPhaseReady {
		return reconcile.Result{}, nil
	}

	// Requeue
	return reconcile.Result{
		RequeueAfter: 5 * time.Second,
	}, nil
}

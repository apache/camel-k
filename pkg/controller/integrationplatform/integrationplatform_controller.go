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

package integrationplatform

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
	"github.com/apache/camel-k/pkg/client"
	camelevent "github.com/apache/camel-k/pkg/event"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/monitoring"
)

// Add creates a new IntegrationPlatform Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	c, err := client.FromManager(mgr)
	if err != nil {
		return err
	}
	return add(mgr, newReconciler(mgr, c))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c client.Client) reconcile.Reconciler {
	return monitoring.NewInstrumentedReconciler(
		&reconcileIntegrationPlatform{
			client:   c,
			reader:   mgr.GetAPIReader(),
			scheme:   mgr.GetScheme(),
			recorder: mgr.GetEventRecorderFor("camel-k-integration-platform-controller"),
		},
		schema.GroupVersionKind{
			Group:   v1.SchemeGroupVersion.Group,
			Version: v1.SchemeGroupVersion.Version,
			Kind:    v1.IntegrationPlatformKind,
		},
	)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("integrationplatform-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource IntegrationPlatform
	err = c.Watch(&source.Kind{Type: &v1.IntegrationPlatform{}},
		&handler.EnqueueRequestForObject{},
		platform.FilteringFuncs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldIntegrationPlatform := e.ObjectOld.(*v1.IntegrationPlatform)
				newIntegrationPlatform := e.ObjectNew.(*v1.IntegrationPlatform)
				// Ignore updates to the integration platform status in which case metadata.Generation
				// does not change, or except when the integration platform phase changes as it's used
				// to transition from one phase to another
				return oldIntegrationPlatform.Generation != newIntegrationPlatform.Generation ||
					oldIntegrationPlatform.Status.Phase != newIntegrationPlatform.Status.Phase
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

	return nil
}

var _ reconcile.Reconciler = &reconcileIntegrationPlatform{}

// reconcileIntegrationPlatform reconciles a IntegrationPlatform object
type reconcileIntegrationPlatform struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the API server
	client client.Client
	// Non-caching client
	reader   ctrl.Reader
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a IntegrationPlatform object and makes changes based
// on the state read and what is in the IntegrationPlatform.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *reconcileIntegrationPlatform) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	rlog := Log.WithValues("request-namespace", request.Namespace, "request-name", request.Name)
	rlog.Info("Reconciling IntegrationPlatform")

	// Make sure the operator is allowed to act on namespace
	if ok, err := platform.IsOperatorAllowedOnNamespace(ctx, r.client, request.Namespace); err != nil {
		return reconcile.Result{}, err
	} else if !ok {
		rlog.Info("Ignoring request because namespace is locked")
		return reconcile.Result{}, nil
	}

	// Fetch the IntegrationPlatform instance
	var instance v1.IntegrationPlatform

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
		NewWarmAction(r.reader),
		NewCreateAction(),
		NewMonitorAction(),
	}

	var targetPhase v1.IntegrationPlatformPhase
	var err error

	target := instance.DeepCopy()
	targetLog := rlog.ForIntegrationPlatform(target)

	for _, a := range actions {
		a.InjectClient(r.client)
		a.InjectLogger(targetLog)

		if a.CanHandle(target) {
			targetLog.Infof("Invoking action %s", a.Name())

			phaseFrom := target.Status.Phase

			target, err = a.Handle(ctx, target)
			if err != nil {
				camelevent.NotifyIntegrationPlatformError(ctx, r.client, r.recorder, &instance, target, err)
				return reconcile.Result{}, err
			}

			if target != nil {
				if err := r.client.Status().Patch(ctx, target, ctrl.MergeFrom(&instance)); err != nil {
					camelevent.NotifyIntegrationPlatformError(ctx, r.client, r.recorder, &instance, target, err)
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
			camelevent.NotifyIntegrationPlatformUpdated(ctx, r.client, r.recorder, &instance, target)
			break
		}
	}

	if targetPhase == v1.IntegrationPlatformPhaseReady {
		return reconcile.Result{}, nil
	}

	// Requeue
	return reconcile.Result{
		RequeueAfter: 5 * time.Second,
	}, nil
}

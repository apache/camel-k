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
package integrationcontext

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
)

// Add creates a new IntegrationContext Controller and adds it to the Manager. The Manager will set fields on the Controller
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
	return &ReconcileIntegrationContext{
		client: c,
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("integrationcontext-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource IntegrationContext
	err = c.Watch(&source.Kind{Type: &v1alpha1.IntegrationContext{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldIntegrationContext := e.ObjectOld.(*v1alpha1.IntegrationContext)
			newIntegrationContext := e.ObjectNew.(*v1alpha1.IntegrationContext)
			// Ignore updates to the integration context status in which case metadata.Generation
			// does not change, or except when the integration context phase changes as it's used
			// to transition from one phase to another
			return oldIntegrationContext.Generation != newIntegrationContext.Generation ||
				oldIntegrationContext.Status.Phase != newIntegrationContext.Status.Phase
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted
			return !e.DeleteStateUnknown
		},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Builds and requeue the owner IntegrationContext
	err = c.Watch(&source.Kind{Type: &v1alpha1.Build{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &v1alpha1.IntegrationContext{},
		},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldBuild := e.ObjectOld.(*v1alpha1.Build)
				newBuild := e.ObjectNew.(*v1alpha1.Build)
				// Ignore updates to the build CR except when the build phase changes
				// as it's used to transition the integration context from one phase
				// to another during the image build
				return oldBuild.Status.Phase != newBuild.Status.Phase
			},
		})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileIntegrationContext{}

// ReconcileIntegrationContext reconciles a IntegrationContext object
type ReconcileIntegrationContext struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a IntegrationContext object and makes changes based on the state read
// and what is in the IntegrationContext.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileIntegrationContext) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rlog := Log.WithValues("request-namespace", request.Namespace, "request-name", request.Name)
	rlog.Info("Reconciling IntegrationContext")

	ctx := context.TODO()

	// Fetch the IntegrationContext instance
	instance := &v1alpha1.IntegrationContext{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	integrationContextActionPool := []Action{
		NewInitializeAction(),
		NewBuildAction(),
		NewMonitorAction(),
	}

	ilog := rlog.ForIntegrationContext(instance)
	for _, a := range integrationContextActionPool {
		a.InjectClient(r.client)
		a.InjectLogger(ilog)
		if a.CanHandle(instance) {
			ilog.Infof("Invoking action %s", a.Name())
			if err := a.Handle(ctx, instance); err != nil {
				if k8serrors.IsConflict(err) {
					ilog.Error(err, "conflict")
					return reconcile.Result{
						Requeue: true,
					}, nil
				}

				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

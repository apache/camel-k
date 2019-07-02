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

	"github.com/apache/camel-k/pkg/platform"

	"github.com/apache/camel-k/pkg/util/digest"

	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/log"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Integration Controller and adds it to the Manager. The Manager will set fields on the Controller
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
	return &ReconcileIntegration{
		client: c,
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("integration-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Integration
	err = c.Watch(&source.Kind{Type: &v1alpha1.Integration{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldIntegration := e.ObjectOld.(*v1alpha1.Integration)
			newIntegration := e.ObjectNew.(*v1alpha1.Integration)
			// Ignore updates to the integration status in which case metadata.Generation does not change,
			// or except when the integration phase changes as it's used to transition from one phase
			// to another
			return oldIntegration.Generation != newIntegration.Generation ||
				oldIntegration.Status.Phase != newIntegration.Status.Phase
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted
			return !e.DeleteStateUnknown
		},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Builds and requeue the owner IntegrationKit
	err = c.Watch(&source.Kind{Type: &v1alpha1.Build{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &v1alpha1.Integration{},
		},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldBuild := e.ObjectOld.(*v1alpha1.Build)
				newBuild := e.ObjectNew.(*v1alpha1.Build)
				// Ignore updates to the build CR except when the build phase changes
				// as it's used to transition the integration from one phase to another
				// during image build
				return oldBuild.Status.Phase != newBuild.Status.Phase
			},
		},
	)

	if err != nil {
		return err
	}

	// Watch for IntegrationKit phase transitioning to ready or error and
	// enqueue requests for any integrations that are in phase waiting for
	// kit
	err = c.Watch(&source.Kind{Type: &v1alpha1.IntegrationKit{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			kit := a.Object.(*v1alpha1.IntegrationKit)
			var requests []reconcile.Request

			if kit.Status.Phase == v1alpha1.IntegrationKitPhaseReady || kit.Status.Phase == v1alpha1.IntegrationKitPhaseError {
				list := &v1alpha1.IntegrationList{}

				if err := mgr.GetClient().List(context.TODO(), &k8sclient.ListOptions{Namespace: kit.Namespace}, list); err != nil {
					log.Error(err, "Failed to retrieve integration list")
					return requests
				}

				for _, integration := range list.Items {
					if integration.Status.Phase == v1alpha1.IntegrationPhaseBuildingKit {
						log.Infof("Kit %s ready, wake-up integration: %s", kit.Name, integration.Name)
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
		}),
	})

	if err != nil {
		return err
	}

	// Watch for IntegrationPlatform phase transitioning to ready and enqueue
	// requests for any integrations that are in phase waiting for platform
	err = c.Watch(&source.Kind{Type: &v1alpha1.IntegrationPlatform{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			platform := a.Object.(*v1alpha1.IntegrationPlatform)
			var requests []reconcile.Request

			if platform.Status.Phase == v1alpha1.IntegrationPlatformPhaseReady {
				list := &v1alpha1.IntegrationList{}

				if err := mgr.GetClient().List(context.TODO(), &k8sclient.ListOptions{Namespace: platform.Namespace}, list); err != nil {
					log.Error(err, "Failed to retrieve integration list")
					return requests
				}

				for _, integration := range list.Items {
					if integration.Status.Phase == v1alpha1.IntegrationPhaseWaitingForPlatform {
						log.Infof("Platform %s ready, wake-up integration: %s", platform.Name, integration.Name)
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
		}),
	})

	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileIntegration{}

// ReconcileIntegration reconciles a Integration object
type ReconcileIntegration struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Integration object and makes changes based on the state read
// and what is in the Integration.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileIntegration) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rlog := Log.WithValues("request-namespace", request.Namespace, "request-name", request.Name)
	rlog.Info("Reconciling Integration")

	ctx := context.TODO()

	// Fetch the Integration instance
	var instance v1alpha1.Integration

	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Delete phase
	if instance.GetDeletionTimestamp() != nil {
		instance.Status.Phase = v1alpha1.IntegrationPhaseDeleting
	}

	target := instance.DeepCopy()
	targetLog := rlog.ForIntegration(target)

	if target.Status.Phase == v1alpha1.IntegrationPhaseNone || target.Status.Phase == v1alpha1.IntegrationPhaseWaitingForPlatform {
		pl, err := platform.GetCurrentPlatform(ctx, r.client, target.Namespace)
		switch {
		case err != nil:
			target.Status.Phase = v1alpha1.IntegrationPhaseError
			target.Status.Failure = v1alpha1.NewErrorFailure(err)
		case pl.Status.Phase != v1alpha1.IntegrationPlatformPhaseReady:
			target.Status.Phase = v1alpha1.IntegrationPhaseWaitingForPlatform
		default:
			target.Status.Phase = v1alpha1.IntegrationPhaseInitialization
		}

		if instance.Status.Phase != target.Status.Phase {
			err = r.update(ctx, target)
			if err != nil {
				if k8serrors.IsConflict(err) {
					targetLog.Error(err, "conflict")
					err = nil
				}
			}
		}

		return reconcile.Result{}, err
	}

	actions := []Action{
		NewInitializeAction(),
		NewBuildKitAction(),
		NewDeployAction(),
		NewMonitorAction(),
		NewErrorAction(),
		NewDeleteAction(),
	}

	for _, a := range actions {
		a.InjectClient(r.client)
		a.InjectLogger(targetLog)

		if a.CanHandle(target) {
			targetLog.Infof("Invoking action %s", a.Name())

			newTarget, err := a.Handle(ctx, target)
			if err != nil {
				return reconcile.Result{}, err
			}

			if newTarget != nil {
				if err := r.update(ctx, newTarget); err != nil {
					if k8serrors.IsConflict(err) {
						targetLog.Error(err, "conflict")
						return reconcile.Result{
							Requeue: true,
						}, nil
					}

					return reconcile.Result{}, err
				}

				if newTarget.Status.Phase != target.Status.Phase {
					targetLog.Info(
						"state transition",
						"phase-from", target.Status.Phase,
						"phase-to", newTarget.Status.Phase,
					)
				}
			}

			// handle one action at time so the resource
			// is always at its latest state
			break
		}
	}

	return reconcile.Result{}, nil
}

// Update --
func (r *ReconcileIntegration) update(ctx context.Context, target *v1alpha1.Integration) error {
	dgst, err := digest.ComputeForIntegration(target)
	if err != nil {
		return err
	}

	target.Status.Digest = dgst

	return r.client.Status().Update(ctx, target)
}

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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	camelevent "github.com/apache/camel-k/pkg/event"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/monitoring"
)

// Add creates a new IntegrationKit Controller and adds it to the Manager. The Manager will set fields on the Controller
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
		&reconcileIntegrationKit{
			client:   c,
			scheme:   mgr.GetScheme(),
			recorder: mgr.GetEventRecorderFor("camel-k-integration-kit-controller"),
		},
		schema.GroupVersionKind{
			Group:   v1.SchemeGroupVersion.Group,
			Version: v1.SchemeGroupVersion.Version,
			Kind:    v1.IntegrationKitKind,
		},
	)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("integrationkit-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource IntegrationKit
	err = c.Watch(&source.Kind{Type: &v1.IntegrationKit{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldIntegrationKit := e.ObjectOld.(*v1.IntegrationKit)
			newIntegrationKit := e.ObjectNew.(*v1.IntegrationKit)
			// Ignore updates to the integration kit status in which case metadata.Generation
			// does not change, or except when the integration kit phase changes as it's used
			// to transition from one phase to another
			return oldIntegrationKit.Generation != newIntegrationKit.Generation ||
				oldIntegrationKit.Status.Phase != newIntegrationKit.Status.Phase
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
	err = c.Watch(&source.Kind{Type: &v1.Build{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &v1.IntegrationKit{},
		},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldBuild := e.ObjectOld.(*v1.Build)
				newBuild := e.ObjectNew.(*v1.Build)
				// Ignore updates to the build CR except when the build phase changes
				// as it's used to transition the integration kit from one phase
				// to another during the image build
				return oldBuild.Status.Phase != newBuild.Status.Phase
			},
		})
	if err != nil {
		return err
	}

	// Watch for IntegrationPlatform phase transitioning to ready and enqueue
	// requests for any integration kits that are in phase waiting for platform
	err = c.Watch(&source.Kind{Type: &v1.IntegrationPlatform{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			platform := a.Object.(*v1.IntegrationPlatform)
			var requests []reconcile.Request

			if platform.Status.Phase == v1.IntegrationPlatformPhaseReady {
				list := &v1.IntegrationKitList{}

				if err := mgr.GetClient().List(context.TODO(), list, k8sclient.InNamespace(platform.Namespace)); err != nil {
					log.Error(err, "Failed to retrieve integrationkit list")
					return requests
				}

				for _, kit := range list.Items {
					if kit.Status.Phase == v1.IntegrationKitPhaseWaitingForPlatform {
						log.Infof("Platform %s ready, wake-up integrationkit: %s", platform.Name, kit.Name)
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: kit.Namespace,
								Name:      kit.Name,
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

var _ reconcile.Reconciler = &reconcileIntegrationKit{}

// reconcileIntegrationKit reconciles a IntegrationKit object
type reconcileIntegrationKit struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the API server
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a IntegrationKit object and makes changes based on the state read
// and what is in the IntegrationKit.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *reconcileIntegrationKit) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rlog := Log.WithValues("request-namespace", request.Namespace, "request-name", request.Name)
	rlog.Info("Reconciling IntegrationKit")

	ctx := context.TODO()

	// Make sure the operator is allowed to act on namespace
	if ok, err := platform.IsOperatorAllowedOnNamespace(ctx, r.client, request.Namespace); err != nil {
		return reconcile.Result{}, err
	} else if !ok {
		rlog.Info("Ignoring request because namespace is locked")
		return reconcile.Result{}, nil
	}

	var instance v1.IntegrationKit

	// Fetch the IntegrationKit instance
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

	target := instance.DeepCopy()
	targetLog := rlog.ForIntegrationKit(target)

	if target.Status.Phase == v1.IntegrationKitPhaseNone || target.Status.Phase == v1.IntegrationKitPhaseWaitingForPlatform {
		// TODO.... here local...
		pl, err := platform.GetOrFindLocal(ctx, r.client, target.Namespace, target.Status.Platform, true)
		if err != nil || pl.Status.Phase != v1.IntegrationPlatformPhaseReady {
			target.Status.Phase = v1.IntegrationKitPhaseWaitingForPlatform
		} else {
			target.Status.Phase = v1.IntegrationKitPhaseInitialization
		}

		if instance.Status.Phase != target.Status.Phase {
			if err != nil {
				target.Status.SetErrorCondition(v1.IntegrationKitConditionPlatformAvailable, v1.IntegrationKitConditionPlatformAvailableReason, err)
			}

			if pl != nil {
				target.SetIntegrationPlatform(pl)
			}

			return r.update(ctx, &instance, target)
		}

		return reconcile.Result{}, err
	}

	actions := []Action{
		NewInitializeAction(),
		NewBuildAction(),
		NewMonitorAction(),
		NewErrorAction(),
	}

	for _, a := range actions {
		a.InjectClient(r.client)
		a.InjectLogger(targetLog)

		if a.CanHandle(target) {
			targetLog.Infof("Invoking action %s", a.Name())

			newTarget, err := a.Handle(ctx, target)
			if err != nil {
				camelevent.NotifyIntegrationKitError(ctx, r.client, r.recorder, &instance, newTarget, err)
				return reconcile.Result{}, err
			}

			if newTarget != nil {
				if res, err := r.update(ctx, &instance, newTarget); err != nil {
					camelevent.NotifyIntegrationKitError(ctx, r.client, r.recorder, &instance, newTarget, err)
					return res, err
				}

				if newTarget.Status.Phase != instance.Status.Phase {
					targetLog.Info(
						"state transition",
						"phase-from", instance.Status.Phase,
						"phase-to", newTarget.Status.Phase,
					)
				}
			}

			// handle one action at time so the resource
			// is always at its latest state
			camelevent.NotifyIntegrationKitUpdated(ctx, r.client, r.recorder, &instance, newTarget)
			break
		}
	}

	return reconcile.Result{}, nil
}

func (r *reconcileIntegrationKit) update(ctx context.Context, base *v1.IntegrationKit, target *v1.IntegrationKit) (reconcile.Result, error) {
	dgst, err := digest.ComputeForIntegrationKit(target)
	if err != nil {
		return reconcile.Result{}, err
	}

	target.Status.Digest = dgst

	err = r.client.Status().Patch(ctx, target, k8sclient.MergeFrom(base))

	return reconcile.Result{}, err
}

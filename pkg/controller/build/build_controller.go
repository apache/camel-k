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
package build

import (
	"context"
	"sync"
	"time"

	"github.com/apache/camel-k/pkg/platform"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/log"
)

// Add creates a new Build Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	c, err := client.FromManager(mgr)
	if err != nil {
		return err
	}
	reconciler, err := newReconciler(mgr, c)
	if err != nil {
		return err
	}
	return add(mgr, reconciler)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c client.Client) (reconcile.Reconciler, error) {
	// Non-caching client to be used whenever caching may cause race conditions,
	// like in the builds scheduling critical section.
	// TODO: to be replaced with Manager.GetAPIReader() as soon as it's available, see:
	// https://github.com/kubernetes-sigs/controller-runtime/pull/327
	clientOptions := k8sclient.Options{
		Scheme: mgr.GetScheme(),
	}
	reader, err := k8sclient.New(mgr.GetConfig(), clientOptions)
	if err != nil {
		return nil, err
	}

	return &ReconcileBuild{
		client:  c,
		reader:  reader,
		scheme:  mgr.GetScheme(),
		builder: builder.New(c),
	}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("build-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Build
	err = c.Watch(&source.Kind{Type: &v1alpha1.Build{}}, &handler.EnqueueRequestForObject{},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldBuild := e.ObjectOld.(*v1alpha1.Build)
				newBuild := e.ObjectNew.(*v1alpha1.Build)
				// Ignore updates to the build status in which case metadata.Generation does not change,
				// or except when the build phase changes as it's used to transition from one phase
				// to another
				return oldBuild.Generation != newBuild.Generation ||
					oldBuild.Status.Phase != newBuild.Status.Phase
			},
		})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner Build
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &v1alpha1.Build{},
		},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldPod := e.ObjectOld.(*corev1.Pod)
				newPod := e.ObjectNew.(*corev1.Pod)
				// Ignore updates to the build pods except when the pod phase changes
				// as it's used to transition the builds from one phase to another
				return oldPod.Status.Phase != newPod.Status.Phase
			},
		})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileBuild{}

// ReconcileBuild reconciles a Build object
type ReconcileBuild struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	reader   k8sclient.Reader
	scheme   *runtime.Scheme
	builder  builder.Builder
	routines sync.Map
}

// Reconcile reads that state of the cluster for a Build object and makes changes based on the state read
// and what is in the Build.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileBuild) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rlog := Log.WithValues("request-namespace", request.Namespace, "request-name", request.Name)
	rlog.Info("Reconciling Build")

	ctx := context.TODO()

	// Fetch the Integration instance
	var instance v1alpha1.Build

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
	targetLog := rlog.ForBuild(target)

	if target.Status.Phase == v1alpha1.BuildPhaseNone || target.Status.Phase == v1alpha1.BuildPhaseWaitingForPlatform {
		pl, err := platform.GetCurrentPlatform(ctx, r.client, target.Namespace)
		switch {
		case err != nil:
			target.Status.Phase = v1alpha1.BuildPhaseError
			target.Status.Failure = v1alpha1.NewErrorFailure(err)
		case pl.Status.Phase != v1alpha1.IntegrationPlatformPhaseReady:
			target.Status.Phase = v1alpha1.BuildPhaseWaitingForPlatform
		default:
			target.Status.Phase = v1alpha1.BuildPhaseInitialization
		}

		if instance.Status.Phase != target.Status.Phase {
			return r.update(ctx, targetLog, target)
		}

		return reconcile.Result{}, err
	}

	actions := []Action{
		NewInitializeAction(),
		NewScheduleRoutineAction(r.reader, r.builder, &r.routines),
		NewSchedulePodAction(r.reader),
		NewMonitorRoutineAction(&r.routines),
		NewMonitorPodAction(),
		NewErrorRecoveryAction(),
		NewErrorAction(),
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
				if r, err := r.update(ctx, targetLog, newTarget); err != nil {
					return r, err
				}

				if newTarget.Status.Phase != target.Status.Phase {
					targetLog.Info(
						"state transition",
						"phase-from", target.Status.Phase,
						"phase-to", newTarget.Status.Phase,
					)
				}

				target = newTarget
			}

			// handle one action at time so the resource
			// is always at its latest state
			break
		}
	}

	// Requeue scheduling build so that it re-enters the build working queue
	if target.Status.Phase == v1alpha1.BuildPhaseScheduling || target.Status.Phase == v1alpha1.BuildPhaseFailed {
		return reconcile.Result{
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	return reconcile.Result{}, nil
}

// Update --
func (r *ReconcileBuild) update(ctx context.Context, log log.Logger, target *v1alpha1.Build) (reconcile.Result, error) {
	err := r.client.Status().Update(ctx, target)
	if err != nil {
		if k8serrors.IsConflict(err) {
			log.Error(err, "conflict")

			return reconcile.Result{
				Requeue: true,
			}, nil
		}
	}

	return reconcile.Result{}, err
}

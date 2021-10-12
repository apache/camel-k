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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	camelevent "github.com/apache/camel-k/pkg/event"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/monitoring"
)

// Add creates a new Build Controller and adds it to the Manager. The Manager will set fields on the Controller
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
		&reconcileBuild{
			client:   c,
			reader:   mgr.GetAPIReader(),
			scheme:   mgr.GetScheme(),
			recorder: mgr.GetEventRecorderFor("camel-k-build-controller"),
		},
		schema.GroupVersionKind{
			Group:   v1.SchemeGroupVersion.Group,
			Version: v1.SchemeGroupVersion.Version,
			Kind:    v1.BuildKind,
		},
	)
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	return builder.ControllerManagedBy(mgr).
		Named("build-controller").
		// Watch for changes to primary resource Build
		For(&v1.Build{}, builder.WithPredicates(
			platform.FilteringFuncs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldBuild := e.ObjectOld.(*v1.Build)
					newBuild := e.ObjectNew.(*v1.Build)
					// Ignore updates to the build status in which case metadata.Generation does not change,
					// or except when the build phase changes as it's used to transition from one phase
					// to another
					return oldBuild.Generation != newBuild.Generation ||
						oldBuild.Status.Phase != newBuild.Status.Phase
				},
			})).
		Complete(r)
}

var _ reconcile.Reconciler = &reconcileBuild{}

// reconcileBuild reconciles a Build object
type reconcileBuild struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the API server
	client client.Client
	// Non-caching client to be used whenever caching may cause race conditions,
	// like in the builds scheduling critical section
	reader   ctrl.Reader
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a Build object and makes changes based on the state read
// and what is in the Build.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *reconcileBuild) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	rlog := Log.WithValues("request-namespace", request.Namespace, "request-name", request.Name)
	rlog.Info("Reconciling Build")

	// Make sure the operator is allowed to act on namespace
	if ok, err := platform.IsOperatorAllowedOnNamespace(ctx, r.client, request.Namespace); err != nil {
		return reconcile.Result{}, err
	} else if !ok {
		rlog.Info("Ignoring request because namespace is locked")
		return reconcile.Result{}, nil
	}

	// Fetch the Build instance
	var instance v1.Build

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

	// Only process resources assigned to the operator
	if !platform.IsOperatorHandler(&instance) {
		rlog.Info("Ignoring request because resource is not assigned to current operator")
		return reconcile.Result{}, nil
	}

	target := instance.DeepCopy()
	targetLog := rlog.ForBuild(target)

	var actions []Action

	switch instance.Spec.Strategy {
	case v1.BuildStrategyPod:
		actions = []Action{
			newInitializePodAction(r.reader),
			newScheduleAction(r.reader),
			newMonitorPodAction(r.reader),
			newErrorRecoveryAction(),
			newErrorAction(),
		}
	case v1.BuildStrategyRoutine:
		actions = []Action{
			newInitializeRoutineAction(),
			newScheduleAction(r.reader),
			newMonitorRoutineAction(),
			newErrorRecoveryAction(),
			newErrorAction(),
		}
	}

	for _, a := range actions {
		a.InjectClient(r.client)
		a.InjectLogger(targetLog)
		a.InjectRecorder(r.recorder)

		if a.CanHandle(target) {
			targetLog.Infof("Invoking action %s", a.Name())

			newTarget, err := a.Handle(ctx, target)
			if err != nil {
				camelevent.NotifyBuildError(ctx, r.client, r.recorder, &instance, newTarget, err)
				return reconcile.Result{}, err
			}

			if newTarget != nil {
				if res, err := r.update(ctx, &instance, newTarget); err != nil {
					camelevent.NotifyBuildError(ctx, r.client, r.recorder, &instance, newTarget, err)
					return res, err
				}

				if newTarget.Status.Phase != instance.Status.Phase {
					targetLog.Info(
						"state transition",
						"phase-from", instance.Status.Phase,
						"phase-to", newTarget.Status.Phase,
					)
				}

				target = newTarget
			}

			// handle one action at time so the resource
			// is always at its latest state
			camelevent.NotifyBuildUpdated(ctx, r.client, r.recorder, &instance, newTarget)
			break
		}
	}

	if target.Status.Phase == v1.BuildPhaseScheduling || target.Status.Phase == v1.BuildPhaseFailed {
		// Requeue scheduling (resp. failed) build so that it re-enters the build (resp. recovery) working queue
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if target.Spec.Strategy == v1.BuildStrategyPod &&
		(target.Status.Phase == v1.BuildPhasePending || target.Status.Phase == v1.BuildPhaseRunning) {
		// Requeue running Build to poll Pod and signal timeout
		return reconcile.Result{RequeueAfter: 1 * time.Second}, nil
	}

	return reconcile.Result{}, nil
}

func (r *reconcileBuild) update(ctx context.Context, base *v1.Build, target *v1.Build) (reconcile.Result, error) {
	err := r.client.Status().Patch(ctx, target, ctrl.MergeFrom(base))

	return reconcile.Result{}, err
}

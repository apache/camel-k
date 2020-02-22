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

	"github.com/apache/camel-k/pkg/events"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/platform"
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

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c client.Client) reconcile.Reconciler {
	return &ReconcileBuild{
		client:   c,
		reader:   mgr.GetAPIReader(),
		scheme:   mgr.GetScheme(),
		builder:  builder.New(c),
		recorder: mgr.GetEventRecorderFor("camel-k-build-controller"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("build-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Build
	err = c.Watch(&source.Kind{Type: &v1.Build{}}, &handler.EnqueueRequestForObject{},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldBuild := e.ObjectOld.(*v1.Build)
				newBuild := e.ObjectNew.(*v1.Build)
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
			OwnerType:    &v1.Build{},
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
	client client.Client
	// Non-caching client to be used whenever caching may cause race conditions,
	// like in the builds scheduling critical section
	reader   k8sclient.Reader
	scheme   *runtime.Scheme
	builder  builder.Builder
	routines sync.Map
	recorder record.EventRecorder
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

	target := instance.DeepCopy()
	targetLog := rlog.ForBuild(target)

	pl, err := platform.GetOrLookupCurrent(ctx, r.client, target.Namespace, target.Status.Platform)
	if target.Status.Phase == v1.BuildPhaseNone || target.Status.Phase == v1.BuildPhaseWaitingForPlatform {
		if err != nil || pl.Status.Phase != v1.IntegrationPlatformPhaseReady {
			target.Status.Phase = v1.BuildPhaseWaitingForPlatform
		} else {
			target.Status.Phase = v1.BuildPhaseInitialization
		}

		if instance.Status.Phase != target.Status.Phase {
			if err != nil {
				target.Status.SetErrorCondition(v1.BuildConditionPlatformAvailable, v1.BuildConditionPlatformAvailableReason, err)
			}

			if pl != nil {
				target.SetIntegrationPlatform(pl)
			}

			return r.update(ctx, &instance, target)
		}

		return reconcile.Result{}, err
	}

	var actions []Action

	switch pl.Status.Build.BuildStrategy {
	case v1.IntegrationPlatformBuildStrategyPod:
		actions = []Action{
			NewInitializePodAction(),
			NewSchedulePodAction(r.reader),
			NewMonitorPodAction(),
			NewErrorRecoveryAction(),
			NewErrorAction(),
		}
	case v1.IntegrationPlatformBuildStrategyRoutine:
		actions = []Action{
			NewInitializeRoutineAction(),
			NewScheduleRoutineAction(r.reader, r.builder, &r.routines),
			NewMonitorRoutineAction(&r.routines),
			NewErrorRecoveryAction(),
			NewErrorAction(),
		}
	}

	for _, a := range actions {
		a.InjectClient(r.client)
		a.InjectLogger(targetLog)

		if a.CanHandle(target) {
			targetLog.Infof("Invoking action %s", a.Name())

			newTarget, err := a.Handle(ctx, target)
			if err != nil {
				events.NotifyBuildError(ctx, r.client, r.recorder, &instance, newTarget, err)
				return reconcile.Result{}, err
			}

			if newTarget != nil {
				if res, err := r.update(ctx, &instance, newTarget); err != nil {
					events.NotifyBuildError(ctx, r.client, r.recorder, &instance, newTarget, err)
					return res, err
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
			events.NotifyBuildUpdated(ctx, r.client, r.recorder, &instance, newTarget)
			break
		}
	}

	// Requeue scheduling (resp. failed) build so that it re-enters the build (resp. recovery) working queue
	if target.Status.Phase == v1.BuildPhaseScheduling || target.Status.Phase == v1.BuildPhaseFailed {
		return reconcile.Result{
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileBuild) update(ctx context.Context, base *v1.Build, target *v1.Build) (reconcile.Result, error) {
	err := r.client.Status().Patch(ctx, target, k8sclient.MergeFrom(base))

	return reconcile.Result{}, err
}

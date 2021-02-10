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
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/batch/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

	sb "github.com/redhat-developer/service-binding-operator/pkg/apis/operators/v1alpha1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	camelevent "github.com/apache/camel-k/pkg/event"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/monitoring"
)

// Add creates a new Integration Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	c, err := client.FromManager(mgr)
	if err != nil {
		return err
	}
	return add(mgr, newReconciler(mgr, c), c)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c client.Client) reconcile.Reconciler {
	return monitoring.NewInstrumentedReconciler(
		&reconcileIntegration{
			client:   c,
			scheme:   mgr.GetScheme(),
			recorder: mgr.GetEventRecorderFor("camel-k-integration-controller"),
		},
		schema.GroupVersionKind{
			Group:   v1.SchemeGroupVersion.Group,
			Version: v1.SchemeGroupVersion.Version,
			Kind:    v1.IntegrationKind,
		},
	)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, c client.Client) error {
	// Create a new controller
	ctrl, err := controller.New("integration-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Integration
	err = ctrl.Watch(&source.Kind{Type: &v1.Integration{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldIntegration := e.ObjectOld.(*v1.Integration)
			newIntegration := e.ObjectNew.(*v1.Integration)
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

	// Watch for IntegrationKit phase transitioning to ready or error and
	// enqueue requests for any integrations that are in phase waiting for
	// kit
	err = ctrl.Watch(&source.Kind{Type: &v1.IntegrationKit{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			kit := a.Object.(*v1.IntegrationKit)
			var requests []reconcile.Request

			if kit.Status.Phase == v1.IntegrationKitPhaseReady || kit.Status.Phase == v1.IntegrationKitPhaseError {
				list := &v1.IntegrationList{}

				if err := mgr.GetClient().List(context.TODO(), list, k8sclient.InNamespace(kit.Namespace)); err != nil {
					log.Error(err, "Failed to retrieve integration list")
					return requests
				}

				for _, integration := range list.Items {
					if integration.Status.Phase == v1.IntegrationPhaseBuildingKit {
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
	err = ctrl.Watch(&source.Kind{Type: &v1.IntegrationPlatform{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			platform := a.Object.(*v1.IntegrationPlatform)
			var requests []reconcile.Request

			if platform.Status.Phase == v1.IntegrationPlatformPhaseReady {
				list := &v1.IntegrationList{}

				if err := mgr.GetClient().List(context.TODO(), list, k8sclient.InNamespace(platform.Namespace)); err != nil {
					log.Error(err, "Failed to retrieve integration list")
					return requests
				}

				for _, integration := range list.Items {
					if integration.Status.Phase == v1.IntegrationPhaseWaitingForPlatform {
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

	// Watch for ReplicaSet to reconcile replicas to the integration status. We cannot use
	// the EnqueueRequestForOwner handler as the owner depends on the deployment strategy,
	// either regular deployment or Knative service. In any case, the integration is not the
	// direct owner of the ReplicaSet.
	err = ctrl.Watch(&source.Kind{Type: &appsv1.ReplicaSet{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			rs := a.Object.(*appsv1.ReplicaSet)
			var requests []reconcile.Request

			labels := rs.GetLabels()
			integrationName, ok := labels[v1.IntegrationLabel]
			if ok {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: rs.Namespace,
						Name:      integrationName,
					},
				})
			}

			return requests
		}),
	}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldReplicaSet := e.ObjectOld.(*appsv1.ReplicaSet)
			newReplicaSet := e.ObjectNew.(*appsv1.ReplicaSet)
			// Ignore updates to the ReplicaSet other than the replicas ones,
			// that are used to reconcile the integration replicas.
			return oldReplicaSet.Status.Replicas != newReplicaSet.Status.Replicas ||
				oldReplicaSet.Status.ReadyReplicas != newReplicaSet.Status.ReadyReplicas ||
				oldReplicaSet.Status.AvailableReplicas != newReplicaSet.Status.AvailableReplicas
		},
	})
	if err != nil {
		return err
	}

	// Watch cronjob to update the ready condition
	err = ctrl.Watch(&source.Kind{Type: &v1beta1.CronJob{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &v1.Integration{},
		IsController: false,
	})
	if err != nil {
		return err
	}

	// Check the ServiceBinding CRD is present
	if ok, err := kubernetes.IsAPIResourceInstalled(c, sb.SchemeGroupVersion.String(), reflect.TypeOf(sb.ServiceBinding{}).Name()); err != nil {
		return err
	} else if !ok {
		log.Info("Service binding is disabled, install the Service Binding Operator if needed")
	} else if ok, err := kubernetes.CheckPermission(context.TODO(), c, sb.SchemeGroupVersion.Group, "servicebindings", "", "", "create"); err != nil {
		return err
	} else if !ok {
		log.Info("Service binding is disabled, the operator is not granted permission to create ServiceBindings!")
	} else {
		// Watch ServiceBindings and enqueue owning Integrations
		err = ctrl.Watch(&source.Kind{Type: &sb.ServiceBinding{}}, &handler.EnqueueRequestForOwner{
			OwnerType:    &v1.Integration{},
			IsController: true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

var _ reconcile.Reconciler = &reconcileIntegration{}

// reconcileIntegration reconciles a Integration object
type reconcileIntegration struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the API server
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a Integration object and makes changes based on the state read
// and what is in the Integration.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *reconcileIntegration) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rlog := Log.WithValues("request-namespace", request.Namespace, "request-name", request.Name)
	rlog.Info("Reconciling Integration")

	ctx := context.TODO()

	// Make sure the operator is allowed to act on namespace
	if ok, err := platform.IsOperatorAllowedOnNamespace(ctx, r.client, request.Namespace); err != nil {
		return reconcile.Result{}, err
	} else if !ok {
		rlog.Info("Ignoring request because namespace is locked")
		return reconcile.Result{}, nil
	}

	// Fetch the Integration instance
	var instance v1.Integration

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

	target := instance.DeepCopy()
	targetLog := rlog.ForIntegration(target)

	actions := []Action{
		NewWaitForBindingsAction(),
		NewPlatformSetupAction(),
		NewInitializeAction(),
		NewBuildKitAction(),
		NewDeployAction(),
		NewMonitorAction(),
		NewErrorAction(),
		NewNoopAction(),
	}

	for _, a := range actions {
		a.InjectClient(r.client)
		a.InjectLogger(targetLog)

		if a.CanHandle(target) {
			targetLog.Infof("Invoking action %s", a.Name())

			newTarget, err := a.Handle(ctx, target)
			if err != nil {
				camelevent.NotifyIntegrationError(ctx, r.client, r.recorder, &instance, newTarget, err)
				return reconcile.Result{}, err
			}

			if newTarget != nil {
				if res, err := r.update(ctx, &instance, newTarget); err != nil {
					camelevent.NotifyIntegrationError(ctx, r.client, r.recorder, &instance, newTarget, err)
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
			camelevent.NotifyIntegrationUpdated(ctx, r.client, r.recorder, &instance, newTarget)
			break
		}
	}

	return reconcile.Result{}, nil
}

func (r *reconcileIntegration) update(ctx context.Context, base *v1.Integration, target *v1.Integration) (reconcile.Result, error) {
	dgst, err := digest.ComputeForIntegration(target)
	if err != nil {
		return reconcile.Result{}, err
	}

	target.Status.Digest = dgst

	err = r.client.Status().Patch(ctx, target, k8sclient.MergeFrom(base))

	return reconcile.Result{}, err
}

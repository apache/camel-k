package integrationcontext

import (
	"context"
	"time"

	camelv1alpha1 "github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_integrationcontext")

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
	return &ReconcileIntegrationContext{client: c, scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("integrationcontext-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource IntegrationContext
	err = c.Watch(&source.Kind{Type: &camelv1alpha1.IntegrationContext{}}, &handler.EnqueueRequestForObject{})
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
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling IntegrationContext")

	ctx := context.TODO()

	// Fetch the IntegrationContext instance
	instance := &camelv1alpha1.IntegrationContext{}
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
		NewBuildAction(ctx),
		NewErrorRecoveryAction(),
		NewMonitorAction(),
	}

	for _, a := range integrationContextActionPool {
		a.InjectClient(r.client)
		if a.CanHandle(instance) {
			logrus.Debug("Invoking action ", a.Name(), " on integration context ", instance.Name)
			if err := a.Handle(ctx, instance); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Fetch the IntegrationContext again and check the state
	if err = r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		return reconcile.Result{}, err
	}

	if instance.Status.Phase == camelv1alpha1.IntegrationContextPhaseReady {
		return reconcile.Result{}, nil
	}
	// Requeue
	return reconcile.Result{
		RequeueAfter: 5 * time.Second,
	}, nil
}

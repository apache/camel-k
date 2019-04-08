package integrationplatform

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

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
	return &ReconcileIntegrationPlatform{
		client: c,
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("integrationplatform-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource IntegrationPlatform
	err = c.Watch(&source.Kind{Type: &v1alpha1.IntegrationPlatform{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldIntegrationPlatform := e.ObjectOld.(*v1alpha1.IntegrationPlatform)
			newIntegrationPlatform := e.ObjectNew.(*v1alpha1.IntegrationPlatform)
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
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Builds and requeue the owner IntegrationContext
	err = c.Watch(&source.Kind{Type: &v1alpha1.IntegrationContext{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &v1alpha1.IntegrationPlatform{},
		},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldIctx := e.ObjectOld.(*v1alpha1.IntegrationContext)
				newIctx := e.ObjectNew.(*v1alpha1.IntegrationContext)
				// Ignore updates to the integration context CR except when the phase changes
				return oldIctx.Status.Phase != newIctx.Status.Phase
			},
		})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileIntegrationPlatform{}

// ReconcileIntegrationPlatform reconciles a IntegrationPlatform object
type ReconcileIntegrationPlatform struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a IntegrationPlatform object and makes changes based on the state read
// and what is in the IntegrationPlatform.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileIntegrationPlatform) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rlog := Log.WithValues("request-namespace", request.Namespace, "request-name", request.Name)
	rlog.Info("Reconciling IntegrationPlatform")

	ctx := context.TODO()

	// Fetch the IntegrationPlatform instance
	instance := &v1alpha1.IntegrationPlatform{}
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

	integrationPlatformActionPool := []Action{
		NewInitializeAction(),
		NewWarmAction(),
		NewCreateAction(),
		NewMonitorAction(),
	}

	ilog := rlog.ForIntegrationPlatform(instance)
	for _, a := range integrationPlatformActionPool {
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

	// Fetch the IntegrationPlatform again and check the state
	if err = r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		return reconcile.Result{}, err
	}

	if instance.Status.Phase == v1alpha1.IntegrationPlatformPhaseWarming {
		// We need to poll the Kaniko warmer pod status
		return reconcile.Result{
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	return reconcile.Result{}, nil
}

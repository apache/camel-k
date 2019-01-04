/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

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
	"fmt"

	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/knative/build/pkg/builder"
	clientset "github.com/knative/build/pkg/client/clientset/versioned"
	buildscheme "github.com/knative/build/pkg/client/clientset/versioned/scheme"
	informers "github.com/knative/build/pkg/client/informers/externalversions/build/v1alpha1"
	listers "github.com/knative/build/pkg/client/listers/build/v1alpha1"
	"github.com/knative/build/pkg/reconciler"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
)

const controllerAgentName = "build-controller"

// Reconciler is the controller.Reconciler implementation for Builds resources
type Reconciler struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// buildclientset is a clientset for our own API group
	buildclientset clientset.Interface

	buildsLister                listers.BuildLister
	buildTemplatesLister        listers.BuildTemplateLister
	clusterBuildTemplatesLister listers.ClusterBuildTemplateLister

	builder builder.Interface

	// Sugared logger is easier to use but is not as performant as the
	// raw logger. In performance critical paths, call logger.Desugar()
	// and use the returned raw logger instead. In addition to the
	// performance benefits, raw logger also preserves type-safety at
	// the expense of slightly greater verbosity.
	Logger *zap.SugaredLogger
}

// Check that we implement the controller.Reconciler interface.
var _ controller.Reconciler = (*Reconciler)(nil)

func init() {
	// Add build-controller types to the default Kubernetes Scheme so Events can be
	// logged for build-controller types.
	buildscheme.AddToScheme(scheme.Scheme)
}

// NewController returns a new build template controller
func NewController(
	logger *zap.SugaredLogger,
	kubeclientset kubernetes.Interface,
	buildclientset clientset.Interface,
	buildInformer informers.BuildInformer,
	buildTemplateInformer informers.BuildTemplateInformer,
	clusterBuildTemplateInformer informers.ClusterBuildTemplateInformer,
	builder builder.Interface,
) *controller.Impl {

	// Enrich the logs with controller name
	logger = logger.Named(controllerAgentName).With(zap.String(logkey.ControllerType, controllerAgentName))

	r := &Reconciler{
		kubeclientset:               kubeclientset,
		buildclientset:              buildclientset,
		buildsLister:                buildInformer.Lister(),
		buildTemplatesLister:        buildTemplateInformer.Lister(),
		clusterBuildTemplatesLister: clusterBuildTemplateInformer.Lister(),
		builder:                     builder,
		Logger:                      logger,
	}
	impl := controller.NewImpl(r, logger, "Builds",
		reconciler.MustNewStatsReporter("Builds", r.Logger))

	logger.Info("Setting up event handlers")
	// Set up an event handler for when Build resources change
	buildInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
	})

	// TODO(mattmoor): Set up a Pod informer, so that Pod updates
	// trigger Build reconciliations.

	return impl
}

// Reconcile implements controller.Reconciler
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	logger := logging.FromContext(ctx)

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Errorf("invalid resource key: %s", key)
		return nil
	}

	// Get the Build resource with this namespace/name
	build, err := c.buildsLister.Builds(namespace).Get(name)
	if errors.IsNotFound(err) {
		// The Build resource may no longer exist, in which case we stop processing.
		logger.Errorf("build %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}

	// Don't mutate the informer's copy of our object.
	build = build.DeepCopy()

	// Validate build
	if err = c.validateBuild(build); err != nil {
		c.Logger.Errorf("Failed to validate build: %v", err)
		return err
	}

	// If the build's done, then ignore it.
	if builder.IsDone(&build.Status) {
		return nil
	}

	// If the build is not done, but is in progress (has an operation), then asynchronously wait for it.
	// TODO(mattmoor): Check whether the Builder matches the kind of our c.builder.
	if build.Status.Builder != "" {
		op, err := c.builder.OperationFromStatus(&build.Status)
		if err != nil {
			return err
		}

		// Check if build has timed out
		if builder.IsTimeout(&build.Status, build.Spec.Timeout) {
			//cleanup operation and update status
			timeoutMsg := fmt.Sprintf("Build %q failed to finish within %q", build.Name, build.Spec.Timeout.Duration.String())

			if err := op.Terminate(); err != nil {
				c.Logger.Errorf("Failed to terminate pod: %v", err)
				return err
			}

			build.Status.SetCondition(&duckv1alpha1.Condition{
				Type:    v1alpha1.BuildSucceeded,
				Status:  corev1.ConditionFalse,
				Reason:  "BuildTimeout",
				Message: timeoutMsg,
			})
			// update build completed time
			build.Status.CompletionTime = metav1.Now()

			if _, err := c.updateStatus(build); err != nil {
				c.Logger.Errorf("Failed to update status for pod: %v", err)
				return err
			}

			c.Logger.Errorf("Timeout: %v", timeoutMsg)
			return nil
		}

		// if not timed out then wait async
		go c.waitForOperation(build, op)
		return nil
	}

	// If the build hasn't even started, then start it and record the operation in our status.
	// Note that by recording our status, we will trigger a reconciliation, so the wait above
	// will kick in.
	build.Status.Builder = c.builder.Builder()
	var tmpl v1alpha1.BuildTemplateInterface
	if build.Spec.Template != nil {
		if build.Spec.Template.Kind == v1alpha1.ClusterBuildTemplateKind {
			tmpl, err = c.clusterBuildTemplatesLister.Get(build.Spec.Template.Name)
			if err != nil {
				// The ClusterBuildTemplate resource may not exist.
				if errors.IsNotFound(err) {
					runtime.HandleError(fmt.Errorf("cluster build template %q does not exist", build.Spec.Template.Name))
				}
				return err
			}
		} else {
			tmpl, err = c.buildTemplatesLister.BuildTemplates(namespace).Get(build.Spec.Template.Name)
			if err != nil {
				// The BuildTemplate resource may not exist.
				if errors.IsNotFound(err) {
					runtime.HandleError(fmt.Errorf("build template %q in namespace %q does not exist", build.Spec.Template.Name, namespace))
				}
				return err
			}
		}
	}
	build, err = builder.ApplyTemplate(build, tmpl)
	if err != nil {
		return err
	}
	// TODO: Validate build except steps+template
	b, err := c.builder.BuildFromSpec(build)
	if err != nil {
		return err
	}
	op, err := b.Execute()
	if err != nil {
		build.Status.SetCondition(&duckv1alpha1.Condition{
			Type:    v1alpha1.BuildSucceeded,
			Status:  corev1.ConditionFalse,
			Reason:  "BuildExecuteFailed",
			Message: err.Error(),
		})

		if _, err := c.updateStatus(build); err != nil {
			return err
		}
		return err
	}
	if err := op.Checkpoint(build, &build.Status); err != nil {
		return err
	}
	build, err = c.updateStatus(build)
	if err != nil {
		return err
	}
	return nil
}

func (c *Reconciler) waitForOperation(build *v1alpha1.Build, op builder.Operation) error {
	status, err := op.Wait()
	if err != nil {
		c.Logger.Errorf("Error while waiting for operation: %v", err)
		return err
	}
	build.Status = *status
	if _, err := c.updateStatus(build); err != nil {
		c.Logger.Errorf("Error updating build status: %v", err)
		return err
	}
	return nil
}

func (c *Reconciler) updateStatus(u *v1alpha1.Build) (*v1alpha1.Build, error) {
	buildClient := c.buildclientset.BuildV1alpha1().Builds(u.Namespace)
	newu, err := buildClient.Get(u.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	newu.Status = u.Status

	// Until #38113 is merged, we must use Update instead of UpdateStatus to
	// update the Status block of the Build resource. UpdateStatus will not
	// allow changes to the Spec of the resource, which is ideal for ensuring
	// nothing other than resource status has been updated.
	return buildClient.Update(newu)
}

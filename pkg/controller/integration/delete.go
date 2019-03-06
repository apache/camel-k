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
	"fmt"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/finalizer"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/log"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewDeleteAction creates a new monitoring action for an integration
func NewDeleteAction() Action {
	return &deleteAction{}
}

type deleteAction struct {
	baseAction
}

func (action *deleteAction) Name() string {
	return "delete"
}

func (action *deleteAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseDeleting
}

func (action *deleteAction) Handle(ctx context.Context, integration *v1alpha1.Integration) error {
	l := log.Log.ForIntegration(integration)

	ok, err := finalizer.Exists(integration, finalizer.CamelIntegrationFinalizer)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	target := integration.DeepCopy()

	// Select all resources created by this integration
	selectors := []string{
		fmt.Sprintf("camel.apache.org/integration=%s", integration.Name),
	}

	l.Info("Collecting resources to delete")
	resources, err := kubernetes.LookUpResources(ctx, action.client, target.Namespace, selectors)
	if err != nil {
		return err
	}

	// If the ForegroundDeletion deletion is not set remove the finalizer and
	// delete child resources from a dedicated goroutine
	foreground, err := finalizer.Exists(target, finalizer.ForegroundDeletion)
	if err != nil {
		return err
	}

	if !foreground {
		//
		// Async
		//
		if err := action.removeFinalizer(ctx, target); err != nil {
			return err
		}

		go func() {
			if err := action.deleteChildResources(context.TODO(), &l, resources); err != nil {
				l.Error(err, "error deleting child resources")
			}
		}()
	} else {
		//
		// Sync
		//
		if err := action.deleteChildResources(ctx, &l, resources); err != nil {
			return err
		}
		if err = action.removeFinalizer(ctx, target); err != nil {
			return err
		}
	}

	return nil
}

func (action *deleteAction) removeFinalizer(ctx context.Context, integration *v1alpha1.Integration) error {
	_, err := finalizer.Remove(integration, finalizer.CamelIntegrationFinalizer)
	if err != nil {
		return err
	}

	return action.client.Update(ctx, integration)
}

func (action *deleteAction) deleteChildResources(ctx context.Context, l *log.Logger, resources []unstructured.Unstructured) error {
	l.Infof("Resources to delete: %d", len(resources))

	var err error

	resources, err = action.deleteChildResourceWithCondition(ctx, l, resources, func(u unstructured.Unstructured) bool {
		return u.GetKind() == "Service" && strings.HasPrefix(u.GetAPIVersion(), "serving.knative.dev/")
	})
	if err != nil {
		return err
	}

	resources, err = action.deleteChildResourceWithCondition(ctx, l, resources, func(u unstructured.Unstructured) bool {
		return u.GetKind() == "Deployment"
	})
	if err != nil {
		return err
	}

	resources, err = action.deleteChildResourceWithCondition(ctx, l, resources, func(u unstructured.Unstructured) bool {
		return u.GetKind() == "ReplicaSet"
	})
	if err != nil {
		return err
	}

	resources, err = action.deleteChildResourceWithCondition(ctx, l, resources, func(u unstructured.Unstructured) bool {
		return u.GetKind() == "Pod"
	})
	if err != nil {
		return err
	}

	// Delete remaining resources
	for _, resource := range resources {
		// pin the resource
		resource := resource

		if err := action.deleteChildResource(ctx, l, resource); err != nil {
			return err
		}
	}

	return nil
}

func (action *deleteAction) deleteChildResourceWithCondition(
	ctx context.Context, l *log.Logger, resources []unstructured.Unstructured, condition func(unstructured.Unstructured) bool) ([]unstructured.Unstructured, error) {

	remaining := resources[:0]
	for _, resource := range resources {
		// pin the resource
		resource := resource

		if condition(resource) {
			if err := action.deleteChildResource(ctx, l, resource); err != nil {
				return resources, err
			}

			continue
		}

		remaining = append(remaining, resource)
	}

	return remaining, nil
}

func (action *deleteAction) deleteChildResource(ctx context.Context, l *log.Logger, resource unstructured.Unstructured) error {
	l.Infof("Deleting child resource: %s:%s/%s", resource.GetAPIVersion(), resource.GetKind(), resource.GetName())

	err := action.client.Delete(ctx, &resource, k8sclient.PropagationPolicy(metav1.DeletePropagationOrphan))
	if err != nil {
		// The resource may have already been deleted
		if !k8serrors.IsNotFound(err) {
			l.Errorf(err, "cannot delete child resource: %s:%s/%s", resource.GetAPIVersion(), resource.GetKind(), resource.GetName())
		}
	} else {
		l.Infof("Child resource deleted: %s:%s/%s", resource.GetAPIVersion(), resource.GetKind(), resource.GetName())
	}

	return nil
}

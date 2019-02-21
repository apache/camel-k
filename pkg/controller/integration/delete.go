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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/apache/camel-k/pkg/util/finalizer"

	"github.com/apache/camel-k/pkg/util/log"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

	resources, err := kubernetes.LookUpResources(ctx, action.client, integration.Namespace, selectors)
	if err != nil {
		return err
	}

	// If the ForegroundDeletion deletion is set remove the finalizer and
	// delete child resources from a dedicated goroutine
	ok, err = finalizer.Exists(integration, finalizer.ForegroundDeletion)
	if err != nil {
		return err
	}

	if ok {
		//
		// Async
		//

		if err := action.removeFinalizer(ctx, target); err != nil {
			return err
		}

		go func() {
			if err := action.deleteResources(context.TODO(), target, resources); err != nil {
				l.Error(err, "error deleting child resources")
			}
		}()
	} else {
		//
		// Sync
		//
		if err := action.deleteResources(ctx, target, resources); err != nil {
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

func (action *deleteAction) deleteResources(ctx context.Context, integration *v1alpha1.Integration, resources []unstructured.Unstructured) error {
	l := log.Log.ForIntegration(integration)

	// And delete them
	for _, resource := range resources {
		// pin the resource
		resource := resource

		// Pods are automatically deleted by the removal of Deployment
		if resource.GetKind() == "Pod" {
			continue
		}
		// ReplicaSet are automatically deleted by the removal of Deployment
		if resource.GetKind() == "ReplicaSet" {
			continue
		}

		l.Infof("Deleting child resource: %s/%s", resource.GetKind(), resource.GetName())

		err := action.client.Delete(ctx, &resource)
		if err != nil {
			// The resource may have already been deleted
			if !k8serrors.IsNotFound(err) {
				l.Errorf(err, "cannot delete child resource: %s/%s", resource.GetKind(), resource.GetName())
			}
		} else {
			l.Infof("Child resource deleted: %s/%s", resource.GetKind(), resource.GetName())
		}
	}

	return nil
}

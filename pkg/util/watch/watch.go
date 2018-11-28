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

package watch

import (
	"context"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// StateChanges watches a integration resource and send it through a channel when its status changes
func StateChanges(ctx context.Context, integration *v1alpha1.Integration) (<-chan *v1alpha1.Integration, error) {
	resourceClient, _, err := k8sclient.GetResourceClient(integration.APIVersion, integration.Kind, integration.Namespace)
	if err != nil {
		return nil, err
	}
	watcher, err := resourceClient.Watch(metav1.ListOptions{
		FieldSelector: "metadata.name=" + integration.Name,
	})
	if err != nil {
		return nil, err
	}
	events := watcher.ResultChan()

	out := make(chan *v1alpha1.Integration)
	var lastObservedState *v1alpha1.IntegrationPhase

	go func() {
		defer watcher.Stop()
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case e, ok := <-events:
				if !ok {
					return
				}

				if e.Object != nil {
					if runtimeUnstructured, ok := e.Object.(runtime.Unstructured); ok {
						unstr := unstructured.Unstructured{
							Object: runtimeUnstructured.UnstructuredContent(),
						}
						icopy := integration.DeepCopy()
						err := k8sutil.UnstructuredIntoRuntimeObject(&unstr, icopy)
						if err != nil {
							logrus.Error("Unexpected error detected when watching resource", err)
							return // closes the channel
						}

						if lastObservedState == nil || *lastObservedState != icopy.Status.Phase {
							lastObservedState = &icopy.Status.Phase
							out <- icopy
						}
					}
				}
			}
		}
	}()

	return out, nil
}

//
// HandleStateChanges watches a integration resource and invoke the given handler when its status changes.
//
//     err := watch.HandleStateChanges(ctx, integration, func(i *v1alpha1.Integration) bool {
//         if i.Status.Phase == v1alpha1.IntegrationPhaseRunning {
//			    return false
//		    }
//
//		    return true
//	    })
//
// This function blocks until the handler function returns true or either the events channel or the context is closed.
//
func HandleStateChanges(ctx context.Context, integration *v1alpha1.Integration, handler func(integration *v1alpha1.Integration) bool) error {
	resourceClient, _, err := k8sclient.GetResourceClient(integration.APIVersion, integration.Kind, integration.Namespace)
	if err != nil {
		return err
	}
	watcher, err := resourceClient.Watch(metav1.ListOptions{
		FieldSelector: "metadata.name=" + integration.Name,
	})
	if err != nil {
		return err
	}

	defer watcher.Stop()
	events := watcher.ResultChan()

	var lastObservedState *v1alpha1.IntegrationPhase

	for {
		select {
		case <-ctx.Done():
			return nil
		case e, ok := <-events:
			if !ok {
				return nil
			}

			if e.Object != nil {
				if runtimeUnstructured, ok := e.Object.(runtime.Unstructured); ok {
					unstr := unstructured.Unstructured{
						Object: runtimeUnstructured.UnstructuredContent(),
					}
					icopy := integration.DeepCopy()
					err := k8sutil.UnstructuredIntoRuntimeObject(&unstr, icopy)
					if err != nil {
						logrus.Error("Unexpected error detected when watching resource", err)
						return nil
					}

					if lastObservedState == nil || *lastObservedState != icopy.Status.Phase {
						lastObservedState = &icopy.Status.Phase
						if !handler(icopy) {
							return nil
						}
					}
				}
			}
		}
	}
}

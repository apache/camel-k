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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"context"
	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"github.com/sirupsen/logrus"
)

// Watches a integration resource and send it through a channel when its status changes
func WatchStateChanges(ctx context.Context, integration *v1alpha1.Integration) (<-chan *v1alpha1.Integration, error) {
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

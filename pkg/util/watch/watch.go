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
	"fmt"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/kubernetes/customclient"
	"github.com/apache/camel-k/pkg/util/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
)

//
// HandleIntegrationStateChanges watches a integration resource and invoke the given handler when its status changes.
//
//     err := watch.HandleIntegrationStateChanges(ctx, integration, func(i *v1.Integration) bool {
//         if i.Status.Phase == v1.IntegrationPhaseRunning {
//			    return false
//		    }
//
//		    return true
//	    })
//
// This function blocks until the handler function returns true or either the events channel or the context is closed.
//
func HandleIntegrationStateChanges(ctx context.Context, integration *v1.Integration,
	handler func(integration *v1.Integration) bool) (*v1.IntegrationPhase, error) {
	dynamicClient, err := customclient.GetDefaultDynamicClientFor("integrations", integration.Namespace)
	if err != nil {
		return nil, err
	}

	watcher, err := dynamicClient.Watch(ctx, metav1.ListOptions{
		FieldSelector:   "metadata.name=" + integration.Name,
		ResourceVersion: integration.ObjectMeta.ResourceVersion,
	})
	if err != nil {
		return nil, err
	}

	defer watcher.Stop()
	events := watcher.ResultChan()

	var lastObservedState *v1.IntegrationPhase

	var handlerWrapper = func(it *v1.Integration) bool {
		if lastObservedState == nil || *lastObservedState != it.Status.Phase {
			lastObservedState = &it.Status.Phase
			if !handler(it) {
				return false
			}
		}
		return true
	}

	// Check completion before starting the watch
	if !handlerWrapper(integration) {
		return lastObservedState, nil
	}

	for {
		select {
		case <-ctx.Done():
			return lastObservedState, nil
		case e, ok := <-events:
			if !ok {
				return lastObservedState, nil
			}

			if e.Object != nil {
				if runtimeUnstructured, ok := e.Object.(runtime.Unstructured); ok {
					jsondata, err := kubernetes.ToJSON(runtimeUnstructured)
					if err != nil {
						return nil, err
					}
					copy := integration.DeepCopy()
					err = json.Unmarshal(jsondata, copy)
					if err != nil {
						log.Error(err, "Unexpected error detected when watching resource")
						return lastObservedState, nil
					}

					if !handlerWrapper(copy) {
						return lastObservedState, nil
					}
				}
			}
		}
	}
}

//
// HandleIntegrationEvents watches all events related to the given integration.
//
//     watch.HandleIntegrationEvents(o.Context, integration, func(event *corev1.Event) bool {
//		 println(event.Message)
//		 return true
//	   })
//
// This function blocks until the handler function returns true or either the events channel or the context is closed.
//
func HandleIntegrationEvents(ctx context.Context, integration *v1.Integration,
	handler func(event *corev1.Event) bool) error {
	dynamicClient, err := customclient.GetDynamicClientFor("", "v1", "events", integration.Namespace)
	if err != nil {
		return err
	}
	watcher, err := dynamicClient.Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.kind=Integration,"+
			"involvedObject.apiVersion=%s,"+
			"involvedObject.name=%s",
			v1.SchemeGroupVersion.String(), integration.Name),
	})
	if err != nil {
		return err
	}

	defer watcher.Stop()
	events := watcher.ResultChan()

	var lastEvent *corev1.Event
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
					jsondata, err := kubernetes.ToJSON(runtimeUnstructured)
					if err != nil {
						return err
					}
					evt := corev1.Event{}
					err = json.Unmarshal(jsondata, &evt)
					if err != nil {
						log.Error(err, "Unexpected error detected when watching resource")
						return nil
					}

					if isAllowed(lastEvent, &evt, integration.CreationTimestamp.UnixNano()) {
						lastEvent = &evt
						if !handler(&evt) {
							return nil
						}
					}
				}
			}
		}
	}
}

//
// HandlePlatformStateChanges watches a platform resource and invoke the given handler when its status changes.
//
//     err := watch.HandlePlatformStateChanges(ctx, platform, func(i *v1.IntegrationPlatform) bool {
//         if i.Status.Phase == v1.IntegrationPlatformPhaseReady {
//			    return false
//		    }
//
//		    return true
//	    })
//
// This function blocks until the handler function returns true or either the events channel or the context is closed.
//
func HandlePlatformStateChanges(ctx context.Context, platform *v1.IntegrationPlatform, handler func(platform *v1.IntegrationPlatform) bool) error {
	dynamicClient, err := customclient.GetDefaultDynamicClientFor("integrationplatforms", platform.Namespace)
	if err != nil {
		return err
	}
	watcher, err := dynamicClient.Watch(ctx, metav1.ListOptions{
		FieldSelector: "metadata.name=" + platform.Name,
	})
	if err != nil {
		return err
	}

	defer watcher.Stop()
	events := watcher.ResultChan()

	var lastObservedState *v1.IntegrationPlatformPhase

	var handlerWrapper = func(pl *v1.IntegrationPlatform) bool {
		if lastObservedState == nil || *lastObservedState != pl.Status.Phase {
			lastObservedState = &pl.Status.Phase
			if !handler(pl) {
				return false
			}
		}
		return true
	}

	// Check completion before starting the watch
	if !handlerWrapper(platform) {
		return nil
	}

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
					jsondata, err := kubernetes.ToJSON(runtimeUnstructured)
					if err != nil {
						return err
					}
					copy := platform.DeepCopy()
					err = json.Unmarshal(jsondata, copy)
					if err != nil {
						log.Error(err, "Unexpected error detected when watching resource")
						return nil
					}

					if !handlerWrapper(copy) {
						return nil
					}
				}
			}
		}
	}
}

//
// HandleIntegrationPlatformEvents watches all events related to the given integration platform.
//
//     watch.HandleIntegrationPlatformEvents(o.Context, platform, func(event *corev1.Event) bool {
//		 println(event.Message)
//		 return true
//	   })
//
// This function blocks until the handler function returns true or either the events channel or the context is closed.
//
func HandleIntegrationPlatformEvents(ctx context.Context, p *v1.IntegrationPlatform,
	handler func(event *corev1.Event) bool) error {
	dynamicClient, err := customclient.GetDynamicClientFor("", "v1", "events", p.Namespace)
	if err != nil {
		return err
	}
	watcher, err := dynamicClient.Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.kind=IntegrationPlatform,"+
			"involvedObject.apiVersion=%s,"+
			"involvedObject.name=%s",
			v1.SchemeGroupVersion.String(), p.Name),
	})
	if err != nil {
		return err
	}

	defer watcher.Stop()
	events := watcher.ResultChan()

	var lastEvent *corev1.Event
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
					jsondata, err := kubernetes.ToJSON(runtimeUnstructured)
					if err != nil {
						return err
					}
					evt := corev1.Event{}
					err = json.Unmarshal(jsondata, &evt)
					if err != nil {
						log.Error(err, "Unexpected error detected when watching resource")
						return nil
					}

					if isAllowed(lastEvent, &evt, p.CreationTimestamp.UnixNano()) {
						lastEvent = &evt
						if !handler(&evt) {
							return nil
						}
					}
				}
			}
		}
	}
}

func isAllowed(lastEvent, event *corev1.Event, baseTime int64) bool {
	if lastEvent == nil {
		return true
	}

	curTime := event.CreationTimestamp.UnixNano()
	if event.LastTimestamp.UnixNano() > curTime {
		curTime = event.LastTimestamp.UnixNano()
	}
	if curTime < baseTime {
		return false
	}

	lastTime := lastEvent.CreationTimestamp.UnixNano()
	if lastEvent.LastTimestamp.UnixNano() > lastTime {
		lastTime = lastEvent.LastTimestamp.UnixNano()
	}
	if curTime < lastTime {
		return false
	}

	if lastEvent.Reason != event.Reason {
		return true
	}
	if lastEvent.Message != event.Message {
		return true
	}
	if lastEvent.Type != event.Type {
		return true
	}
	if lastEvent.InvolvedObject.Kind != event.InvolvedObject.Kind {
		return true
	}
	if lastEvent.InvolvedObject.APIVersion != event.InvolvedObject.APIVersion {
		return true
	}
	if lastEvent.InvolvedObject.Namespace != event.InvolvedObject.Namespace {
		return true
	}
	if lastEvent.InvolvedObject.Name != event.InvolvedObject.Name {
		return true
	}
	return false
}

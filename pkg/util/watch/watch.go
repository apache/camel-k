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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
)

// HandleIntegrationStateChanges watches an Integration resource and invoke the given handler when its status changes.
// This function blocks until the handler function returns true or either the events channel or the context is closed.
func HandleIntegrationStateChanges(ctx context.Context, c client.Client, integration *v1.Integration,
	handler func(integration *v1.Integration) bool) (*v1.IntegrationPhase, error) {
	watcher, err := c.CamelV1().Integrations(integration.Namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector:   "metadata.name=" + integration.Name,
		ResourceVersion: integration.ObjectMeta.ResourceVersion,
	})
	if err != nil {
		return nil, err
	}

	defer watcher.Stop()
	events := watcher.ResultChan()

	var lastObservedState *v1.IntegrationPhase

	handlerWrapper := func(it *v1.Integration) bool {
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
				if it, ok := e.Object.(*v1.Integration); ok {
					if !handlerWrapper(it) {
						return lastObservedState, nil
					}
				}
			}
		}
	}
}

// HandleIntegrationEvents watches all events related to the given integration.
// This function blocks until the handler function returns true or either the events channel or the context is closed.
func HandleIntegrationEvents(ctx context.Context, c client.Client, integration *v1.Integration,
	handler func(event *corev1.Event) bool) error {
	watcher, err := c.CoreV1().Events(integration.Namespace).
		Watch(ctx, metav1.ListOptions{
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
			if e.Object == nil {
				continue
			}
			if evt, ok := e.Object.(*corev1.Event); ok {
				if isAllowed(lastEvent, evt, integration.CreationTimestamp.UnixNano()) {
					lastEvent = evt
					if !handler(evt) {
						return nil
					}
				}
			}
		}
	}
}

// HandlePlatformStateChanges watches a platform resource and invoke the given handler when its status changes.
// This function blocks until the handler function returns true or either the events channel or the context is closed.
func HandlePlatformStateChanges(ctx context.Context, c client.Client, platform *v1.IntegrationPlatform,
	handler func(platform *v1.IntegrationPlatform) bool) error {
	watcher, err := c.CamelV1().IntegrationPlatforms(platform.Namespace).
		Watch(ctx, metav1.ListOptions{
			FieldSelector: "metadata.name=" + platform.Name,
		})
	if err != nil {
		return err
	}

	defer watcher.Stop()
	events := watcher.ResultChan()

	var lastObservedState *v1.IntegrationPlatformPhase

	handlerWrapper := func(pl *v1.IntegrationPlatform) bool {
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
				if p, ok := e.Object.(*v1.IntegrationPlatform); ok {
					if !handlerWrapper(p) {
						return nil
					}
				}
			}
		}
	}
}

// HandleIntegrationPlatformEvents watches all events related to the given integration platform.
// This function blocks until the handler function returns true or either the events channel or the context is closed.
func HandleIntegrationPlatformEvents(ctx context.Context, c client.Client, p *v1.IntegrationPlatform,
	handler func(event *corev1.Event) bool) error {
	watcher, err := c.CoreV1().Events(p.Namespace).
		Watch(ctx, metav1.ListOptions{
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
				if evt, ok := e.Object.(*corev1.Event); ok {
					if isAllowed(lastEvent, evt, p.CreationTimestamp.UnixNano()) {
						lastEvent = evt
						if !handler(evt) {
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

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

package event

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/log"
)

const (
	// ReasonIntegrationPhaseUpdated --.
	ReasonIntegrationPhaseUpdated = "IntegrationPhaseUpdated"
	// ReasonIntegrationConditionChanged --.
	ReasonIntegrationConditionChanged = "IntegrationConditionChanged"
	// ReasonIntegrationError --.
	ReasonIntegrationError = "IntegrationError"

	// ReasonIntegrationKitPhaseUpdated --.
	ReasonIntegrationKitPhaseUpdated = "IntegrationKitPhaseUpdated"
	// ReasonIntegrationKitConditionChanged --.
	ReasonIntegrationKitConditionChanged = "IntegrationKitConditionChanged"
	// ReasonIntegrationKitError --.
	ReasonIntegrationKitError = "IntegrationKitError"

	// ReasonIntegrationPlatformPhaseUpdated --.
	ReasonIntegrationPlatformPhaseUpdated = "IntegrationPlatformPhaseUpdated"
	// ReasonIntegrationPlatformConditionChanged --.
	ReasonIntegrationPlatformConditionChanged = "IntegrationPlatformConditionChanged"
	// ReasonIntegrationPlatformError --.
	ReasonIntegrationPlatformError = "IntegrationPlatformError"

	// ReasonBuildPhaseUpdated --.
	ReasonBuildPhaseUpdated = "BuildPhaseUpdated"
	// ReasonBuildConditionChanged --.
	ReasonBuildConditionChanged = "BuildConditionChanged"
	// ReasonBuildError --.
	ReasonBuildError = "BuildError"

	// ReasonKameletError --.
	ReasonKameletError = "KameletError"
	// ReasonKameletConditionChanged --.
	ReasonKameletConditionChanged = "KameletConditionChanged"
	// ReasonKameletPhaseUpdated --.
	ReasonKameletPhaseUpdated = "KameletPhaseUpdated"

	// ReasonKameletBindingError --.
	ReasonKameletBindingError = "KameletBindingError"
	// ReasonKameletBindingConditionChanged --.
	ReasonKameletBindingConditionChanged = "KameletBindingConditionChanged"
	// ReasonKameletBindingPhaseUpdated --.
	ReasonKameletBindingPhaseUpdated = "KameletBindingPhaseUpdated"

	// ReasonRelatedObjectChanged --.
	ReasonRelatedObjectChanged = "ReasonRelatedObjectChanged"
)

// NotifyIntegrationError automatically generates error events when the integration reconcile cycle phase has an error.
func NotifyIntegrationError(ctx context.Context, c client.Client, recorder record.EventRecorder,
	old, newResource *v1.Integration, err error) {
	it := old
	if newResource != nil {
		it = newResource
	}
	if it == nil {
		return
	}
	recorder.Eventf(it, corev1.EventTypeWarning, ReasonIntegrationError,
		"Cannot reconcile Integration %s: %v",
		it.Name, err)
}

// NotifyIntegrationUpdated automatically generates events when the integration changes.
func NotifyIntegrationUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder,
	old, newResource *v1.Integration) {
	if newResource == nil {
		return
	}
	oldPhase := ""
	var oldConditions []v1.ResourceCondition
	if old != nil {
		oldPhase = string(old.Status.Phase)
		oldConditions = old.Status.GetConditions()
	}
	if newResource.Status.Phase != v1.IntegrationPhaseNone {
		notifyIfConditionUpdated(recorder, newResource, oldConditions, newResource.Status.GetConditions(),
			"Integration", newResource.Name, ReasonIntegrationConditionChanged)
	}
	notifyIfPhaseUpdated(ctx, c, recorder, newResource, oldPhase, string(newResource.Status.Phase),
		"Integration", newResource.Name, ReasonIntegrationPhaseUpdated, "")
}

// NotifyIntegrationKitUpdated automatically generates events when an integration kit changes.
func NotifyIntegrationKitUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder,
	old, newResource *v1.IntegrationKit) {
	if newResource == nil {
		return
	}
	oldPhase := ""
	var oldConditions []v1.ResourceCondition
	if old != nil {
		oldPhase = string(old.Status.Phase)
		oldConditions = old.Status.GetConditions()
	}
	if newResource.Status.Phase != v1.IntegrationKitPhaseNone {
		notifyIfConditionUpdated(recorder, newResource, oldConditions, newResource.Status.GetConditions(),
			"Integration Kit", newResource.Name, ReasonIntegrationKitConditionChanged)
	}
	notifyIfPhaseUpdated(ctx, c, recorder, newResource, oldPhase, string(newResource.Status.Phase),
		"Integration Kit", newResource.Name, ReasonIntegrationKitPhaseUpdated, "")
}

// NotifyIntegrationKitError automatically generates error events when the integration kit reconcile cycle phase
// has an error.
func NotifyIntegrationKitError(ctx context.Context, c client.Client, recorder record.EventRecorder,
	old, newResource *v1.IntegrationKit, err error) {
	kit := old
	if newResource != nil {
		kit = newResource
	}
	if kit == nil {
		return
	}
	recorder.Eventf(kit, corev1.EventTypeWarning, ReasonIntegrationKitError,
		"Cannot reconcile Integration Kit %s: %v",
		kit.Name, err)
}

// NotifyIntegrationPlatformUpdated automatically generates events when an integration platform changes.
func NotifyIntegrationPlatformUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder,
	old, newResource *v1.IntegrationPlatform) {
	if newResource == nil {
		return
	}
	oldPhase := ""
	var oldConditions []v1.ResourceCondition
	if old != nil {
		oldPhase = string(old.Status.Phase)
		oldConditions = old.Status.GetConditions()
	}
	if newResource.Status.Phase != v1.IntegrationPlatformPhaseNone {
		notifyIfConditionUpdated(recorder, newResource, oldConditions, newResource.Status.GetConditions(),
			"Integration Platform", newResource.Name, ReasonIntegrationPlatformConditionChanged)
	}
	notifyIfPhaseUpdated(ctx, c, recorder, newResource, oldPhase, string(newResource.Status.Phase),
		"Integration Platform", newResource.Name, ReasonIntegrationPlatformPhaseUpdated, "")
}

// NotifyIntegrationPlatformError automatically generates error events when the integration Platform
// reconcile cycle phase has an error.
func NotifyIntegrationPlatformError(ctx context.Context, c client.Client, recorder record.EventRecorder,
	old, newResource *v1.IntegrationPlatform, err error) {
	p := old
	if newResource != nil {
		p = newResource
	}
	if p == nil {
		return
	}
	recorder.Eventf(p, corev1.EventTypeWarning, ReasonIntegrationPlatformError,
		"Cannot reconcile Integration Platform %s: %v",
		p.Name, err)
}

// NotifyKameletUpdated automatically generates events when a Kamelet changes.
func NotifyKameletUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder,
	old, newResource *v1alpha1.Kamelet) {
	if newResource == nil {
		return
	}
	oldPhase := ""
	var oldConditions []v1.ResourceCondition
	if old != nil {
		oldPhase = string(old.Status.Phase)
		oldConditions = old.Status.GetConditions()
	}
	if newResource.Status.Phase != v1alpha1.KameletPhaseNone {
		notifyIfConditionUpdated(recorder, newResource, oldConditions, newResource.Status.GetConditions(),
			"Kamelet", newResource.Name, ReasonKameletConditionChanged)
	}
	notifyIfPhaseUpdated(ctx, c, recorder, newResource, oldPhase, string(newResource.Status.Phase),
		"Kamelet", newResource.Name, ReasonKameletPhaseUpdated, "")
}

// NotifyKameletError automatically generates error events when the kamelet reconcile cycle phase has an error.
func NotifyKameletError(ctx context.Context, c client.Client, recorder record.EventRecorder,
	old, newResource *v1alpha1.Kamelet, err error) {
	k := old
	if newResource != nil {
		k = newResource
	}
	if k == nil {
		return
	}
	recorder.Eventf(k, corev1.EventTypeWarning, ReasonKameletError, "Cannot reconcile Kamelet %s: %v", k.Name, err)
}

// NotifyKameletBindingUpdated automatically generates events when a KameletBinding changes.
func NotifyKameletBindingUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder,
	old, newResource *v1alpha1.KameletBinding) {
	if newResource == nil {
		return
	}
	oldPhase := ""
	var oldConditions []v1.ResourceCondition
	if old != nil {
		oldPhase = string(old.Status.Phase)
		oldConditions = old.Status.GetConditions()
	}
	if newResource.Status.Phase != v1alpha1.KameletBindingPhaseNone {
		notifyIfConditionUpdated(recorder, newResource, oldConditions, newResource.Status.GetConditions(),
			"KameletBinding", newResource.Name, ReasonKameletBindingConditionChanged)
	}
	notifyIfPhaseUpdated(ctx, c, recorder, newResource, oldPhase, string(newResource.Status.Phase),
		"KameletBinding", newResource.Name, ReasonKameletBindingPhaseUpdated, "")
}

// NotifyKameletBindingError automatically generates error events when the kameletBinding reconcile cycle phase
// has an error.
func NotifyKameletBindingError(ctx context.Context, c client.Client, recorder record.EventRecorder,
	old, newResource *v1alpha1.KameletBinding, err error) {
	k := old
	if newResource != nil {
		k = newResource
	}
	if k == nil {
		return
	}
	recorder.Eventf(k, corev1.EventTypeWarning, ReasonKameletError,
		"Cannot reconcile KameletBinding %s: %v",
		k.Name, err)
}

// NotifyBuildUpdated automatically generates events when a build changes.
func NotifyBuildUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder,
	old, newResource *v1.Build) {
	if newResource == nil {
		return
	}
	oldPhase := ""
	var oldConditions []v1.ResourceCondition
	if old != nil {
		oldPhase = string(old.Status.Phase)
		oldConditions = old.Status.GetConditions()
	}
	if newResource.Status.Phase != v1.BuildPhaseNone {
		notifyIfConditionUpdated(recorder, newResource, oldConditions, newResource.Status.GetConditions(),
			"Build", newResource.Name, ReasonBuildConditionChanged)
	}
	info := ""
	if newResource.Status.Failure != nil {
		attempt := newResource.Status.Failure.Recovery.Attempt
		attemptMax := newResource.Status.Failure.Recovery.AttemptMax
		info = fmt.Sprintf(" (recovery %d of %d)", attempt, attemptMax)
	}
	notifyIfPhaseUpdated(ctx, c, recorder, newResource, oldPhase, string(newResource.Status.Phase),
		"Build", newResource.Name, ReasonBuildPhaseUpdated, info)
}

// NotifyBuildError automatically generates error events when the build reconcile cycle phase has an error.
func NotifyBuildError(ctx context.Context, c client.Client, recorder record.EventRecorder,
	old, newResource *v1.Build, err error) {
	p := old
	if newResource != nil {
		p = newResource
	}
	if p == nil {
		return
	}
	recorder.Eventf(p, corev1.EventTypeWarning, ReasonBuildError, "Cannot reconcile Build %s: %v", p.Name, err)
}

func notifyIfPhaseUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder, newResource ctrl.Object,
	oldPhase, newPhase string, resourceType, name, reason, info string) {
	if oldPhase == newPhase {
		return
	}

	// Update information about phase changes
	phase := newPhase
	if phase == "" {
		phase = "[none]"
	}
	recorder.Eventf(newResource, corev1.EventTypeNormal, reason, "%s %q in phase %q%s", resourceType, name, phase, info)

	if creatorRef, creator := getCreatorObject(ctx, c, newResource); creatorRef != nil && creator != nil {
		if namespace := newResource.GetNamespace(); namespace == creatorRef.Namespace {
			recorder.Eventf(creator, corev1.EventTypeNormal, ReasonRelatedObjectChanged,
				"%s %q, created by %s %q, changed phase to %q%s",
				resourceType, name, creatorRef.Kind, creatorRef.Name, phase, info)
		} else {
			recorder.Eventf(creator, corev1.EventTypeNormal, ReasonRelatedObjectChanged,
				"%s \"%s/%s\", created by %s %q, changed phase to %q%s",
				resourceType, namespace, name, creatorRef.Kind, creatorRef.Name, phase, info)
		}
	}
}

func notifyIfConditionUpdated(recorder record.EventRecorder, newResource runtime.Object,
	oldConditions, newConditions []v1.ResourceCondition, resourceType, name, reason string) {
	// Update information about changes in conditions
	for _, cond := range getCommonChangedConditions(oldConditions, newConditions) {
		tail := ""
		if cond.GetMessage() != "" {
			tail = fmt.Sprintf(": %s", cond.GetMessage())
		}
		recorder.Eventf(newResource, corev1.EventTypeNormal, reason,
			"Condition %q is %q for %s %s%s",
			cond.GetType(), cond.GetStatus(), resourceType, name, tail)
	}
}

func getCommonChangedConditions(oldConditions, newConditions []v1.ResourceCondition) []v1.ResourceCondition {
	oldState := make(map[string]v1.ResourceCondition)
	for _, c := range oldConditions {
		oldState[c.GetType()] = c
	}

	var res []v1.ResourceCondition
	for _, newCond := range newConditions {
		oldCond := oldState[newCond.GetType()]
		if oldCond == nil || oldCond.GetStatus() != newCond.GetStatus() ||
			oldCond.GetMessage() != newCond.GetMessage() {
			res = append(res, newCond)
		}
	}
	return res
}

func getCreatorObject(ctx context.Context, c client.Client, obj runtime.Object) (
	*corev1.ObjectReference, runtime.Object,
) {
	if ref := kubernetes.GetCamelCreator(obj); ref != nil {
		if ref.Kind == "Integration" {
			it := v1.NewIntegration(ref.Namespace, ref.Name)
			if err := c.Get(ctx, ctrl.ObjectKeyFromObject(&it), &it); err != nil {
				log.Infof("Cannot get information about the creator Integration %v: %v", ref, err)
				return nil, nil
			}
			return ref, &it
		}
	}
	return nil, nil
}

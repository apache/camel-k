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
	// ReasonIntegrationPhaseUpdated --
	ReasonIntegrationPhaseUpdated = "IntegrationPhaseUpdated"
	// ReasonIntegrationConditionChanged --
	ReasonIntegrationConditionChanged = "IntegrationConditionChanged"
	// ReasonIntegrationError --
	ReasonIntegrationError = "IntegrationError"

	// ReasonIntegrationKitPhaseUpdated --
	ReasonIntegrationKitPhaseUpdated = "IntegrationKitPhaseUpdated"
	// ReasonIntegrationKitConditionChanged --
	ReasonIntegrationKitConditionChanged = "IntegrationKitConditionChanged"
	// ReasonIntegrationKitError --
	ReasonIntegrationKitError = "IntegrationKitError"

	// ReasonIntegrationPlatformPhaseUpdated --
	ReasonIntegrationPlatformPhaseUpdated = "IntegrationPlatformPhaseUpdated"
	// ReasonIntegrationPlatformConditionChanged --
	ReasonIntegrationPlatformConditionChanged = "IntegrationPlatformConditionChanged"
	// ReasonIntegrationPlatformError --
	ReasonIntegrationPlatformError = "IntegrationPlatformError"

	// ReasonBuildPhaseUpdated --
	ReasonBuildPhaseUpdated = "BuildPhaseUpdated"
	// ReasonBuildConditionChanged --
	ReasonBuildConditionChanged = "BuildConditionChanged"
	// ReasonBuildError --
	ReasonBuildError = "BuildError"

	// ReasonKameletError --
	ReasonKameletError = "KameletError"
	// ReasonKameletConditionChanged --
	ReasonKameletConditionChanged = "KameletConditionChanged"
	// ReasonKameletPhaseUpdated --
	ReasonKameletPhaseUpdated = "KameletPhaseUpdated"

	// ReasonKameletBindingError --
	ReasonKameletBindingError = "KameletBindingError"
	// ReasonKameletBindingConditionChanged --
	ReasonKameletBindingConditionChanged = "KameletBindingConditionChanged"
	// ReasonKameletBindingPhaseUpdated --
	ReasonKameletBindingPhaseUpdated = "KameletBindingPhaseUpdated"

	// ReasonRelatedObjectChanged --
	ReasonRelatedObjectChanged = "ReasonRelatedObjectChanged"
)

// NotifyIntegrationError automatically generates error events when the integration reconcile cycle phase has an error
func NotifyIntegrationError(ctx context.Context, c client.Client, recorder record.EventRecorder, old, new *v1.Integration, err error) {
	it := old
	if new != nil {
		it = new
	}
	if it == nil {
		return
	}
	recorder.Eventf(it, corev1.EventTypeWarning, ReasonIntegrationError, "Cannot reconcile Integration %s: %v", it.Name, err)
}

// NotifyIntegrationUpdated automatically generates events when the integration changes
func NotifyIntegrationUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder, old, new *v1.Integration) {
	if new == nil {
		return
	}
	oldPhase := ""
	var oldConditions []v1.ResourceCondition
	if old != nil {
		oldPhase = string(old.Status.Phase)
		oldConditions = old.Status.GetConditions()
	}
	if new.Status.Phase != v1.IntegrationPhaseNone {
		notifyIfConditionUpdated(recorder, new, oldConditions, new.Status.GetConditions(), "Integration", new.Name, ReasonIntegrationConditionChanged)
	}
	notifyIfPhaseUpdated(ctx, c, recorder, new, oldPhase, string(new.Status.Phase), "Integration", new.Name, ReasonIntegrationPhaseUpdated, "")
}

// NotifyIntegrationKitUpdated automatically generates events when an integration kit changes
func NotifyIntegrationKitUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder, old, new *v1.IntegrationKit) {
	if new == nil {
		return
	}
	oldPhase := ""
	var oldConditions []v1.ResourceCondition
	if old != nil {
		oldPhase = string(old.Status.Phase)
		oldConditions = old.Status.GetConditions()
	}
	if new.Status.Phase != v1.IntegrationKitPhaseNone {
		notifyIfConditionUpdated(recorder, new, oldConditions, new.Status.GetConditions(), "Integration Kit", new.Name, ReasonIntegrationKitConditionChanged)
	}
	notifyIfPhaseUpdated(ctx, c, recorder, new, oldPhase, string(new.Status.Phase), "Integration Kit", new.Name, ReasonIntegrationKitPhaseUpdated, "")
}

// NotifyIntegrationKitError automatically generates error events when the integration kit reconcile cycle phase has an error
func NotifyIntegrationKitError(ctx context.Context, c client.Client, recorder record.EventRecorder, old, new *v1.IntegrationKit, err error) {
	kit := old
	if new != nil {
		kit = new
	}
	if kit == nil {
		return
	}
	recorder.Eventf(kit, corev1.EventTypeWarning, ReasonIntegrationKitError, "Cannot reconcile Integration Kit %s: %v", kit.Name, err)
}

// NotifyIntegrationPlatformUpdated automatically generates events when an integration platform changes
func NotifyIntegrationPlatformUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder, old, new *v1.IntegrationPlatform) {
	if new == nil {
		return
	}
	oldPhase := ""
	var oldConditions []v1.ResourceCondition
	if old != nil {
		oldPhase = string(old.Status.Phase)
		oldConditions = old.Status.GetConditions()
	}
	if new.Status.Phase != v1.IntegrationPlatformPhaseNone {
		notifyIfConditionUpdated(recorder, new, oldConditions, new.Status.GetConditions(), "Integration Platform", new.Name, ReasonIntegrationPlatformConditionChanged)
	}
	notifyIfPhaseUpdated(ctx, c, recorder, new, oldPhase, string(new.Status.Phase), "Integration Platform", new.Name, ReasonIntegrationPlatformPhaseUpdated, "")
}

// NotifyIntegrationPlatformError automatically generates error events when the integration Platform reconcile cycle phase has an error
func NotifyIntegrationPlatformError(ctx context.Context, c client.Client, recorder record.EventRecorder, old, new *v1.IntegrationPlatform, err error) {
	p := old
	if new != nil {
		p = new
	}
	if p == nil {
		return
	}
	recorder.Eventf(p, corev1.EventTypeWarning, ReasonIntegrationPlatformError, "Cannot reconcile Integration Platform %s: %v", p.Name, err)
}

// NotifyKameletUpdated automatically generates events when a Kamelet changes
func NotifyKameletUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder, old, new *v1alpha1.Kamelet) {
	if new == nil {
		return
	}
	oldPhase := ""
	var oldConditions []v1.ResourceCondition
	if old != nil {
		oldPhase = string(old.Status.Phase)
		oldConditions = old.Status.GetConditions()
	}
	if new.Status.Phase != v1alpha1.KameletPhaseNone {
		notifyIfConditionUpdated(recorder, new, oldConditions, new.Status.GetConditions(), "Kamelet", new.Name, ReasonKameletConditionChanged)
	}
	notifyIfPhaseUpdated(ctx, c, recorder, new, oldPhase, string(new.Status.Phase), "Kamelet", new.Name, ReasonKameletPhaseUpdated, "")
}

// NotifyKameletError automatically generates error events when the kamelet reconcile cycle phase has an error
func NotifyKameletError(ctx context.Context, c client.Client, recorder record.EventRecorder, old, new *v1alpha1.Kamelet, err error) {
	k := old
	if new != nil {
		k = new
	}
	if k == nil {
		return
	}
	recorder.Eventf(k, corev1.EventTypeWarning, ReasonKameletError, "Cannot reconcile Kamelet %s: %v", k.Name, err)
}

// NotifyKameletBindingUpdated automatically generates events when a KameletBinding changes
func NotifyKameletBindingUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder, old, new *v1alpha1.KameletBinding) {
	if new == nil {
		return
	}
	oldPhase := ""
	var oldConditions []v1.ResourceCondition
	if old != nil {
		oldPhase = string(old.Status.Phase)
		oldConditions = old.Status.GetConditions()
	}
	if new.Status.Phase != v1alpha1.KameletBindingPhaseNone {
		notifyIfConditionUpdated(recorder, new, oldConditions, new.Status.GetConditions(), "KameletBinding", new.Name, ReasonKameletBindingConditionChanged)
	}
	notifyIfPhaseUpdated(ctx, c, recorder, new, oldPhase, string(new.Status.Phase), "KameletBinding", new.Name, ReasonKameletBindingPhaseUpdated, "")
}

// NotifyKameletBindingError automatically generates error events when the kameletBinding reconcile cycle phase has an error
func NotifyKameletBindingError(ctx context.Context, c client.Client, recorder record.EventRecorder, old, new *v1alpha1.KameletBinding, err error) {
	k := old
	if new != nil {
		k = new
	}
	if k == nil {
		return
	}
	recorder.Eventf(k, corev1.EventTypeWarning, ReasonKameletError, "Cannot reconcile KameletBinding %s: %v", k.Name, err)
}

// NotifyBuildUpdated automatically generates events when a build changes
func NotifyBuildUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder, old, new *v1.Build) {
	if new == nil {
		return
	}
	oldPhase := ""
	var oldConditions []v1.ResourceCondition
	if old != nil {
		oldPhase = string(old.Status.Phase)
		oldConditions = old.Status.GetConditions()
	}
	if new.Status.Phase != v1.BuildPhaseNone {
		notifyIfConditionUpdated(recorder, new, oldConditions, new.Status.GetConditions(), "Build", new.Name, ReasonBuildConditionChanged)
	}
	info := ""
	if new.Status.Failure != nil {
		attempt := new.Status.Failure.Recovery.Attempt
		attemptMax := new.Status.Failure.Recovery.AttemptMax
		info = fmt.Sprintf(" (recovery %d of %d)", attempt, attemptMax)
	}
	notifyIfPhaseUpdated(ctx, c, recorder, new, oldPhase, string(new.Status.Phase), "Build", new.Name, ReasonBuildPhaseUpdated, info)
}

// NotifyBuildError automatically generates error events when the build reconcile cycle phase has an error
func NotifyBuildError(ctx context.Context, c client.Client, recorder record.EventRecorder, old, new *v1.Build, err error) {
	p := old
	if new != nil {
		p = new
	}
	if p == nil {
		return
	}
	recorder.Eventf(p, corev1.EventTypeWarning, ReasonBuildError, "Cannot reconcile Build %s: %v", p.Name, err)
}

// nolint:lll
func notifyIfPhaseUpdated(ctx context.Context, c client.Client, recorder record.EventRecorder, new ctrl.Object, oldPhase, newPhase string, resourceType, name, reason, info string) {
	// Update information about phase changes
	if oldPhase != newPhase {
		phase := newPhase
		if phase == "" {
			phase = "[none]"
		}
		recorder.Eventf(new, corev1.EventTypeNormal, reason, "%s %q in phase %q%s", resourceType, name, phase, info)

		if creatorRef, creator := getCreatorObject(ctx, c, new); creatorRef != nil && creator != nil {
			if namespace := new.GetNamespace(); namespace == creatorRef.Namespace {
				recorder.Eventf(creator, corev1.EventTypeNormal, ReasonRelatedObjectChanged, "%s %q, created by %s %q, changed phase to %q%s", resourceType, name, creatorRef.Kind, creatorRef.Name, phase, info)
			} else {
				recorder.Eventf(creator, corev1.EventTypeNormal, ReasonRelatedObjectChanged, "%s \"%s/%s\", created by %s %q, changed phase to %q%s", resourceType, namespace, name, creatorRef.Kind, creatorRef.Name, phase, info)
			}
		}
	}
}

func notifyIfConditionUpdated(recorder record.EventRecorder, new runtime.Object, oldConditions, newConditions []v1.ResourceCondition, resourceType, name, reason string) {
	// Update information about changes in conditions
	for _, cond := range getCommonChangedConditions(oldConditions, newConditions) {
		tail := ""
		if cond.GetMessage() != "" {
			tail = fmt.Sprintf(": %s", cond.GetMessage())
		}
		recorder.Eventf(new, corev1.EventTypeNormal, reason, "Condition %q is %q for %s %s%s", cond.GetType(), cond.GetStatus(), resourceType, name, tail)
	}
}

func getCommonChangedConditions(old, new []v1.ResourceCondition) (res []v1.ResourceCondition) {
	oldState := make(map[string]v1.ResourceCondition)
	for _, c := range old {
		oldState[c.GetType()] = c
	}

	for _, newCond := range new {
		oldCond := oldState[newCond.GetType()]
		if oldCond == nil || oldCond.GetStatus() != newCond.GetStatus() || oldCond.GetMessage() != newCond.GetMessage() {
			res = append(res, newCond)
		}
	}
	return res
}

func getCreatorObject(ctx context.Context, c client.Client, obj runtime.Object) (ref *corev1.ObjectReference, creator runtime.Object) {
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

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

package kameletbinding

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/v2/pkg/trait"
)

// NewMonitorAction returns an action that monitors the Binding after it's fully initialized.
func NewMonitorAction() Action {
	return &monitorAction{}
}

type monitorAction struct {
	baseAction
}

func (action *monitorAction) Name() string {
	return "monitor"
}

func (action *monitorAction) CanHandle(binding *v1alpha1.KameletBinding) bool {
	return binding.Status.Phase == v1alpha1.KameletBindingPhaseCreating ||
		(binding.Status.Phase == v1alpha1.KameletBindingPhaseError &&
			binding.Status.GetCondition(v1alpha1.KameletBindingIntegrationConditionError) == nil) ||
		binding.Status.Phase == v1alpha1.KameletBindingPhaseReady
}

func (action *monitorAction) Handle(ctx context.Context, binding *v1alpha1.KameletBinding) (*v1alpha1.KameletBinding, error) {
	key := client.ObjectKey{
		Namespace: binding.Namespace,
		Name:      binding.Name,
	}
	it := v1.Integration{}
	if err := action.client.Get(ctx, key, &it); err != nil && k8serrors.IsNotFound(err) {
		target := binding.DeepCopy()
		// Rebuild the integration
		target.Status.Phase = v1alpha1.KameletBindingPhaseNone
		target.Status.SetCondition(
			v1alpha1.KameletBindingConditionReady,
			corev1.ConditionFalse,
			"",
			"",
		)
		return target, nil
	} else if err != nil {
		return nil, fmt.Errorf("could not load integration for Binding %q: %w", binding.Name, err)
	}

	operatorIDChanged := v1.GetOperatorIDAnnotation(binding) != "" &&
		(v1.GetOperatorIDAnnotation(binding) != v1.GetOperatorIDAnnotation(&it))

	sameTraits, err := trait.IntegrationAndKameletBindingSameTraits(&it, binding)
	if err != nil {
		return nil, err
	}

	// Check if the integration needs to be changed
	expected, err := CreateIntegrationFor(ctx, action.client, binding)
	if err != nil {
		binding.Status.Phase = v1alpha1.KameletBindingPhaseError
		binding.Status.SetErrorCondition(v1alpha1.KameletBindingIntegrationConditionError,
			"Couldn't create an Integration custom resource", err)
		return binding, err
	}

	semanticEquality := equality.Semantic.DeepDerivative(expected.Spec, it.Spec)

	if !semanticEquality || operatorIDChanged || !sameTraits {
		action.L.Info(
			"Binding needs a rebuild",
			"semantic-equality", !semanticEquality,
			"operatorid-changed", operatorIDChanged,
			"traits-changed", !sameTraits)

		// Binding has changed and needs rebuild
		target := binding.DeepCopy()
		// Rebuild the integration
		target.Status.Phase = v1alpha1.KameletBindingPhaseNone
		target.Status.SetCondition(
			v1alpha1.KameletBindingConditionReady,
			corev1.ConditionFalse,
			"",
			"",
		)
		return target, nil
	}

	// Map integration phase and conditions to Binding
	target := binding.DeepCopy()

	switch it.Status.Phase {

	case v1.IntegrationPhaseRunning:
		target.Status.Phase = v1alpha1.KameletBindingPhaseReady
		setKameletBindingReadyCondition(target, &it)

	case v1.IntegrationPhaseError:
		target.Status.Phase = v1alpha1.KameletBindingPhaseError
		setKameletBindingReadyCondition(target, &it)

	default:
		target.Status.Phase = v1alpha1.KameletBindingPhaseCreating

		c := v1alpha1.KameletBindingCondition{
			Type:    v1alpha1.KameletBindingConditionReady,
			Status:  corev1.ConditionFalse,
			Reason:  string(target.Status.Phase),
			Message: fmt.Sprintf("Integration %q is in %q phase", it.GetName(), target.Status.Phase),
		}

		if condition := it.Status.GetCondition(v1.IntegrationConditionReady); condition != nil {
			if condition.Pods != nil {
				c.Pods = make([]v1.PodCondition, 0, len(condition.Pods))
				c.Pods = append(c.Pods, condition.Pods...)
			}
		}

		target.Status.SetConditions(c)
	}

	// Mirror status replicas and selector
	target.Status.Replicas = it.Status.Replicas
	target.Status.Selector = it.Status.Selector

	return target, nil
}

func setKameletBindingReadyCondition(kb *v1alpha1.KameletBinding, it *v1.Integration) {
	if condition := it.Status.GetCondition(v1.IntegrationConditionReady); condition != nil {
		message := condition.Message
		if message == "" {
			message = fmt.Sprintf("Integration %q readiness condition is %q", it.GetName(), condition.Status)
		}

		c := v1alpha1.KameletBindingCondition{
			Type:    v1alpha1.KameletBindingConditionReady,
			Status:  condition.Status,
			Reason:  condition.Reason,
			Message: message,
		}

		if condition.Pods != nil {
			c.Pods = make([]v1.PodCondition, 0, len(condition.Pods))
			c.Pods = append(c.Pods, condition.Pods...)
		}

		kb.Status.SetConditions(c)

	} else {
		kb.Status.SetCondition(
			v1alpha1.KameletBindingConditionReady,
			corev1.ConditionUnknown,
			"",
			fmt.Sprintf("Integration %q does not have a readiness condition", it.GetName()),
		)
	}
}

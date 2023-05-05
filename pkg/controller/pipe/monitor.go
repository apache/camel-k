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

package pipe

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/trait"
)

// NewMonitorAction returns an action that monitors the Pipe after it's fully initialized.
func NewMonitorAction() Action {
	return &monitorAction{}
}

type monitorAction struct {
	baseAction
}

func (action *monitorAction) Name() string {
	return "monitor"
}

func (action *monitorAction) CanHandle(binding *v1.Pipe) bool {
	return binding.Status.Phase == v1.PipePhaseCreating ||
		(binding.Status.Phase == v1.PipePhaseError &&
			binding.Status.GetCondition(v1.PipeIntegrationConditionError) == nil) ||
		binding.Status.Phase == v1.PipePhaseReady
}

func (action *monitorAction) Handle(ctx context.Context, binding *v1.Pipe) (*v1.Pipe, error) {
	key := client.ObjectKey{
		Namespace: binding.Namespace,
		Name:      binding.Name,
	}
	it := v1.Integration{}
	if err := action.client.Get(ctx, key, &it); err != nil && k8serrors.IsNotFound(err) {
		target := binding.DeepCopy()
		// Rebuild the integration
		target.Status.Phase = v1.PipePhaseNone
		target.Status.SetCondition(
			v1.PipeConditionReady,
			corev1.ConditionFalse,
			"",
			"",
		)
		return target, nil
	} else if err != nil {
		return nil, fmt.Errorf("could not load integration for Pipe %q: %w", binding.Name, err)
	}

	operatorIDChanged := v1.GetOperatorIDAnnotation(binding) != "" &&
		(v1.GetOperatorIDAnnotation(binding) != v1.GetOperatorIDAnnotation(&it))

	sameTraits, err := trait.IntegrationAndPipeSameTraits(&it, binding)
	if err != nil {
		return nil, err
	}

	// Check if the integration needs to be changed
	expected, err := CreateIntegrationFor(ctx, action.client, binding)
	if err != nil {
		binding.Status.Phase = v1.PipePhaseError
		binding.Status.SetErrorCondition(v1.PipeIntegrationConditionError,
			"Couldn't create an Integration custom resource", err)
		return binding, err
	}

	semanticEquality := equality.Semantic.DeepDerivative(expected.Spec, it.Spec)

	if !semanticEquality || operatorIDChanged || !sameTraits {
		action.L.Info(
			"Pipe needs a rebuild",
			"semantic-equality", !semanticEquality,
			"operatorid-changed", operatorIDChanged,
			"traits-changed", !sameTraits)

		// Pipe has changed and needs rebuild
		target := binding.DeepCopy()
		// Rebuild the integration
		target.Status.Phase = v1.PipePhaseNone
		target.Status.SetCondition(
			v1.PipeConditionReady,
			corev1.ConditionFalse,
			"",
			"",
		)
		return target, nil
	}

	// Map integration phase and conditions to Pipe
	target := binding.DeepCopy()

	switch it.Status.Phase {

	case v1.IntegrationPhaseRunning:
		target.Status.Phase = v1.PipePhaseReady
		setPipeReadyCondition(target, &it)

	case v1.IntegrationPhaseError:
		target.Status.Phase = v1.PipePhaseError
		setPipeReadyCondition(target, &it)

	default:
		target.Status.Phase = v1.PipePhaseCreating

		c := v1.PipeCondition{
			Type:    v1.PipeConditionReady,
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

func setPipeReadyCondition(kb *v1.Pipe, it *v1.Integration) {
	if condition := it.Status.GetCondition(v1.IntegrationConditionReady); condition != nil {
		message := condition.Message
		if message == "" {
			message = fmt.Sprintf("Integration %q readiness condition is %q", it.GetName(), condition.Status)
		}

		c := v1.PipeCondition{
			Type:    v1.PipeConditionReady,
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
			v1.PipeConditionReady,
			corev1.ConditionUnknown,
			"",
			fmt.Sprintf("Integration %q does not have a readiness condition", it.GetName()),
		)
	}
}

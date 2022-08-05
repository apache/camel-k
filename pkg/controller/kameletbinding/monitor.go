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

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/trait"
)

// NewMonitorAction returns an action that monitors the KameletBinding after it's fully initialized.
func NewMonitorAction() Action {
	return &monitorAction{}
}

type monitorAction struct {
	baseAction
}

func (action *monitorAction) Name() string {
	return "monitor"
}

func (action *monitorAction) CanHandle(kameletbinding *v1alpha1.KameletBinding) bool {
	return kameletbinding.Status.Phase == v1alpha1.KameletBindingPhaseCreating ||
		kameletbinding.Status.Phase == v1alpha1.KameletBindingPhaseError ||
		kameletbinding.Status.Phase == v1alpha1.KameletBindingPhaseReady
}

func (action *monitorAction) Handle(ctx context.Context, kameletbinding *v1alpha1.KameletBinding) (*v1alpha1.KameletBinding, error) {
	key := client.ObjectKey{
		Namespace: kameletbinding.Namespace,
		Name:      kameletbinding.Name,
	}
	it := v1.Integration{}
	if err := action.client.Get(ctx, key, &it); err != nil && k8serrors.IsNotFound(err) {
		target := kameletbinding.DeepCopy()
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
		return nil, errors.Wrapf(err, "could not load integration for KameletBinding %q", kameletbinding.Name)
	}

	operatorIDChanged := v1.GetOperatorIDAnnotation(kameletbinding) != "" &&
		(v1.GetOperatorIDAnnotation(kameletbinding) != v1.GetOperatorIDAnnotation(&it))

	sameTraits, err := trait.IntegrationAndBindingSameTraits(&it, kameletbinding)
	if err != nil {
		return nil, err
	}

	// Check if the integration needs to be changed
	expected, err := CreateIntegrationFor(ctx, action.client, kameletbinding)
	if err != nil {
		return nil, err
	}

	semanticEquality := equality.Semantic.DeepDerivative(expected.Spec, it.Spec)

	if !semanticEquality || operatorIDChanged || !sameTraits {
		action.L.Info(
			"KameletBinding needs a rebuild",
			"semantic-equality", !semanticEquality,
			"operatorid-changed", operatorIDChanged,
			"traits-changed", !sameTraits)

		// KameletBinding has changed and needs rebuild
		target := kameletbinding.DeepCopy()
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

	// Map integration phase and conditions to KameletBinding
	target := kameletbinding.DeepCopy()

	switch it.Status.Phase {

	case v1.IntegrationPhaseRunning:
		target.Status.Phase = v1alpha1.KameletBindingPhaseReady
		setKameletBindingReadyCondition(target, &it)

	case v1.IntegrationPhaseError:
		target.Status.Phase = v1alpha1.KameletBindingPhaseError
		setKameletBindingReadyCondition(target, &it)

	default:
		target.Status.Phase = v1alpha1.KameletBindingPhaseCreating
		target.Status.SetCondition(
			v1alpha1.KameletBindingConditionReady,
			corev1.ConditionFalse,
			string(target.Status.Phase),
			fmt.Sprintf("Integration %q is in %q phase", it.GetName(), target.Status.Phase),
		)
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
		kb.Status.SetCondition(
			v1alpha1.KameletBindingConditionReady,
			condition.Status,
			condition.Reason,
			message,
		)
	} else {
		kb.Status.SetCondition(
			v1alpha1.KameletBindingConditionReady,
			corev1.ConditionUnknown,
			"",
			fmt.Sprintf("Integration %q does not have a readiness condition", it.GetName()),
		)
	}
}

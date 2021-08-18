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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewMonitorAction returns an action that monitors the kamelet binding after it's fully initialized
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

	// Check if the integration needs to be changed
	expected, err := createIntegrationFor(ctx, action.client, kameletbinding)
	if err != nil {
		return nil, err
	}

	if !equality.Semantic.DeepDerivative(expected.Spec, it.Spec) {
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

	// Map integration phases to KameletBinding phases
	target := kameletbinding.DeepCopy()
	if it.Status.Phase == v1.IntegrationPhaseRunning {
		target.Status.Phase = v1alpha1.KameletBindingPhaseReady
		target.Status.SetCondition(
			v1alpha1.KameletBindingConditionReady,
			corev1.ConditionTrue,
			"",
			"",
		)
	} else if it.Status.Phase == v1.IntegrationPhaseError {
		target.Status.Phase = v1alpha1.KameletBindingPhaseError
		target.Status.SetCondition(
			v1alpha1.KameletBindingConditionReady,
			corev1.ConditionFalse,
			string(target.Status.Phase),
			"",
		)
	} else {
		target.Status.Phase = v1alpha1.KameletBindingPhaseCreating
		target.Status.SetCondition(
			v1alpha1.KameletBindingConditionReady,
			corev1.ConditionFalse,
			string(target.Status.Phase),
			"",
		)
	}

	// Mirror status replicas and selector
	target.Status.Replicas = it.Status.Replicas
	target.Status.Selector = it.Status.Selector

	return target, nil
}

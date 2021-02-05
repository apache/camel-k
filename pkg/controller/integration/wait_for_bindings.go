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

package integration

import (
	"context"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	corev1 "k8s.io/api/core/v1"
)

// NewWaitForBindingsAction waits until Bindings are available and ready to be used by Integrations.
func NewWaitForBindingsAction() Action {
	return &waitForBindingsAction{}
}

type waitForBindingsAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *waitForBindingsAction) Name() string {
	return "wait-for-bindings"
}

// CanHandle tells whether this action can handle the integration
func (action *waitForBindingsAction) CanHandle(integration *v1.Integration) bool {
	return integration.Status.Phase == v1.IntegrationPhaseWaitingForBindings
}

// Handle handles the integrations
func (action *waitForBindingsAction) Handle(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {
	if _, err := trait.Apply(ctx, action.client, integration, nil); err != nil {
		return nil, err
	}
	// TODO: add new binding types here e.g knative binding
	bindingsCondTypes := []v1.IntegrationConditionType{v1.IntegrationConditionServiceBindingsCollectionReady}
	// let's see if bindings are ready
	for _, bindingType := range bindingsCondTypes {
		cond := integration.Status.GetCondition(bindingType)
		if cond != nil && cond.GetStatus() == corev1.ConditionFalse {
			// let's wait some more
			return integration, nil
		}
	}
	// bindings are ready
	integration.Status.Phase = v1.IntegrationPhaseInitialization
	return integration, nil
}

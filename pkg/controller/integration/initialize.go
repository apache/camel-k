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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

// NewInitializeAction creates a new initialize action.
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

// Name returns a common name of the action.
func (action *initializeAction) Name() string {
	return "initialize"
}

// CanHandle tells whether this action can handle the integration.
func (action *initializeAction) CanHandle(integration *v1.Integration) bool {
	return integration.Status.Phase == v1.IntegrationPhaseInitialization ||
		integration.Status.Phase == v1.IntegrationPhaseUnDeploying
}

// Handle handles the integrations.
func (action *initializeAction) Handle(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {
	action.L.Info("Initializing Integration")

	// Only move to the build submitted when we're initializing, never on undeploying
	if integration.Status.Phase == v1.IntegrationPhaseInitialization && integration.Spec.Git != nil {
		integration.Status.Phase = v1.IntegrationPhaseBuildSubmitted

		return integration, nil
	}

	// We must clear deployment conditions when undeploying.
	if integration.Status.Phase == v1.IntegrationPhaseUnDeploying {
		integration.Status.Replicas = nil
		integration.Status.RemoveCondition(v1.IntegrationConditionReady)
	}

	if _, err := trait.Apply(ctx, action.client, integration, nil); err != nil {
		integration.Status.Phase = v1.IntegrationPhaseError
		integration.SetReadyCondition(corev1.ConditionFalse,
			v1.IntegrationConditionInitializationFailedReason, err.Error())

		return integration, err
	}

	if integration.Status.IntegrationKit == nil {
		ikt := action.lookupIntegrationKit(integration)
		integration.SetIntegrationKit(ikt)
	}

	if integration.Status.Image != "" {
		integration.SetBuildCompletePhase()

		return integration, nil
	}

	integration.Status.Phase = v1.IntegrationPhaseBuildingKit
	integration.Status.Version = defaults.Version
	if timestamp := integration.Status.InitializationTimestamp; timestamp == nil || timestamp.IsZero() {
		// Round to second precision, as meta.Time fields are marshalled in RFC3339 format
		now := metav1.Now().Rfc3339Copy()
		integration.Status.InitializationTimestamp = &now
	}

	return integration, nil
}

func (action *initializeAction) lookupIntegrationKit(integration *v1.Integration) *v1.IntegrationKit {
	if integration.Spec.IntegrationKit == nil || integration.Spec.IntegrationKit.Name == "" {
		return nil
	}

	kitNamespace := integration.Spec.IntegrationKit.Namespace
	kitName := integration.Spec.IntegrationKit.Name

	return v1.NewIntegrationKit(kitNamespace, kitName)
}

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

	"github.com/apache/camel-k/pkg/util/kubernetes"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/digest"
)

// NewInitializeAction creates a new initialize action
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *initializeAction) Name() string {
	return "initialize"
}

// CanHandle tells whether this action can handle the integration
func (action *initializeAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseInitial || integration.Status.Phase == v1alpha1.IntegrationPhaseWaitingForPlatform
}

// Handle handles the integrations
func (action *initializeAction) Handle(ctx context.Context, integration *v1alpha1.Integration) error {
	pl, err := platform.GetCurrentPlatform(ctx, action.client, integration.Namespace)
	if err != nil {
		return err
	}

	// The integration platform needs to be ready before starting to create integrations
	if pl.Status.Phase != v1alpha1.IntegrationPlatformPhaseReady {
		action.L.Info("Waiting for the integration platform to be initialized")

		if integration.Status.Phase != v1alpha1.IntegrationPhaseWaitingForPlatform {
			target := integration.DeepCopy()
			target.Status.Phase = v1alpha1.IntegrationPhaseWaitingForPlatform

			action.L.Info("Integration state transition", "phase", target.Status.Phase)

			return action.client.Status().Update(ctx, target)
		}

		return nil
	}

	dgst, err := digest.ComputeForIntegration(integration)
	if err != nil {
		return err
	}

	//
	// restore phase to initial phase as traits are not aware of
	// WaitingForPlatform phase
	//
	if integration.Status.Phase == v1alpha1.IntegrationPhaseWaitingForPlatform {
		target := integration.DeepCopy()
		target.Status.Phase = v1alpha1.IntegrationPhaseInitial
		target.Status.Digest = dgst

		return action.client.Status().Update(ctx, target)
	}

	// better not changing the spec section of the target because it may be used for comparison by a
	// higher level controller (e.g. Knative source controller)

	target := integration.DeepCopy()

	// execute custom initialization
	env, err := trait.Apply(ctx, action.client, target, nil)
	if err != nil {
		return err
	}

	err = kubernetes.ReplaceResources(ctx, action.client, env.Resources.Items())
	if err != nil {
		return err
	}

	target.Status.Phase = v1alpha1.IntegrationPhaseBuildingKit
	target.Status.Digest = dgst
	target.Status.Kit = integration.Spec.Kit
	target.Status.Image = ""

	action.L.Info("Integration state transition", "phase", target.Status.Phase)

	return action.client.Status().Update(ctx, target)
}

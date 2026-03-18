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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/trait"
	corev1 "k8s.io/api/core/v1"
)

// NewBuildCompleteAction creates a new build complete action.
func NewBuildCompleteAction() Action {
	return &buildCompleteAction{}
}

type buildCompleteAction struct {
	baseAction
}

// Name returns a common name of the action.
func (action *buildCompleteAction) Name() string {
	return "build-complete"
}

// CanHandle tells whether this action can handle the integration.
func (action *buildCompleteAction) CanHandle(integration *v1.Integration) bool {
	return integration.Status.Phase == v1.IntegrationPhaseBuildComplete
}

// Handle handles the integrations.
func (action *buildCompleteAction) Handle(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {
	// Run traits that are enabled for the "Build Complete" phase (ie, gitops)
	_, err := trait.Apply(ctx, action.client, integration, nil)
	if err != nil {
		integration.Status.Phase = v1.IntegrationPhaseError
		integration.SetReadyCondition(
			corev1.ConditionFalse,
			v1.IntegrationConditionInitializationFailedReason,
			err.Error(),
		)

		return integration, err
	}
	if integration.Annotations[v1.IntegrationDontRunAfterBuildAnnotation] != v1.IntegrationDontRunAfterBuildAnnotationTrueValue {
		// We only move to Deploying phase if the Integration is not marked as "build only"
		integration.SetDeployingPhase()
	}

	return integration, nil
}

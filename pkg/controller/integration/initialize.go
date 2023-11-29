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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/platform"
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
	return integration.Status.Phase == v1.IntegrationPhaseInitialization
}

// Handle handles the integrations.
func (action *initializeAction) Handle(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {
	action.L.Info("Initializing Integration")

	if integration.Annotations[v1.IntegrationImportedNameLabel] != "" {
		return action.importFromExternalApp(integration)
	}

	if _, err := trait.Apply(ctx, action.client, integration, nil); err != nil {
		integration.Status.Phase = v1.IntegrationPhaseError
		integration.SetReadyCondition(corev1.ConditionFalse,
			v1.IntegrationConditionInitializationFailedReason, err.Error())
		return integration, err
	}

	if integration.Status.IntegrationKit == nil {
		if integration.Spec.IntegrationKit != nil && integration.Spec.IntegrationKit.Name != "" {
			kitNamespace := integration.Spec.IntegrationKit.Namespace
			kitName := integration.Spec.IntegrationKit.Name

			if kitNamespace == "" {
				pl, err := platform.GetForResource(ctx, action.client, integration)
				if err != nil && !k8serrors.IsNotFound(err) {
					return nil, err
				}
				if pl != nil {
					kitNamespace = pl.Namespace
				}
			}
			kit := v1.NewIntegrationKit(kitNamespace, kitName)
			integration.SetIntegrationKit(kit)
		} else {
			integration.Status.IntegrationKit = nil
		}
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

func (action *initializeAction) importFromExternalApp(integration *v1.Integration) (*v1.Integration, error) {
	readyMessage := fmt.Sprintf(
		"imported from %s %s",
		integration.Annotations[v1.IntegrationImportedNameLabel],
		integration.Annotations[v1.IntegrationImportedKindLabel],
	)
	// We need to set the condition for which this Integration is imported (required later by monitoring)
	integration.Status.SetConditions(
		getCamelAppImportingCondition(
			integration.Annotations[v1.IntegrationImportedKindLabel],
			readyMessage,
		)...,
	)
	// If it's ready, then we can safely assume the integration is running
	if integration.IsConditionTrue(v1.IntegrationConditionReady) {
		integration.Status.Phase = v1.IntegrationPhaseRunning
	} else {
		integration.Status.Phase = v1.IntegrationPhaseError
	}

	return integration, nil
}

func getCamelAppImportingCondition(kind, message string) []v1.IntegrationCondition {
	switch kind {
	case "Deployment":
		return []v1.IntegrationCondition{
			{
				Type:    v1.IntegrationConditionDeploymentAvailable,
				Status:  corev1.ConditionTrue,
				Reason:  v1.IntegrationConditionDeploymentAvailableReason,
				Message: message,
			},
			{
				Type:    v1.IntegrationConditionReady,
				Status:  corev1.ConditionTrue,
				Reason:  v1.IntegrationConditionDeploymentReadyReason,
				Message: message,
			},
		}
	case "CronJob":
		return []v1.IntegrationCondition{
			{
				Type:    v1.IntegrationConditionCronJobAvailable,
				Status:  corev1.ConditionTrue,
				Reason:  v1.IntegrationConditionCronJobCreatedReason,
				Message: message,
			},
			{
				Type:    v1.IntegrationConditionReady,
				Status:  corev1.ConditionTrue,
				Reason:  v1.IntegrationConditionDeploymentReadyReason,
				Message: message,
			},
		}
	case "KnativeService":
		return []v1.IntegrationCondition{
			{
				Type:    v1.IntegrationConditionKnativeServiceAvailable,
				Status:  corev1.ConditionTrue,
				Reason:  v1.IntegrationConditionKnativeServiceAvailableReason,
				Message: message,
			},
			{
				Type:    v1.IntegrationConditionReady,
				Status:  corev1.ConditionTrue,
				Reason:  v1.IntegrationConditionKnativeServiceReadyReason,
				Message: message,
			},
		}
	default:
		return []v1.IntegrationCondition{
			{
				Type:    v1.IntegrationConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  v1.IntegrationConditionImportingKindAvailableReason,
				Message: fmt.Sprintf("Unsupported %s import kind", kind),
			},
		}
	}
}

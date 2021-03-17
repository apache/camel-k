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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/defaults"
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
func (action *initializeAction) CanHandle(integration *v1.Integration) bool {
	return integration.Status.Phase == v1.IntegrationPhaseInitialization
}

// Handle handles the integrations
func (action *initializeAction) Handle(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {
	if _, err := trait.Apply(ctx, action.client, integration, nil); err != nil {
		return nil, err
	}

	if integration.Spec.IntegrationKit == nil && integration.Spec.Kit != "" {
		// TODO: temporary fallback until deprecated field gets removed
		integration.Spec.IntegrationKit = &corev1.ObjectReference{
			Name: integration.Spec.Kit,
		}
	}

	if integration.Spec.IntegrationKit != nil && integration.Spec.IntegrationKit.Name != "" {
		kitNamespace := integration.Spec.IntegrationKit.Namespace
		kitName := integration.Spec.IntegrationKit.Name

		if kitNamespace == "" {
			pl, err := platform.GetCurrent(ctx, action.client, integration.Namespace)
			if err != nil && !k8serrors.IsNotFound(err) {
				return nil, err
			}
			if pl != nil {
				kitNamespace = pl.Namespace
			}
		}
		kit := v1.NewIntegrationKit(kitNamespace, kitName)
		integration.SetIntegrationKit(&kit)
	} else {
		integration.Status.IntegrationKit = nil
	}

	integration.Status.Phase = v1.IntegrationPhaseBuildingKit
	integration.Status.Version = defaults.Version
	if timestamp := integration.Status.InitializationTimestamp; timestamp == nil || timestamp.IsZero() {
		now := metav1.Now()
		integration.Status.InitializationTimestamp = &now
	}

	return integration, nil
}

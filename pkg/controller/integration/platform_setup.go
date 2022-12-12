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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/knative"
)

// NewPlatformSetupAction creates a new platform-setup action.
func NewPlatformSetupAction() Action {
	return &platformSetupAction{}
}

type platformSetupAction struct {
	baseAction
}

// Name returns a common name of the action.
func (action *platformSetupAction) Name() string {
	return "platform-setup"
}

// CanHandle tells whether this action can handle the integration.
func (action *platformSetupAction) CanHandle(integration *v1.Integration) bool {
	return integration.Status.Phase == v1.IntegrationPhaseNone ||
		integration.Status.Phase == v1.IntegrationPhaseWaitingForPlatform
}

// Handle handles the integrations.
func (action *platformSetupAction) Handle(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {
	if _, err := trait.Apply(ctx, action.client, integration, nil); err != nil {
		integration.Status.Phase = v1.IntegrationPhaseError
		integration.SetReadyCondition(corev1.ConditionFalse,
			v1.IntegrationConditionInitializationFailedReason, err.Error())
		return integration, err
	}

	pl, err := platform.GetForResource(ctx, action.client, integration)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	} else if pl != nil {
		profile, err := determineBestProfile(action.client, integration, pl)
		if err != nil {
			return nil, err
		}
		integration.Status.Profile = profile
	}

	// Change the integration phase to Initialization after traits have been applied
	// so that traits targeting Initialization phase don't get applied unintentionally
	// at the platform setup step.
	if integration.Status.Phase != v1.IntegrationPhaseWaitingForPlatform {
		integration.Status.Phase = v1.IntegrationPhaseInitialization
	}

	return integration, nil
}

// DetermineBestProfile tries to detect the best trait profile for the integration.
func determineBestProfile(c client.Client, integration *v1.Integration, p *v1.IntegrationPlatform) (v1.TraitProfile, error) {
	if integration.Spec.Profile != "" {
		return integration.Spec.Profile, nil
	}
	if integration.Status.Profile != "" {
		// Integration already has a profile
		return integration.Status.Profile, nil
	}
	if p.Status.Profile != "" {
		// Use platform profile if set
		return p.Status.Profile, nil
	}
	if p.Spec.Profile != "" {
		// Use platform spec profile if set
		return p.Spec.Profile, nil
	}
	if ok, err := knative.IsInstalled(c); err != nil {
		return "", err
	} else if ok {
		return v1.TraitProfileKnative, nil
	}
	return platform.GetProfile(p), nil
}

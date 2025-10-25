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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util/digest"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func newBuildKitAction() Action {
	return &buildKitAction{}
}

type buildKitAction struct {
	baseAction
}

func (action *buildKitAction) Name() string {
	return "build-kit"
}

func (action *buildKitAction) CanHandle(integration *v1.Integration) bool {
	return integration.Status.Phase == v1.IntegrationPhaseBuildingKit
}

func (action *buildKitAction) Handle(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {
	// TODO: we may need to add a timeout strategy, i.e give up after some time in case of an unrecoverable error.
	secrets, configmaps := getIntegrationSecretAndConfigmapResourceVersions(ctx, action.client, integration)
	hash, err := digest.ComputeForIntegration(integration, configmaps, secrets)
	if err != nil {
		return nil, err
	}
	if hash != integration.Status.Digest {
		action.L.Info("Integration %s digest has changed: resetting its status. Will check if it needs to be rebuilt and restarted.", integration.Name)
		integration.Initialize()
		integration.Status.Digest = hash
		return integration, nil
	}

	//
	// IntegrationKit may be nil if its being upgraded
	//
	if integration.Status.IntegrationKit != nil {
		it, err := action.checkIntegrationKit(ctx, integration)
		if err != nil {
			return nil, err
		}

		return it, nil
	}

	action.L.Debug("No kit specified in integration status so looking up", "integration", integration.Name, "namespace", integration.Namespace)
	existingKits, err := lookupKitsForIntegration(ctx, action.client, integration)
	if err != nil {
		err = fmt.Errorf("failed to lookup kits for integration %s/%s: %w", integration.Namespace, integration.Name, err)
		integration.Status.Phase = v1.IntegrationPhaseError
		integration.SetReadyConditionError(err.Error())
		return integration, err
	}

	action.L.Debug("Applying traits to integration",
		"integration", integration.Name,
		"namespace", integration.Namespace)
	env, err := trait.Apply(ctx, action.client, integration, nil)
	if err != nil {
		err = fmt.Errorf("failed to apply traits to integration %s/%s: %w", integration.Namespace, integration.Name, err)
		integration.Status.Phase = v1.IntegrationPhaseError
		integration.SetReadyConditionError(err.Error())
		return integration, err
	}

	action.L.Debug("Searching integration kits to assign to integration", "integration",
		integration.Name, "namespace", integration.Namespace)
	var integrationKit *v1.IntegrationKit
kits:
	for j := range env.IntegrationKits {
		kit := env.IntegrationKits[j]
		for i := range existingKits {
			k := &existingKits[i]

			action.L.Debug("Comparing existing kit with environment", "env kit", kit.Name, "existing kit", k.Name)
			match, err := kitMatches(action.client, &kit, k)
			if err != nil {
				err = fmt.Errorf("error occurred matches integration kits with environment for integration %s/%s: %w", integration.Namespace, integration.Name, err)
				integration.Status.Phase = v1.IntegrationPhaseError
				integration.SetReadyConditionError(err.Error())
				return integration, err
			}
			if match {
				if integrationKit == nil ||
					integrationKit.Status.Phase != v1.IntegrationKitPhaseReady && k.Status.Phase == v1.IntegrationKitPhaseReady ||
					integrationKit.Status.Phase == v1.IntegrationKitPhaseReady && k.Status.Phase == v1.IntegrationKitPhaseReady && k.HasHigherPriorityThan(integrationKit) {
					integrationKit = k
					action.L.Debug("Found matching kit", "integration kit", integrationKit.Name)
				}

				continue kits
			} else {
				action.L.Debug("Cannot match kits", "env kit", kit.Name, "existing kit", k.Name)
			}
		}

		action.L.Debug("No existing kit available for integration. Creating a new one.",
			"integration", integration.Name,
			"namespace", integration.Namespace,
			"integration kit", kit.Name)
		if err := action.client.Create(ctx, &kit); err != nil {
			err = fmt.Errorf("failed to create new integration kit for integration %s/%s: %w", integration.Namespace, integration.Name, err)
			integration.Status.Phase = v1.IntegrationPhaseError
			integration.SetReadyConditionError(err.Error())
			return integration, err
		}
		if integrationKit == nil {
			integrationKit = &kit
		}
	}

	//nolint:nestif
	if integrationKit != nil {
		action.L.Debug("Setting integration kit for integration", "integration", integration.Name, "namespace", integration.Namespace, "integration kit", integrationKit.Name)
		// Set the kit name so the next handle loop, will fall through the
		// same path as integration with a user defined kit
		integration.SetIntegrationKit(integrationKit)
		if integrationKit.Status.Phase == v1.IntegrationKitPhaseReady {
			if integration.Annotations[v1.IntegrationDontRunAfterBuildAnnotation] == v1.IntegrationDontRunAfterBuildAnnotationTrueValue {
				integration.Status.Phase = v1.IntegrationPhaseBuildComplete
			} else {
				integration.Status.Phase = v1.IntegrationPhaseDeploying
			}
		}
	} else {
		action.L.Debug("Not yet able to assign an integration kit to integration",
			"integration", integration.Name,
			"namespace", integration.Namespace)
	}

	return integration, nil
}

func (action *buildKitAction) checkIntegrationKit(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {

	// IntegrationKit fully defined so find it
	action.L.Debugf("Finding integration kit %s for integration %s\n",
		integration.Status.IntegrationKit.Name, integration.Name)

	kit, err := kubernetes.GetIntegrationKit(ctx, action.client,
		integration.Status.IntegrationKit.Name, integration.Status.IntegrationKit.Namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to find integration kit %s/%s: %w",
			integration.Status.IntegrationKit.Namespace, integration.Status.IntegrationKit.Name, err)
	}

	if kit.Labels[v1.IntegrationKitTypeLabel] == v1.IntegrationKitTypePlatform {
		match, err := integrationMatches(ctx, action.client, integration, kit)
		if err != nil {
			return nil, fmt.Errorf("unable to match any integration kit with integration %s/%s: %w",
				integration.Namespace, integration.Name, err)

		}

		if !match {
			// We need to re-generate a kit, or search for a new one that
			// matches the integration, so let's remove the association
			// with the kit.

			//
			// All tests & conditionals check for a nil assignment
			//
			action.L.Debug("No match found between integration and integrationkit. Resetting integration's integrationkit to empty",
				"integration", integration.Name,
				"integrationkit", integration.Status.IntegrationKit.Name,
				"namespace", integration.Namespace)

			integration.SetIntegrationKit(nil)

			return integration, nil
		}
	}

	if kit.Status.Phase == v1.IntegrationKitPhaseError {
		integration.Status.Phase = v1.IntegrationPhaseError
		integration.SetIntegrationKit(kit)
		return integration, nil
	}

	if kit.Status.Phase == v1.IntegrationKitPhaseReady {
		if integration.Annotations[v1.IntegrationDontRunAfterBuildAnnotation] == v1.IntegrationDontRunAfterBuildAnnotationTrueValue {
			integration.Status.Phase = v1.IntegrationPhaseBuildComplete
		} else {
			integration.Status.Phase = v1.IntegrationPhaseDeploying
		}
		integration.SetIntegrationKit(kit)
		return integration, nil
	}

	return nil, nil
}

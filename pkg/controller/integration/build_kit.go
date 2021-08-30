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

	"github.com/pkg/errors"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
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

	if integration.Status.IntegrationKit != nil {
		kit, err := kubernetes.GetIntegrationKit(ctx, action.client, integration.Status.IntegrationKit.Name, integration.Status.IntegrationKit.Namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to find integration kit %s/%s, %s", integration.Status.IntegrationKit.Namespace, integration.Status.IntegrationKit.Name, err)
		}

		if kit.Labels[v1.IntegrationKitTypeLabel] == v1.IntegrationKitTypePlatform {
			match, err := integrationMatches(integration, kit)
			if err != nil {
				return nil, err
			} else if !match {
				// We need to re-generate a kit, or search for a new one that
				// matches the integration, so let's remove the association
				// with the kit.
				integration.SetIntegrationKit(&v1.IntegrationKit{})
				return integration, nil
			}
		}

		if kit.Status.Phase == v1.IntegrationKitPhaseError {
			integration.Status.Phase = v1.IntegrationPhaseError
			integration.SetIntegrationKit(kit)
			return integration, nil
		}

		if kit.Status.Phase == v1.IntegrationKitPhaseReady {
			integration.Status.Phase = v1.IntegrationPhaseDeploying
			integration.SetIntegrationKit(kit)
			return integration, nil
		}

		return nil, nil
	}

	existingKits, err := lookupKitsForIntegration(ctx, action.client, integration)
	if err != nil {
		return nil, err
	}

	env, err := trait.Apply(ctx, action.client, integration, nil)
	if err != nil {
		return nil, err
	}

	var integrationKit *v1.IntegrationKit
kits:
	for _, kit := range env.IntegrationKits {
		kit := kit
		for i, k := range existingKits {
			match, err := action.kitMatches(&kit, &k)
			if err != nil {
				return nil, err
			}
			if match {
				if integrationKit == nil ||
					integrationKit.Status.Phase != v1.IntegrationKitPhaseReady && k.Status.Phase == v1.IntegrationKitPhaseReady ||
					integrationKit.Status.Phase == v1.IntegrationKitPhaseReady && k.Status.Phase == v1.IntegrationKitPhaseReady && k.HasHigherPriorityThan(integrationKit) {
					integrationKit = &existingKits[i]
				}
				continue kits
			}
		}
		if err := action.client.Create(ctx, &kit); err != nil {
			return nil, err
		}
		if integrationKit == nil {
			integrationKit = &kit
		}
	}

	// Set the kit name so the next handle loop, will fall through the
	// same path as integration with a user defined kit
	integration.SetIntegrationKit(integrationKit)

	if integrationKit.Status.Phase == v1.IntegrationKitPhaseReady {
		integration.Status.Phase = v1.IntegrationPhaseDeploying
	}

	return integration, nil
}


// kitMatches returns whether the v1.IntegrationKit match
func (action *buildKitAction) kitMatches(k1 *v1.IntegrationKit, k2 *v1.IntegrationKit) (bool, error) {
	version := k1.Status.Version
	if version == "" {
		// Defaults with the version that is going to be set during the kit initialization
		version = defaults.Version
	}
	if version != k2.Status.Version {
		return false, nil
	}
	if len(k1.Spec.Dependencies) != len(k2.Spec.Dependencies) {
		return false, nil
	}
	if len(k1.Spec.Traits) != len(k2.Spec.Traits) {
		return false, nil
	}
	for name, kt1 := range k1.Spec.Traits {
		kt2, ok := k2.Spec.Traits[name]
		if !ok {
			return false, nil
		}
		match, err := hasMatchingTrait(&kt1, &kt2)
		if !match || err != nil {
			return false, err
		}
	}
	if !util.StringSliceContains(k1.Spec.Dependencies, k2.Spec.Dependencies) {
		return false, nil
	}
	return true, nil
}

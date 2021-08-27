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
	"encoding/json"
	"reflect"

	"github.com/pkg/errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/platform"
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
			match, err := action.integrationMatches(integration, kit)
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

	existingKits, err := action.lookupKitsForIntegration(ctx, action.client, integration)
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

func (action *buildKitAction) lookupKitsForIntegration(ctx context.Context, c ctrl.Reader, integration *v1.Integration) ([]v1.IntegrationKit, error) {
	pl, err := platform.GetCurrent(ctx, c, integration.Namespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}

	kitTypes, err := labels.NewRequirement(v1.IntegrationKitTypeLabel, selection.In, []string{
		v1.IntegrationKitTypePlatform,
		v1.IntegrationKitTypeExternal,
	})
	if err != nil {
		return nil, err
	}

	options := []ctrl.ListOption{
		ctrl.InNamespace(integration.GetIntegrationKitNamespace(pl)),
		ctrl.MatchingLabels{
			"camel.apache.org/runtime.version":  integration.Status.RuntimeVersion,
			"camel.apache.org/runtime.provider": string(integration.Status.RuntimeProvider),
		},
		ctrl.MatchingLabelsSelector{
			Selector: labels.NewSelector().Add(*kitTypes),
		},
	}

	list := v1.NewIntegrationKitList()
	if err := c.List(ctx, &list, options...); err != nil {
		return nil, err
	}

	kits := make([]v1.IntegrationKit, 0)
	for _, kit := range list.Items {
		match, err := action.integrationMatches(integration, &kit)
		if err != nil {
			return nil, err
		} else if !match {
			continue
		}
		kits = append(kits, kit)
	}

	return kits, nil
}

// integrationMatches returns whether the v1.IntegrationKit meets the requirements of the v1.Integration
func (action *buildKitAction) integrationMatches(integration *v1.Integration, kit *v1.IntegrationKit) (bool, error) {
	if kit.Status.Phase == v1.IntegrationKitPhaseError {
		return false, nil
	}
	if kit.Status.Version != integration.Status.Version {
		return false, nil
	}
	if len(integration.Status.Dependencies) != len(kit.Spec.Dependencies) {
		return false, nil
	}
	// When a platform kit is created it inherits the traits from the integrations and as
	// some traits may influence the build thus the artifacts present on the container image,
	// we need to take traits into account when looking up for compatible kits.
	//
	// It could also happen that an integration is updated and a trait is modified, if we do
	// not include traits in the lookup, we may use a kit that does not have all the
	// characteristics required by the integration.
	//
	// A kit can be used only if it contains a subset of the traits and related configurations
	// declared on integration.
	if match, err := action.hasMatchingTraits(integration, kit); !match || err != nil {
		return false, err
	}
	if !util.StringSliceContains(kit.Spec.Dependencies, integration.Status.Dependencies) {
		return false, nil
	}
	return true, nil
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
		match, err := action.hasMatchingTrait(&kt1, &kt2)
		if !match || err != nil {
			return false, err
		}
	}
	if !util.StringSliceContains(k1.Spec.Dependencies, k2.Spec.Dependencies) {
		return false, nil
	}
	return true, nil
}

// hasMatchingTraits compares the traits defined on the v1.Integration with those defined on the v1.IntegrationKit
func (action *buildKitAction) hasMatchingTraits(integration *v1.Integration, kit *v1.IntegrationKit) (bool, error) {
	catalog := trait.NewCatalog(action.client)

	traitCount := 0
	for name, itTrait := range integration.Spec.Traits {
		t := catalog.GetTrait(name)
		if t != nil && !t.InfluencesKit() {
			// We don't store the trait configuration if the trait cannot influence the kit behavior
			continue
		}
		traitCount++
		kitTrait, ok := kit.Spec.Traits[name]
		if !ok {
			// skip it because trait configured on integration is not defined on kit
			return false, nil
		}
		if ct, ok := t.(trait.ComparableTrait); ok {
			comparable, err := action.hasComparableTrait(ct, &itTrait, &kitTrait)
			if !comparable || err != nil {
				return false, err
			}
		} else {
			match, err := action.hasMatchingTrait(&itTrait, &kitTrait)
			if !match || err != nil {
				return false, err
			}
		}
	}

	// Check the number of influencing traits matches
	if len(kit.Spec.Traits) != traitCount {
		return false, nil
	}

	return true, nil
}

func (action *buildKitAction) hasComparableTrait(c trait.ComparableTrait, itTrait *v1.TraitSpec, kitTrait *v1.TraitSpec) (bool, error) {
	it := reflect.New(reflect.TypeOf(c).Elem()).Interface()
	data, err := json.Marshal(itTrait.Configuration)
	if err != nil {
		return false, err
	}
	err = json.Unmarshal(data, &it)
	if err != nil {
		return false, err
	}

	kt := reflect.New(reflect.TypeOf(c).Elem()).Interface()
	data, err = json.Marshal(kitTrait.Configuration)
	if err != nil {
		return false, err
	}
	err = json.Unmarshal(data, &it)
	if err != nil {
		return false, err
	}

	return kt.(trait.ComparableTrait).Matches(it.(trait.Trait)), nil
}

func (action *buildKitAction) hasMatchingTrait(itTrait *v1.TraitSpec, kitTrait *v1.TraitSpec) (bool, error) {
	data, err := json.Marshal(itTrait.Configuration)
	if err != nil {
		return false, err
	}
	itConf := make(map[string]interface{})
	err = json.Unmarshal(data, &itConf)
	if err != nil {
		return false, err
	}
	data, err = json.Marshal(kitTrait.Configuration)
	if err != nil {
		return false, err
	}
	kitConf := make(map[string]interface{})
	err = json.Unmarshal(data, &kitConf)
	if err != nil {
		return false, err
	}
	for ck, cv := range kitConf {
		iv, ok := itConf[ck]
		if !ok {
			// skip it because trait configured on kit has a value that is not defined
			// in integration trait
			return false, nil
		}
		if !equal(iv, cv) {
			// skip it because trait configured on kit has a value that differs from
			// the one configured on integration
			return false, nil
		}
	}
	return true, nil
}

// We need to try to perform a slice equality in order to prevent a runtime panic
func equal(a, b interface{}) bool {
	aSlice, aOk := a.([]interface{})
	bSlice, bOk := b.([]interface{})

	if aOk && bOk {
		// Both are slices
		return sliceEqual(aSlice, bSlice)
	}

	if aOk || bOk {
		// One of the 2 is a slice
		return false
	}

	// None is a slice
	return a == b
}

func sliceEqual(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

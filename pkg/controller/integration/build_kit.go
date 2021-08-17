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
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/xid"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
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
	return integration.Status.Phase == v1.IntegrationPhaseBuildingKit ||
		integration.Status.Phase == v1.IntegrationPhaseResolvingKit
}

func (action *buildKitAction) Handle(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {
	kit, err := action.lookupKitForIntegration(ctx, action.client, integration)
	if err != nil {
		// TODO: we may need to add a wait strategy, i.e give up after some time
		return nil, err
	}

	if kit != nil {
		if kit.Labels[v1.IntegrationKitTypeLabel] == v1.IntegrationKitTypePlatform {
			// This is a platform kit and as it is auto generated it may get
			// out of sync if the integration that has generated it, has been
			// amended to add/remove dependencies

			versionMatch := kit.Status.Version == integration.Status.Version

			// TODO: this is a very simple check, we may need to provide a deps comparison strategy
			dependenciesMatch := util.StringSliceContains(kit.Spec.Dependencies, integration.Status.Dependencies)

			if !dependenciesMatch || !versionMatch {
				// We need to re-generate a kit or search for a new one that
				// satisfies integrations needs so let's remove the association
				// with a kit
				integration.SetIntegrationKit(&v1.IntegrationKit{})

				return integration, nil
			}
		}

		if kit.Status.Phase == v1.IntegrationKitPhaseError {
			integration.Status.Image = kit.Status.Image
			integration.Status.Phase = v1.IntegrationPhaseError
			integration.SetIntegrationKit(kit)

			return integration, nil
		}

		if kit.Status.Phase == v1.IntegrationKitPhaseReady {
			integration.Status.Image = kit.Status.Image
			integration.SetIntegrationKit(kit)

			if _, err := trait.Apply(ctx, action.client, integration, kit); err != nil {
				return nil, err
			}

			return integration, nil
		}

		if integration.Status.IntegrationKit == nil || integration.Status.IntegrationKit.Name == "" {
			integration.SetIntegrationKit(kit)

			return integration, nil
		}

		return nil, nil
	}

	pl, err := platform.GetCurrent(ctx, action.client, integration.Namespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}

	kit = v1.NewIntegrationKit(integration.GetIntegrationKitNamespace(pl), fmt.Sprintf("kit-%s", xid.New()))

	// Add some information for post-processing, this may need to be refactored
	// to a proper data structure
	kit.Labels = map[string]string{
		v1.IntegrationKitTypeLabel:            v1.IntegrationKitTypePlatform,
		"camel.apache.org/runtime.version":    integration.Status.RuntimeVersion,
		"camel.apache.org/runtime.provider":   string(integration.Status.RuntimeProvider),
		kubernetes.CamelCreatorLabelKind:      v1.IntegrationKind,
		kubernetes.CamelCreatorLabelName:      integration.Name,
		kubernetes.CamelCreatorLabelNamespace: integration.Namespace,
		kubernetes.CamelCreatorLabelVersion:   integration.ResourceVersion,
	}

	// Set the kit to have the same characteristics as the integrations
	kit.Spec = v1.IntegrationKitSpec{
		Dependencies: integration.Status.Dependencies,
		Repositories: integration.Spec.Repositories,
		Traits:       action.filterKitTraits(ctx, integration.Spec.Traits),
	}

	if _, err := trait.Apply(ctx, action.client, integration, kit); err != nil {
		return nil, err
	}

	if err := action.client.Create(ctx, kit); err != nil {
		return nil, err
	}

	// Set the kit name so the next handle loop, will fall through the
	// same path as integration with a user defined kit
	integration.SetIntegrationKit(kit)

	return integration, nil
}

func (action *buildKitAction) filterKitTraits(ctx context.Context, in map[string]v1.TraitSpec) map[string]v1.TraitSpec {
	if len(in) == 0 {
		return in
	}
	catalog := trait.NewCatalog(ctx, action.client)
	out := make(map[string]v1.TraitSpec)
	for name, conf := range in {
		t := catalog.GetTrait(name)
		if t != nil && !t.InfluencesKit() {
			// We don't store the trait configuration if the trait cannot influence the kit behavior
			continue
		}
		out[name] = conf
	}
	return out
}

func (action *buildKitAction) lookupKitForIntegration(ctx context.Context, c ctrl.Reader, integration *v1.Integration) (*v1.IntegrationKit, error) {
	if integration.Status.IntegrationKit != nil {
		kit, err := kubernetes.GetIntegrationKit(ctx, c, integration.Status.IntegrationKit.Name, integration.Status.IntegrationKit.Namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to find integration kit %s/%s, %s", integration.Status.IntegrationKit.Namespace, integration.Status.IntegrationKit.Name, err)
		}

		return kit, nil
	}

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

	kits := v1.NewIntegrationKitList()
	if err := c.List(ctx, &kits, options...); err != nil {
		return nil, err
	}

	for _, kit := range kits.Items {
		kit := kit // pin

		if kit.Status.Phase == v1.IntegrationKitPhaseError {
			continue
		}

		/*
			TODO: moved to label selector
			if kit.Status.RuntimeVersion != integration.Status.RuntimeVersion {
				continue
			}
			if kit.Status.RuntimeProvider != integration.Status.RuntimeProvider {
				continue
			}
		*/

		if kit.Status.Version != integration.Status.Version {
			continue
		}

		ideps := len(integration.Status.Dependencies)
		cdeps := len(kit.Spec.Dependencies)

		if ideps != cdeps {
			continue
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
		match, err := action.hasMatchingTraits(ctx, &kit, integration)
		if err != nil {
			return nil, err
		}
		if !match {
			continue
		}
		if util.StringSliceContains(kit.Spec.Dependencies, integration.Status.Dependencies) {
			return &kit, nil
		}
	}

	return nil, nil
}

// hasMatchingTraits compares traits defined on kit against those defined on integration
func (action *buildKitAction) hasMatchingTraits(ctx context.Context, kit *v1.IntegrationKit, integration *v1.Integration) (bool, error) {
	traits := action.filterKitTraits(ctx, integration.Spec.Traits)

	// The kit has no trait, but the integration need some
	if len(kit.Spec.Traits) == 0 && len(traits) > 0 {
		return false, nil
	}
	for name, kitTrait := range kit.Spec.Traits {
		itTrait, ok := traits[name]
		if !ok {
			// skip it because trait configured on kit is not defined on integration
			return false, nil
		}
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

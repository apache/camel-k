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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/defaults"
)

func lookupKitsForIntegration(ctx context.Context, c ctrl.Reader, integration *v1.Integration, options ...ctrl.ListOption) ([]v1.IntegrationKit, error) {
	pl, err := platform.GetCurrent(ctx, c, integration.Namespace)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	kitTypes, err := labels.NewRequirement(v1.IntegrationKitTypeLabel, selection.In, []string{
		v1.IntegrationKitTypePlatform,
		v1.IntegrationKitTypeExternal,
	})
	if err != nil {
		return nil, err
	}

	listOptions := []ctrl.ListOption{
		ctrl.InNamespace(integration.GetIntegrationKitNamespace(pl)),
		ctrl.MatchingLabels{
			"camel.apache.org/runtime.version":  integration.Status.RuntimeVersion,
			"camel.apache.org/runtime.provider": string(integration.Status.RuntimeProvider),
		},
		ctrl.MatchingLabelsSelector{
			Selector: labels.NewSelector().Add(*kitTypes),
		},
	}
	listOptions = append(listOptions, options...)

	list := v1.NewIntegrationKitList()
	if err := c.List(ctx, &list, listOptions...); err != nil {
		return nil, err
	}

	kits := make([]v1.IntegrationKit, 0)
	for _, kit := range list.Items {
		match, err := integrationMatches(integration, &kit)
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
func integrationMatches(integration *v1.Integration, kit *v1.IntegrationKit) (bool, error) {
	if kit.Status.Phase == v1.IntegrationKitPhaseError {
		return false, nil
	}
	if kit.Status.Version != integration.Status.Version {
		return false, nil
	}
	if kit.Status.RuntimeProvider != integration.Status.RuntimeProvider {
		return false, nil
	}
	if kit.Status.RuntimeVersion != integration.Status.RuntimeVersion {
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
	if match, err := hasMatchingTraits(integration.Spec.Traits, kit.Spec.Traits); !match || err != nil {
		return false, err
	}
	if !util.StringSliceContains(kit.Spec.Dependencies, integration.Status.Dependencies) {
		return false, nil
	}
	return true, nil
}

// kitMatches returns whether the two v1.IntegrationKit match
func kitMatches(kit1 *v1.IntegrationKit, kit2 *v1.IntegrationKit) (bool, error) {
	version := kit1.Status.Version
	if version == "" {
		// Defaults with the version that is going to be set during the kit initialization
		version = defaults.Version
	}
	if version != kit2.Status.Version {
		return false, nil
	}
	if len(kit1.Spec.Dependencies) != len(kit2.Spec.Dependencies) {
		return false, nil
	}
	if match, err := hasMatchingTraits(kit1.Spec.Traits, kit2.Spec.Traits); !match || err != nil {
		return false, err
	}
	if !util.StringSliceContains(kit1.Spec.Dependencies, kit2.Spec.Dependencies) {
		return false, nil
	}
	return true, nil
}

func hasMatchingTraits(traits1 map[string]v1.TraitSpec, traits2 map[string]v1.TraitSpec) (bool, error) {
	catalog := trait.NewCatalog(nil)
	for _, t := range catalog.AllTraits() {
		if t != nil && !t.InfluencesKit() {
			// We don't store the trait configuration if the trait cannot influence the kit behavior
			continue
		}
		id := string(t.ID())
		t1, ok1 := traits1[id]
		t2, ok2 := traits2[id]

		if !ok1 && !ok2 {
			continue
		}
		if ct, ok := t.(trait.ComparableTrait); ok {
			if comparable, err := hasComparableTrait(ct, &t1, &t2); err != nil {
				return false, err
			} else if comparable {
				continue
			}
		} else if ok1 && ok2 {
			if match, err := hasMatchingTrait(&t1, &t2); err != nil {
				return false, err
			} else if match {
				continue
			}
		}
		return false, nil
	}

	return true, nil
}

func hasComparableTrait(t trait.ComparableTrait, ts1 *v1.TraitSpec, ts2 *v1.TraitSpec) (bool, error) {
	t1 := reflect.New(reflect.TypeOf(t).Elem()).Interface()
	if ts1.Configuration.RawMessage != nil {
		data, err := json.Marshal(ts1.Configuration)
		if err != nil {
			return false, err
		}
		err = json.Unmarshal(data, &t1)
		if err != nil {
			return false, err
		}
	}

	t2 := reflect.New(reflect.TypeOf(t).Elem()).Interface()
	if ts2.Configuration.RawMessage != nil {
		data, err := json.Marshal(ts2.Configuration)
		if err != nil {
			return false, err
		}
		err = json.Unmarshal(data, &t2)
		if err != nil {
			return false, err
		}
	}

	return t2.(trait.ComparableTrait).Matches(t1.(trait.Trait)), nil
}

func hasMatchingTrait(ts1 *v1.TraitSpec, ts2 *v1.TraitSpec) (bool, error) {
	data, err := json.Marshal(ts1.Configuration)
	if err != nil {
		return false, err
	}
	t1 := make(map[string]interface{})
	err = json.Unmarshal(data, &t1)
	if err != nil {
		return false, err
	}
	data, err = json.Marshal(ts2.Configuration)
	if err != nil {
		return false, err
	}
	t2 := make(map[string]interface{})
	err = json.Unmarshal(data, &t2)
	if err != nil {
		return false, err
	}
	for ck, cv := range t2 {
		iv, ok := t1[ck]
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

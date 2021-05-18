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

	"github.com/apache/camel-k/pkg/platform"
	"github.com/pkg/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/selection"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/controller"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

// LookupKitForIntegration --
func LookupKitForIntegration(ctx context.Context, c k8sclient.Reader, integration *v1.Integration) (*v1.IntegrationKit, error) {
	if integration.Status.IntegrationKit != nil {
		kit, err := kubernetes.GetIntegrationKit(ctx, c, integration.Status.IntegrationKit.Name, integration.Status.IntegrationKit.Namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to find integration kit %s/%s, %s", integration.Status.IntegrationKit.Namespace, integration.Status.IntegrationKit.Name, err)
		}

		return kit, nil
	}

	pl, err := platform.GetCurrent(ctx, c, integration.Namespace)
	if err != nil && !k8errors.IsNotFound(err) {
		return nil, err
	}

	options := []k8sclient.ListOption{
		k8sclient.InNamespace(integration.GetIntegrationKitNamespace(pl)),
		k8sclient.MatchingLabels{
			"camel.apache.org/runtime.version":  integration.Status.RuntimeVersion,
			"camel.apache.org/runtime.provider": string(integration.Status.RuntimeProvider),
		},
		controller.NewLabelSelector("camel.apache.org/kit.type", selection.In, []string{
			v1.IntegrationKitTypePlatform,
			v1.IntegrationKitTypeExternal,
		}),
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
		match, err := HasMatchingTraits(&kit, integration)
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

// HasMatchingTraits compare traits defined on kit against those defined on integration.
func HasMatchingTraits(kit *v1.IntegrationKit, integration *v1.Integration) (bool, error) {
	// The kit has no trait, but the integration need some
	if len(kit.Spec.Traits) == 0 && len(integration.Spec.Traits) > 0 {
		return false, nil
	}
	for name, kitTrait := range kit.Spec.Traits {
		intTrait, ok := integration.Spec.Traits[name]
		if !ok {
			// skip it because trait configured on kit is not defined on integration
			return false, nil
		}
		data, err := json.Marshal(intTrait.Configuration)
		if err != nil {
			return false, err
		}
		intConf := make(map[string]interface{})
		err = json.Unmarshal(data, &intConf)
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
			iv, ok := intConf[ck]
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

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

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/log"
)

func lookupKitsForIntegration(ctx context.Context, c client.Client, integration *v1.Integration, options ...ctrl.ListOption) ([]v1.IntegrationKit, error) {
	pl, err := platform.GetForResource(ctx, c, integration)
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
	for i := range list.Items {
		kit := &list.Items[i]
		match, err := integrationMatches(ctx, c, integration, kit)
		if err != nil {
			return nil, err
		} else if !match || kit.Status.Phase == v1.IntegrationKitPhaseError {
			continue
		}
		kits = append(kits, *kit)
	}

	return kits, nil
}

// sameOrMatch returns whether the v1.IntegrationKit is the one used by the v1.Integration or if it meets the
// requirements of the v1.Integration.
func sameOrMatch(ctx context.Context, c client.Client, kit *v1.IntegrationKit, integration *v1.Integration) (bool, error) {
	if integration.Status.IntegrationKit != nil {
		if integration.Status.IntegrationKit.Namespace == kit.Namespace && integration.Status.IntegrationKit.Name == kit.Name {
			return true, nil
		}
	}

	return integrationMatches(ctx, c, integration, kit)
}

// integrationMatches returns whether the v1.IntegrationKit meets the requirements of the v1.Integration.
func integrationMatches(ctx context.Context, c client.Client, integration *v1.Integration, kit *v1.IntegrationKit) (bool, error) {
	ilog := log.ForIntegration(integration)

	ilog.Debug("Matching integration", "integration", integration.Name, "integration-kit", kit.Name, "namespace", integration.Namespace)
	if !statusMatches(integration, kit, &ilog) {
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

	pl, err := platform.GetForResource(ctx, c, integration)
	if err != nil && !k8serrors.IsNotFound(err) {
		return false, err
	}
	if _, err := platform.ApplyIntegrationProfile(ctx, c, pl, integration); err != nil {
		return false, err
	}

	itc, err := trait.NewSpecTraitsOptionsForIntegrationAndPlatform(integration, pl)
	if err != nil {
		return false, err
	}
	ikc, err := trait.NewSpecTraitsOptionsForIntegrationKit(kit)
	if err != nil {
		return false, err
	}

	if match, err := trait.HasMatchingTraits(itc, ikc); !match || err != nil {
		ilog.Debug("Integration and integration-kit traits do not match", "integration", integration.Name, "integration-kit", kit.Name, "namespace", integration.Namespace)
		return false, err
	}
	if !util.StringSliceContains(kit.Spec.Dependencies, integration.Status.Dependencies) {
		ilog.Debug("Integration and integration-kit dependencies do not match", "integration", integration.Name, "integration-kit", kit.Name, "namespace", integration.Namespace)
		return false, nil
	}
	// If IntegrationKit has any source, we must verify that it corresponds with the one in the Integration.
	// This is important in case of Native builds as we need to rebuild when language requires a source during build.
	if (kit.Spec.Sources != nil && len(kit.Spec.Sources) > 0) && !hasMatchingSourcesForNative(integration, kit) {
		ilog.Debug("Integration and integration-kit sources do not match", "integration", integration.Name, "integration-kit", kit.Name, "namespace", integration.Namespace)
		return false, nil
	}

	ilog.Debug("Matched Integration and integration-kit", "integration", integration.Name, "integration-kit", kit.Name, "namespace", integration.Namespace)
	return true, nil
}

func statusMatches(integration *v1.Integration, kit *v1.IntegrationKit, ilog *log.Logger) bool {
	if kit.Status.Version != integration.Status.Version {
		ilog.Debug("Integration and integration-kit versions do not match", "integration", integration.Name, "integration-kit", kit.Name, "namespace", integration.Namespace)
		return false
	}
	if kit.Status.RuntimeProvider != integration.Status.RuntimeProvider {
		ilog.Debug("Integration and integration-kit runtime providers do not match", "integration", integration.Name, "integration-kit", kit.Name, "namespace", integration.Namespace)
		return false
	}
	if kit.Status.RuntimeVersion != integration.Status.RuntimeVersion {
		ilog.Debug("Integration and integration-kit runtime versions do not match", "integration", integration.Name, "integration-kit", kit.Name, "namespace", integration.Namespace)
		return false
	}
	if len(integration.Status.Dependencies) != len(kit.Spec.Dependencies) {
		ilog.Debug("Integration and integration-kit have different number of dependencies", "integration", integration.Name, "integration-kit", kit.Name, "namespace", integration.Namespace)
	}

	return true
}

// kitMatches returns whether the kit matches with the existing target kit.
func kitMatches(kit *v1.IntegrationKit, target *v1.IntegrationKit) (bool, error) {
	version := kit.Status.Version
	if version == "" {
		// Defaults with the version that is going to be set during the kit initialization
		version = defaults.Version
	}
	if version != target.Status.Version {
		return false, nil
	}
	if len(kit.Spec.Dependencies) != len(target.Spec.Dependencies) {
		return false, nil
	}

	// We cannot have yet the status set
	c1, err := trait.NewSpecTraitsOptionsForIntegrationKit(kit)
	if err != nil {
		return false, err
	}
	c2, err := trait.NewSpecTraitsOptionsForIntegrationKit(target)
	if err != nil {
		return false, err
	}

	if match, err := trait.HasMatchingTraits(c1, c2); !match || err != nil {
		return false, err
	}
	if !util.StringSliceContains(kit.Spec.Dependencies, target.Spec.Dependencies) {
		return false, nil
	}

	return true, nil
}

func hasMatchingSourcesForNative(it *v1.Integration, kit *v1.IntegrationKit) bool {
	if len(it.UserDefinedSources()) != len(kit.Spec.Sources) {
		return false
	}
	for _, itSource := range it.UserDefinedSources() {
		found := false
		for _, ikSource := range kit.Spec.Sources {
			if itSource.Content == ikSource.Content {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

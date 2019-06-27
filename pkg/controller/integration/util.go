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

	"github.com/apache/camel-k/pkg/util"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/pkg/errors"
)

var allowedLookupLabels = map[string]bool{
	v1alpha1.IntegrationKitTypePlatform: true,
	v1alpha1.IntegrationKitTypeExternal: true,
}

// LookupKitForIntegration --
func LookupKitForIntegration(ctx context.Context, c k8sclient.Reader, integration *v1alpha1.Integration) (*v1alpha1.IntegrationKit, error) {
	if integration.Status.Kit != "" {
		name := integration.Status.Kit
		kit := v1alpha1.NewIntegrationKit(integration.Namespace, name)
		key := k8sclient.ObjectKey{
			Namespace: integration.Namespace,
			Name:      name,
		}
		if err := c.Get(ctx, key, &kit); err != nil {
			return nil, errors.Wrapf(err, "unable to find integration kit %s, %s", name, err)
		}

		return &kit, nil
	}

	ctxList := v1alpha1.NewIntegrationKitList()
	if err := c.List(ctx, &k8sclient.ListOptions{Namespace: integration.Namespace}, &ctxList); err != nil {
		return nil, err
	}

	for _, ctx := range ctxList.Items {
		ctx := ctx // pin

		if ctx.Status.Phase == v1alpha1.IntegrationKitPhaseError {
			continue
		}
		if ctx.Status.CamelVersion != integration.Status.CamelVersion {
			continue
		}
		if ctx.Status.RuntimeVersion != integration.Status.RuntimeVersion {
			continue
		}
		if ctx.Status.Version != integration.Status.Version {
			continue
		}

		if allowed, ok := allowedLookupLabels[ctx.Labels["camel.apache.org/kit.type"]]; ok && allowed {
			ideps := len(integration.Status.Dependencies)
			cdeps := len(ctx.Spec.Dependencies)

			if ideps != cdeps {
				continue
			}

			//
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
			//
			if !HasMatchingTraits(&ctx, integration) {
				continue
			}

			if util.StringSliceContains(ctx.Spec.Dependencies, integration.Status.Dependencies) {
				return &ctx, nil
			}
		}
	}

	return nil, nil
}

// HasMatchingTraits compare traits defined on kit against those defined on integration.
func HasMatchingTraits(kit *v1alpha1.IntegrationKit, integration *v1alpha1.Integration) bool {
	for ctxTraitName, ctxTraitConf := range kit.Spec.Traits {
		iTraitConf, ok := integration.Spec.Traits[ctxTraitName]
		if !ok {
			// skip it because trait configured on kit is not defined on integration.
			return false
		}

		for ck, cv := range ctxTraitConf.Configuration {
			iv, ok := iTraitConf.Configuration[ck]

			if !ok {
				// skip it because trait configured on kit has a value that is not defined
				// in integration trait
				return false
			}
			if iv != cv {
				// skip it because trait configured on kit has a value that differs from
				// the one configured on integration
				return false
			}
		}
	}

	return true
}

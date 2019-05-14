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
	v1alpha1.IntegrationContextTypePlatform: true,
	v1alpha1.IntegrationContextTypeExternal: true,
}

// LookupContextForIntegration --
func LookupContextForIntegration(ctx context.Context, c k8sclient.Reader, integration *v1alpha1.Integration) (*v1alpha1.IntegrationContext, error) {
	if integration.Status.Context != "" {
		name := integration.Status.Context
		ictx := v1alpha1.NewIntegrationContext(integration.Namespace, name)
		key := k8sclient.ObjectKey{
			Namespace: integration.Namespace,
			Name:      name,
		}
		if err := c.Get(ctx, key, &ictx); err != nil {
			return nil, errors.Wrapf(err, "unable to find integration context %s, %s", name, err)
		}

		return &ictx, nil
	}

	ctxList := v1alpha1.NewIntegrationContextList()
	if err := c.List(ctx, &k8sclient.ListOptions{Namespace: integration.Namespace}, &ctxList); err != nil {
		return nil, err
	}

	for _, ctx := range ctxList.Items {
		ctx := ctx // pin

		if ctx.Status.Phase == v1alpha1.IntegrationContextPhaseError {
			continue
		}
		if ctx.Status.CamelVersion != integration.Status.CamelVersion {
			continue
		}
		if ctx.Status.RuntimeVersion != integration.Status.RuntimeVersion {
			continue
		}

		if allowed, ok := allowedLookupLabels[ctx.Labels["camel.apache.org/context.type"]]; ok && allowed {
			ideps := len(integration.Status.Dependencies)
			cdeps := len(ctx.Spec.Dependencies)

			if ideps != cdeps {
				continue
			}

			//
			// When a platform context is created it inherits the traits from the integrations and as
			// some traits may influence the build thus the artifacts present on the container image,
			// we need to take traits into account when looking up for compatible contexts.
			//
			// It could also happen that an integration is updated and a trait is modified, if we do
			// not include traits in the lookup, we may use a context that does not have all the
			// characteristics required by the integration.
			//
			// An context be used only if it contains a subset of the traits and related configurations
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

// HasMatchingTraits compare traits defined on context against those defined on integration.
func HasMatchingTraits(ctx *v1alpha1.IntegrationContext, integration *v1alpha1.Integration) bool {
	for ctxTraitName, ctxTraitConf := range ctx.Spec.Traits {
		iTraitConf, ok := integration.Spec.Traits[ctxTraitName]
		if !ok {
			// skip it because trait configured on context is not defined on integration.
			return false
		}

		for ck, cv := range ctxTraitConf.Configuration {
			iv, ok := iTraitConf.Configuration[ck]

			if !ok {
				// skip it because trait configured on context has a value that is not defined
				// in integration trait
				return false
			}
			if iv != cv {
				// skip it because trait configured on context has a value that differs from
				// the one configured on integration
				return false
			}
		}
	}

	return true
}

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

package action

import (
	"fmt"

	"github.com/rs/xid"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

// NewBuildAction create an action that handles integration build
func NewBuildAction(namespace string) IntegrationAction {
	return &buildAction{
		namespace: namespace,
	}
}

type buildAction struct {
	namespace string
}

func (action *buildAction) Name() string {
	return "build"
}

func (action *buildAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseBuilding
}

func (action *buildAction) Handle(integration *v1alpha1.Integration) error {
	ctx, err := LookupContextForIntegration(integration)
	if err != nil {
		//TODO: we may need to add a wait strategy, i.e give up after some time
		return err
	}

	if ctx != nil {
		if ctx.Labels["camel.apache.org/context.type"] == "platform" {
			// This is a platform context and as it is auto generated it may get
			// out of sync if the integration that has generated it, has been
			// amended to add/remove dependencies

			//TODO: this is a very simple check, we may need to provide a deps comparison strategy
			if !StringSliceContains(ctx.Spec.Dependencies, integration.Spec.Dependencies) {
				// We need to re-generate a context or search for a new one that
				// satisfies integrations needs so let's remove the association
				// with a context
				target := integration.DeepCopy()
				target.Spec.Context = ""
				return sdk.Update(target)
			}
		}

		if ctx.Status.Phase == v1alpha1.IntegrationContextPhaseReady {
			target := integration.DeepCopy()
			target.Status.Image = ctx.Status.Image
			target.Spec.Context = ctx.Name
			target.Status.Phase = v1alpha1.IntegrationPhaseDeploying
			return sdk.Update(target)
		}

		if integration.Spec.Context == "" {
			// We need to set the context
			target := integration.DeepCopy()
			target.Spec.Context = ctx.Name
			return sdk.Update(target)
		}

		return nil
	}

	platformCtxName := fmt.Sprintf("ctx-%s", xid.New())
	platformCtx := v1alpha1.NewIntegrationContext(action.namespace, platformCtxName)

	// Add some information for post-processing, this may need to be refactored
	// to a proper data structure
	platformCtx.Labels = map[string]string{
		"camel.apache.org/context.type":               "platform",
		"camel.apache.org/context.created.by.kind":    v1alpha1.IntegrationKind,
		"camel.apache.org/context.created.by.name":    integration.Name,
		"camel.apache.org/context.created.by.version": integration.ResourceVersion,
	}

	// Set the context to have the same dependencies as the integrations
	platformCtx.Spec = v1alpha1.IntegrationContextSpec{
		Dependencies: integration.Spec.Dependencies,
	}

	if err := sdk.Create(&platformCtx); err != nil {
		return err
	}

	// Set the context name so the next handle loop, will fall through the
	// same path as integration with a user defined context
	target := integration.DeepCopy()
	target.Spec.Context = platformCtxName
	return sdk.Update(target)
}

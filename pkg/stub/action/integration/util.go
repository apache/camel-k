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
	"github.com/apache/camel-k/pkg/util"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
)

// LookupContextForIntegration --
func LookupContextForIntegration(integration *v1alpha1.Integration) (*v1alpha1.IntegrationContext, error) {
	if integration.Spec.Context != "" {
		name := integration.Spec.Context
		ctx := v1alpha1.NewIntegrationContext(integration.Namespace, name)

		if err := sdk.Get(&ctx); err != nil {
			return nil, errors.Wrapf(err, "unable to find integration context %s, %s", ctx.Name, err)
		}

		return &ctx, nil
	}

	ctxList := v1alpha1.NewIntegrationContextList()
	if err := sdk.List(integration.Namespace, &ctxList); err != nil {
		return nil, err
	}

	for _, ctx := range ctxList.Items {
		ctx := ctx // pin
		if ctx.Labels["camel.apache.org/context.type"] == v1alpha1.KamelPlatform {
			ideps := len(integration.Spec.Dependencies)
			cdeps := len(ctx.Spec.Dependencies)

			if ideps != cdeps {
				continue
			}

			if util.StringSliceContains(ctx.Spec.Dependencies, integration.Spec.Dependencies) {
				return &ctx, nil
			}
		}
	}

	return nil, nil
}

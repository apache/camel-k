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

package integrationplatform

import (
	"context"
	"fmt"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/camel"

	corev1 "k8s.io/api/core/v1"
)

// NewCreateCatalogAction returns an action to create a new CamelCatalog.
func NewCreateCatalogAction() Action {
	return &catalogAction{}
}

type catalogAction struct {
	baseAction
}

func (action *catalogAction) Name() string {
	return "catalog"
}

func (action *catalogAction) CanHandle(platform *v1.IntegrationPlatform) bool {
	return platform.Status.Phase == v1.IntegrationPlatformPhaseCreateCatalog
}

func (action *catalogAction) Handle(ctx context.Context, platform *v1.IntegrationPlatform) (*v1.IntegrationPlatform, error) {
	// New runtime version set - check that catalog exists and create it if it does not exist
	runtimeSpec := v1.RuntimeSpec{
		Version:  platform.Status.Build.RuntimeVersion,
		Provider: v1.RuntimeProviderQuarkus,
	}

	if catalog, err := camel.LoadCatalog(ctx, action.client, platform.Namespace, runtimeSpec); err != nil {
		action.L.Error(err, "IntegrationPlatform unable to load Camel catalog",
			"runtime-version", runtimeSpec.Version, "runtime-provider", runtimeSpec.Provider)
		return platform, nil
	} else if catalog == nil {
		if _, err = camel.CreateCatalog(ctx, action.client, platform.Namespace, platform, runtimeSpec); err != nil {
			action.L.Error(err, "IntegrationPlatform unable to create Camel catalog",
				"runtime-version", runtimeSpec.Version, "runtime-provider", runtimeSpec.Provider)

			platform.Status.Phase = v1.IntegrationPlatformPhaseError
			platform.Status.SetCondition(
				v1.IntegrationPlatformConditionCamelCatalogAvailable,
				corev1.ConditionFalse,
				v1.IntegrationPlatformConditionCamelCatalogAvailableReason,
				fmt.Sprintf("camel catalog %s not available, please review given runtime version", runtimeSpec.Version))

			return platform, nil
		}
	}

	platform.Status.Phase = v1.IntegrationPlatformPhaseReady
	platform.Status.SetCondition(
		v1.IntegrationPlatformConditionCamelCatalogAvailable,
		corev1.ConditionTrue,
		v1.IntegrationPlatformConditionCamelCatalogAvailableReason,
		fmt.Sprintf("camel catalog %s available", runtimeSpec.Version))

	return platform, nil
}

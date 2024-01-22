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
	platformutil "github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
	corev1 "k8s.io/api/core/v1"
)

// NewMonitorAction returns an action that monitors the integration platform after it's fully initialized.
func NewMonitorAction() Action {
	return &monitorAction{}
}

type monitorAction struct {
	baseAction
}

func (action *monitorAction) Name() string {
	return "monitor"
}

func (action *monitorAction) CanHandle(platform *v1.IntegrationPlatform) bool {
	return platform.Status.Phase == v1.IntegrationPlatformPhaseReady || platform.Status.Phase == v1.IntegrationPlatformPhaseError
}

func (action *monitorAction) Handle(ctx context.Context, platform *v1.IntegrationPlatform) (*v1.IntegrationPlatform, error) {
	// Just track the version of the operator in the platform resource
	if platform.Status.Version != defaults.Version {
		platform.Status.Version = defaults.Version
		action.L.Info("IntegrationPlatform version updated", "version", platform.Status.Version)
	}

	platformPhase := v1.IntegrationPlatformPhaseReady

	// Refresh applied configuration
	if err := platformutil.ConfigureDefaults(ctx, action.client, platform, false); err != nil {
		return nil, err
	}

	// Registry condition
	isOpenshift, err := openshift.IsOpenShift(action.client)
	if err != nil {
		return platform, err
	}
	if isOpenshift {
		platform.Status.SetCondition(
			v1.IntegrationPlatformConditionTypeRegistryAvailable,
			corev1.ConditionFalse,
			v1.IntegrationPlatformConditionTypeRegistryAvailableReason,
			"registry not available because provided by Openshift")
	} else {
		if platform.Status.Build.Registry.Address == "" {
			// error, we need a registry if we're not on Openshift
			platformPhase = v1.IntegrationPlatformPhaseError
			platform.Status.SetCondition(
				v1.IntegrationPlatformConditionTypeRegistryAvailable,
				corev1.ConditionFalse,
				v1.IntegrationPlatformConditionTypeRegistryAvailableReason,
				"registry address not available, you need to set one")
		} else {
			platform.Status.SetCondition(
				v1.IntegrationPlatformConditionTypeRegistryAvailable,
				corev1.ConditionTrue,
				v1.IntegrationPlatformConditionTypeRegistryAvailableReason,
				fmt.Sprintf("registry available at %s", platform.Status.Build.Registry.Address))
		}
	}

	if platformPhase == v1.IntegrationPlatformPhaseReady {
		// Camel catalog condition
		runtimeSpec := v1.RuntimeSpec{
			Version:  platform.Status.Build.RuntimeVersion,
			Provider: v1.RuntimeProviderQuarkus,
		}
		if catalog, err := camel.LoadCatalog(ctx, action.client, platform.Namespace, runtimeSpec); err != nil {
			action.L.Error(err, "IntegrationPlatform unable to load Camel catalog",
				"runtime-version", runtimeSpec.Version, "runtime-provider", runtimeSpec.Provider)
		} else if catalog == nil {
			if platform.Status.Phase != v1.IntegrationPlatformPhaseError {
				platformPhase = v1.IntegrationPlatformPhaseCreateCatalog
			} else {
				// IntegrationPlatform is in error phase for some reason - that error state must be resolved before we move into create catalog phase
				// avoids to run into endless loop of error and catalog creation phase ping pong
				platformPhase = v1.IntegrationPlatformPhaseError
			}

			platform.Status.SetCondition(
				v1.IntegrationPlatformConditionCamelCatalogAvailable,
				corev1.ConditionFalse,
				v1.IntegrationPlatformConditionCamelCatalogAvailableReason,
				fmt.Sprintf("camel catalog %s not available, please review given runtime version", runtimeSpec.Version))
		} else {
			platform.Status.SetCondition(
				v1.IntegrationPlatformConditionCamelCatalogAvailable,
				corev1.ConditionTrue,
				v1.IntegrationPlatformConditionCamelCatalogAvailableReason,
				fmt.Sprintf("camel catalog %s available", runtimeSpec.Version))
		}
	}

	platform.Status.Phase = platformPhase

	return platform, nil
}

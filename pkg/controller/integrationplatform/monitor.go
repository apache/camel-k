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
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	platformutil "github.com/apache/camel-k/v2/pkg/platform"
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

//nolint:staticcheck
func (action *monitorAction) CanHandle(platform *v1.IntegrationPlatform) bool {
	return platform.Status.Phase == v1.IntegrationPlatformPhaseReady || platform.Status.Phase == v1.IntegrationPlatformPhaseError
}

//nolint:staticcheck
func (action *monitorAction) Handle(ctx context.Context, platform *v1.IntegrationPlatform) (*v1.IntegrationPlatform, error) {
	runtimeVersion := specOrDefault(platform.Spec.Build.RuntimeVersion)
	if platform.Status.Build.RuntimeVersion != runtimeVersion {
		action.L.Infof("IntegrationPlatform version updated from %s to %s", platform.Status.Build.RuntimeVersion, runtimeVersion)
		// Reset the status to reinitialize the resource
		platform.Status = v1.IntegrationPlatformStatus{}

		return platform, nil
	}
	// Sync status configuration
	if err := platformutil.ConfigureDefaults(ctx, action.client, platform, false); err != nil {
		return nil, err
	}
	// Get the information about Camel version in the catalog
	runtimeSpec := v1.RuntimeSpec{
		Version:  platform.Status.Build.RuntimeVersion,
		Provider: platform.Status.Build.RuntimeProvider,
	}
	if runtimeSpec.Provider == v1.RuntimeProviderPlainQuarkus {
		runtimeSpec.Version = defaults.CamelKRuntimeCatalogVersion
	}
	catalog, err := loadCatalog(ctx, action.client, platform.Namespace, runtimeSpec)
	if catalog == nil || err != nil {
		// error, a catalog must be available
		platform.Status.Phase = v1.IntegrationPlatformPhaseError
		platform.Status.SetCondition(
			v1.IntegrationPlatformConditionCamelCatalogAvailable,
			corev1.ConditionFalse,
			v1.IntegrationPlatformConditionCamelCatalogAvailableReason,
			fmt.Sprintf("camel catalog %s not available, please review given runtime version", runtimeSpec.Version))

		return platform, err
	}
	// So far the ITP is good
	platform.Status.Phase = v1.IntegrationPlatformPhaseReady
	platform.Status.Build.RuntimeCoreVersion = catalog.Spec.GetCamelVersion()
	// Registry condition
	isOpenshift, err := openshift.IsOpenShift(action.client)
	if err != nil {
		return platform, err
	}
	//nolint: nestif
	if isOpenshift && platform.Status.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategyS2I { //nolint:staticcheck
		platform.Status.SetCondition(
			v1.IntegrationPlatformConditionTypeRegistryAvailable,
			corev1.ConditionFalse,
			v1.IntegrationPlatformConditionTypeRegistryAvailableReason,
			"registry not available because provided by Openshift S2I")
		action.L.Infof("WARN: S2I publishing strategy is deprecated and may be removed in the future, use Jib strategy instead")
		platform.Status.SetCondition(
			"S2IPublishingStrategyDeprecated",
			corev1.ConditionTrue,
			"PublishingStrategyDeprecationNoticeReason",
			"S2I publishing strategy is deprecated and may be removed in the future, use Jib strategy instead")
	} else {
		if platform.Status.Build.Registry.Address == "" {
			// error, we need a registry if we're not on Openshift
			platform.Status.Phase = v1.IntegrationPlatformPhaseError
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
				"registry available at "+platform.Status.Build.Registry.Address)
			// Warn if insecure registry
			if platform.Status.Build.Registry.Insecure {
				platform.Status.SetCondition(
					v1.IntegrationPlatformConditionType("InsecureRegistryWarning"),
					corev1.ConditionTrue,
					"InsecureRegistryWarningReason",
					"Registry is insecure. This setup should not be used in a production environment.")
				action.L.Infof(
					"WARN: provided container registry is insecure. This setup should not be used in a production environment.",
				)
			}
		}
	}
	action.checkTraitAnnotationsDeprecatedNotice(platform)
	action.checkMavenSettings(platform)
	action.checkKameletsCatalogRepoDeprecatedNotice(platform)

	platform.Status.SetCondition(
		v1.IntegrationPlatformConditionType("IntegrationPlatformDeprecated"),
		corev1.ConditionTrue,
		"DeprecationNotice",
		"The IntegrationPlatform custom resource is deprecated. Please, configure operator environment variables instead.",
	)

	return platform, nil
}

// Deprecated: to be removed in future versions, when we won't support any longer trait annotations into IntegrationPlatforms.
func (action *monitorAction) checkTraitAnnotationsDeprecatedNotice(platform *v1.IntegrationPlatform) {
	if platform.Annotations != nil {
		for k := range platform.Annotations {
			if strings.HasPrefix(k, v1.TraitAnnotationPrefix) {
				platform.Status.SetCondition(
					v1.IntegrationPlatformConditionType("AnnotationTraitsDeprecated"),
					corev1.ConditionTrue,
					"DeprecationNotice",
					"Annotation traits configuration is deprecated and will be removed soon. Use .spec.traits configuration instead.",
				)
				action.L.Infof(
					"WARN: annotation traits configuration is deprecated and will be removed soon. Use .spec.traits configuration for %s platform instead.",
					platform.Name,
				)

				return
			}
		}
	}
}

// Deprecated: to be removed in future versions, when we won't support any longer Kamelets catalog into IntegrationPlatforms.
func (action *monitorAction) checkKameletsCatalogRepoDeprecatedNotice(platform *v1.IntegrationPlatform) {
	if platform.Status.Kamelet.Repositories != nil {
		platform.Status.SetCondition(
			v1.IntegrationPlatformConditionType("KameletsCatalogRepositoryDeprecated"),
			corev1.ConditionTrue,
			"DeprecationNotice",
			"Kamelets Catalog Repository is deprecated and will be removed soon. Bundle Kamelets into a Maven dependency instead.",
		)
		action.L.Infof(
			"Kamelets Catalog Repository is deprecated and will be removed soon from IntegrationPlatform configuration. " +
				"Bundle Kamelets into a Maven dependency instead")
	}
}

//nolint:staticcheck
func (action *monitorAction) checkMavenSettings(platform *v1.IntegrationPlatform) {
	if platform.Status.Build.Maven.Settings.ConfigMapKeyRef != nil ||
		platform.Status.Build.Maven.Settings.SecretKeyRef != nil {
		platform.Status.SetCondition(
			v1.IntegrationPlatformConditionMavenSettingsAvailable,
			corev1.ConditionTrue,
			"MavenSettingsAvailable",
			"Maven settings are available.",
		)
	} else {
		platform.Status.SetCondition(
			v1.IntegrationPlatformConditionMavenSettingsAvailable,
			corev1.ConditionFalse,
			"MavenSettingsAvailable",
			"Maven settings are missing. You need to provide at least a Maven proxy configuration for performance reasons.",
		)
		action.L.Infof(
			"WARN: Maven settings are missing for platform %s. "+
				"You need to provide at least a Maven proxy configuration for performance reasons.",
			platform.Name,
		)
	}
}

func specOrDefault(runtimeVersionSpec string) string {
	if runtimeVersionSpec == "" {
		return defaults.DefaultRuntimeVersion
	}

	return runtimeVersionSpec
}

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

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/resources"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/log"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewCreateAction returns the action that creates resources needed by the platform.
func NewCreateAction() Action {
	return &createAction{}
}

type createAction struct {
	baseAction
}

func (action *createAction) Name() string {
	return "create"
}

func (action *createAction) CanHandle(platform *v1.IntegrationPlatform) bool {
	return platform.Status.Phase == v1.IntegrationPlatformPhaseCreating
}

func (action *createAction) Handle(ctx context.Context, platform *v1.IntegrationPlatform) (*v1.IntegrationPlatform, error) {
	runtimeSpec := v1.RuntimeSpec{
		Version:  platform.Status.Build.RuntimeVersion,
		Provider: platform.Status.Build.RuntimeProvider,
	}
	catalog, err := loadCatalog(ctx, action.client, platform.Namespace, runtimeSpec)
	if err != nil {
		return nil, err
	}

	// if bundled version, load catalog spec from resources
	if platform.Status.Build.RuntimeVersion == defaults.DefaultRuntimeVersion {
		if platform, err = action.handleBundledCatalog(ctx, platform, catalog); err != nil {
			return platform, err
		}
	} else {
		// otherwise get the catalog from external dependency
		if platform, err = action.handleNewCatalog(ctx, platform, catalog, runtimeSpec); err != nil {
			return platform, err
		}
	}

	platform.Status.Phase = v1.IntegrationPlatformPhaseReady
	platform.Status.SetCondition(
		v1.IntegrationPlatformConditionCamelCatalogAvailable,
		corev1.ConditionTrue,
		v1.IntegrationPlatformConditionCamelCatalogAvailableReason,
		fmt.Sprintf("camel catalog %s available", platform.Status.Build.RuntimeVersion))

	if platform.Status.Build.RuntimeCoreVersion != "" {
		action.L.Infof("IntegrationPlatform is about to install Apache Kamelet Catalog version %s", platform.Status.Build.RuntimeCoreVersion)
		return installKamelets(ctx, action.client, platform)
	} else {
		action.L.Info("IntegrationPlatform has no Camel core version. " +
			"It is likely an unsupported specification, please, update to the latest one")
	}

	return platform, nil
}

func loadCatalog(ctx context.Context, c client.Client, namespace string, runtimeSpec v1.RuntimeSpec) (*v1.CamelCatalog, error) {
	options := []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
	}
	list := v1.NewCamelCatalogList()
	if err := c.List(ctx, &list, options...); err != nil {
		return nil, err
	}
	for _, cc := range list.Items {
		if cc.Spec.Runtime.Provider == runtimeSpec.Provider && cc.Spec.Runtime.Version == runtimeSpec.Version {
			return &cc, nil
		}
	}

	return nil, nil
}

func (action *createAction) handleBundledCatalog(ctx context.Context, platform *v1.IntegrationPlatform, catalog *v1.CamelCatalog) (*v1.IntegrationPlatform, error) {
	var camelVersion string
	// Create the catalog only if it was not yet created
	if catalog == nil {
		camelCatalogData, err := resources.Resource(fmt.Sprintf("/resources/camel-catalog-%s.yaml", platform.Status.Build.RuntimeVersion))
		if err != nil {
			return nil, err
		}
		var cat v1.CamelCatalog
		if err = yaml.Unmarshal(camelCatalogData, &cat); err != nil {
			return nil, err
		}
		// Copy platform annotations to the catalog
		cat.SetAnnotations(platform.Annotations)
		cat.SetNamespace(platform.Namespace)
		action.L.Infof("Installing bundled camel catalog: %s", platform.Status.Build.RuntimeVersion)
		if err = action.client.Create(ctx, &cat); err != nil {
			return nil, err
		}
		camelVersion = cat.Spec.GetCamelVersion()
	} else {
		camelVersion = catalog.Spec.GetCamelVersion()
	}
	platform.Status.Build.RuntimeCoreVersion = camelVersion

	return platform, nil
}

func (action *createAction) handleNewCatalog(ctx context.Context, platform *v1.IntegrationPlatform,
	catalog *v1.CamelCatalog, runtimeSpec v1.RuntimeSpec) (*v1.IntegrationPlatform, error) {
	var camelVersion string
	if catalog == nil {
		cat, err := camel.CreateCatalog(
			ctx, action.client, platform.Namespace, platform.Status.Build.Maven,
			platform.Status.Build.GetTimeout().Duration, runtimeSpec, nil)
		if err != nil {
			action.L.Error(err, "IntegrationPlatform unable to create Camel catalog",
				"runtime-version", runtimeSpec.Version, "runtime-provider", runtimeSpec.Provider)

			platform.Status.Phase = v1.IntegrationPlatformPhaseError
			platform.Status.SetCondition(
				v1.IntegrationPlatformConditionCamelCatalogAvailable,
				corev1.ConditionFalse,
				v1.IntegrationPlatformConditionCamelCatalogAvailableReason,
				fmt.Sprintf("camel catalog %s not available, please review given runtime version. Error: %s", runtimeSpec.Version, err))

			return platform, err
		}
		camelVersion = cat.GetCamelVersion()
	} else {
		camelVersion = catalog.Spec.GetCamelVersion()
	}
	platform.Status.Build.RuntimeCoreVersion = camelVersion

	return platform, nil
}

func installKamelets(ctx context.Context, c client.Client, platform *v1.IntegrationPlatform) (*v1.IntegrationPlatform, error) {
	// We bundle the Kamelets driven by the catalog
	if defaults.InstallDefaultKamelets() {
		camelVersion := platform.Status.Build.RuntimeCoreVersion
		installedKam, erroredKam, err := installKameletCatalog(ctx, c, platform, camelVersion)
		if err != nil {
			// An error here should not be disruptive, we just report it happened
			platform.Status.SetCondition(
				v1.IntegrationPlatformConditionKameletCatalogAvailable,
				corev1.ConditionFalse,
				"IntegrationPlatformKameletCatalogAvailable",
				fmt.Sprintf("kamelet catalog %s not available. Error: %v", camelVersion, err),
			)
			log.Errorf(err, "A failure happened while loading Kamelet catalog %s", camelVersion)
		} else {
			message := fmt.Sprintf("successfully installed Kamelet catalog version %s: success %d Kamelets, failed %d Kamelets",
				camelVersion, installedKam, erroredKam)
			if erroredKam > 0 {
				message += ". Check operator log to discover more about the failure"
			}
			platform.Status.SetCondition(
				v1.IntegrationPlatformConditionKameletCatalogAvailable,
				corev1.ConditionTrue,
				"IntegrationPlatformKameletCatalogAvailable",
				message,
			)
		}
	}

	return platform, nil
}

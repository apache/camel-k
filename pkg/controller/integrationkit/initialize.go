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

package integrationkit

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewInitializeAction creates a new initialization handling action for the kit.
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(kit *v1.IntegrationKit) bool {
	return kit.Status.Phase == v1.IntegrationKitPhaseInitialization ||
		kit.Status.Phase == v1.IntegrationKitPhaseWaitingForCatalog
}

func (action *initializeAction) Handle(ctx context.Context, kit *v1.IntegrationKit) (*v1.IntegrationKit, error) {
	action.L.Info("Initializing IntegrationKit")
	if kit.Spec.Image != "" {
		// Synthetic Kit
		action.L.Info("Synthetic Kit, won't be able to build or monitor this one.")
		kit.Status.Phase = v1.IntegrationKitPhaseReady
		kit.Status.Image = kit.Spec.Image

		return kit, nil
	}

	// Managed Kit
	env, err := trait.Apply(ctx, action.client, nil, kit)
	if err != nil {
		return nil, err
	}

	kit.Status.Version = defaults.Version

	if kit.Spec.Image != "" {
		return kit, nil
	}

	if err := action.image(ctx, env, kit); err != nil {
		return nil, err
	}

	return kit, nil
}

func (action *initializeAction) image(ctx context.Context, env *trait.Environment, kit *v1.IntegrationKit) error {
	catalogName := fmt.Sprintf("camel-catalog-%s", strings.ToLower(env.CamelCatalog.GetRuntimeVersion()))
	if env.CamelCatalog.GetRuntimeProvider() == v1.RuntimeProviderPlainQuarkus {
		// We need this workaround to load the last existing catalog
		// TODO: this part will be subject to future refactoring
		catalogName = fmt.Sprintf("camel-catalog-quarkus-%s", strings.ToLower(defaults.DefaultRuntimeVersion))
	}
	// Wait for CamelCatalog to be ready
	catalog, err := kubernetes.GetCamelCatalog(
		ctx,
		action.client,
		catalogName,
		kit.Namespace,
	)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			// If the catalog is not available, likely it was required to be created
			// by Integration trait, so we'll need to wait for it to be available
			kit.Status.Phase = v1.IntegrationKitPhaseWaitingForCatalog

			return nil
		}

		return err
	}

	if catalog.Status.Phase == v1.CamelCatalogPhaseError {
		errorReason := fmt.Sprintf("Camel Catalog %s error", catalog.Spec.Runtime.Version)

		kit.Status.Phase = v1.IntegrationKitPhaseError
		kit.Status.SetErrorCondition(
			v1.IntegrationKitConditionCatalogAvailable,
			errorReason,
			fmt.Errorf("%s", catalog.Status.GetCondition(v1.CamelCatalogConditionReady).Reason),
		)

		// Adding the failure in order to include this info in the Integration as well
		kit.Status.Failure = &v1.Failure{
			Reason: errorReason,
			Time:   metav1.Now(),
		}

		return nil
	}

	if catalog.Status.Phase != v1.CamelCatalogPhaseReady {
		kit.Status.Phase = v1.IntegrationKitPhaseWaitingForCatalog

		return nil
	}

	// now the kit can be built
	kit.Status.Phase = v1.IntegrationKitPhaseBuildSubmitted

	return nil
}

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
	env, err := trait.Apply(ctx, action.client, nil, kit)
	if err != nil {
		return nil, err
	}

	action.L.Info("Initializing IntegrationKit")
	kit.Status.Version = defaults.Version

	if kit.Spec.Image == "" {
		// Wait for CamelCatalog to be ready
		catalog, err := kubernetes.GetCamelCatalog(
			ctx,
			action.client,
			fmt.Sprintf("camel-catalog-%s", strings.ToLower(env.RuntimeVersion)),
			kit.Namespace,
		)

		if err != nil {
			if k8serrors.IsNotFound(err) {
				// If the catalog is not available, likely it was required to be created
				// by Integration trait, so we'll need to wait for it the be available
				kit.Status.Phase = v1.IntegrationKitPhaseWaitingForCatalog
				return kit, nil
			}
			return nil, err
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
			return kit, nil
		}

		if catalog.Status.Phase != v1.CamelCatalogPhaseReady {
			kit.Status.Phase = v1.IntegrationKitPhaseWaitingForCatalog
			return kit, nil
		}
		// now the kit can be built
		kit.Status.Phase = v1.IntegrationKitPhaseBuildSubmitted
	} else {
		// but in case it has been created from an image, mark the
		// kit as ready
		kit.Status.Phase = v1.IntegrationKitPhaseReady

		// and set the image to be used
		kit.Status.Image = kit.Spec.Image
	}

	return kit, nil
}

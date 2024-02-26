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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	platformutil "github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

// NewInitializeAction returns a action that initializes the platform configuration when not provided by the user.
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(platform *v1.IntegrationPlatform) bool {
	return platform.Status.Phase == v1.IntegrationPlatformPhaseNone || platform.Status.Phase == v1.IntegrationPlatformPhaseDuplicate
}

func (action *initializeAction) Handle(ctx context.Context, platform *v1.IntegrationPlatform) (*v1.IntegrationPlatform, error) {
	duplicate, err := action.isPrimaryDuplicate(ctx, platform)
	if err != nil {
		return nil, err
	}
	if duplicate {
		// another platform already present in the namespace
		if platform.Status.Phase != v1.IntegrationPlatformPhaseDuplicate {
			platform := platform.DeepCopy()
			platform.Status.Phase = v1.IntegrationPlatformPhaseDuplicate

			return platform, nil
		}

		return nil, nil
	}

	action.L.Info("Initializing IntegrationPlatform")
	if err = platformutil.ConfigureDefaults(ctx, action.client, platform, true); err != nil {
		return nil, err
	}
	platform.Status.Phase = v1.IntegrationPlatformPhaseCreating
	platform.Status.Version = defaults.Version

	return platform, nil
}

func (action *initializeAction) isPrimaryDuplicate(ctx context.Context, thisPlatform *v1.IntegrationPlatform) (bool, error) {
	if platformutil.IsSecondary(thisPlatform) {
		// Always reconcile secondary platforms
		return false, nil
	}
	platforms, err := platformutil.ListPrimaryPlatforms(ctx, action.client, thisPlatform.Namespace)
	if err != nil {
		return false, err
	}
	for _, p := range platforms.Items {
		p := p // pin
		if p.Name != thisPlatform.Name && platformutil.IsActive(&p) {
			return true, nil
		}
	}

	return false, nil
}

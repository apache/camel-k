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
	corev1 "k8s.io/api/core/v1"
)

// NewInitializeAction returns the action that initializes the integration platform when not provided by the user.
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
	return platform.Status.Phase == v1.IntegrationPlatformPhaseNone
}

func (action *initializeAction) Handle(ctx context.Context, platform *v1.IntegrationPlatform) (*v1.IntegrationPlatform, error) {
	action.L.Info("Initializing IntegrationPlatform")
	if err := platformutil.ConfigureDefaults(ctx, action.client, platform, true); err != nil {
		return nil, err
	}
	if platform.Status.Build.RuntimeVersion == "" {
		platform.Status.Phase = v1.IntegrationPlatformPhaseError
		platform.Status.SetCondition(
			v1.IntegrationPlatformConditionTypeCreated,
			corev1.ConditionFalse,
			"MissingRuntimeVersionSpec",
			"Runtime version missing from build spec")

		return platform, fmt.Errorf("runtime version missing from build spec")
	} else {
		platform.Status.Phase = v1.IntegrationPlatformPhaseCreating
	}

	return platform, nil
}

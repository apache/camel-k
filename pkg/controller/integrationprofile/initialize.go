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

package integrationprofile

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

// NewInitializeAction returns the action that initializes the integration profile when not provided by the user.
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(profile *v1.IntegrationProfile) bool {
	return profile.Status.Phase == v1.IntegrationProfilePhaseNone
}

func (action *initializeAction) Handle(ctx context.Context, profile *v1.IntegrationProfile) (*v1.IntegrationProfile, error) {
	action.L.Info("Initializing IntegrationProfile")
	profile.ResyncStatusFullConfig()

	profile.Status.SetCondition(
		v1.IntegrationProfileConditionTypeCreated,
		corev1.ConditionTrue,
		v1.IntegrationProfileConditionCreatedReason,
		"integration profile created")

	profile.Status.SetCondition(
		"DeprecationNotice",
		corev1.ConditionTrue,
		"DeprecationNoticeWarning",
		"Usage of IntegrationProfile is deprecated. Make use of IntegrationPlatform instead.")

	profile.Status.Phase = v1.IntegrationProfilePhaseReady

	return profile, nil
}

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

package action

import (
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

// initializes the integration status to trigger the deployment
type InitializeAction struct {
}

func NewInitializeAction() IntegrationAction {
	return &InitializeAction{}
}

func (b *InitializeAction) Name() string {
	return "initialize"
}

func (b *InitializeAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == ""
}

func (b *InitializeAction) Handle(integration *v1alpha1.Integration) error {
	target := integration.DeepCopy()
	// set default values
	var defaultReplicas int32 = 1
	if target.Spec.Replicas == nil {
		target.Spec.Replicas = &defaultReplicas
	}
	// update the status
	target.Status.Phase = v1alpha1.IntegrationPhaseBuilding
	target.Status.Digest = digest.ComputeForIntegration(integration)
	return sdk.Update(target)
}

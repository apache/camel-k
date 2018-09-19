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

// NewIntegrationContextInitializeAction creates a new initialization handling action for the context
func NewIntegrationContextInitializeAction() IntegrationContextAction {
	return &integrationContextInitializeAction{}
}

type integrationContextInitializeAction struct {
}

func (action *integrationContextInitializeAction) Name() string {
	return "initialize"
}

func (action *integrationContextInitializeAction) CanHandle(context *v1alpha1.IntegrationContext) bool {
	return context.Status.Phase == ""
}

func (action *integrationContextInitializeAction) Handle(context *v1alpha1.IntegrationContext) error {
	target := context.DeepCopy()

	// update the status
	target.Status.Phase = v1alpha1.IntegrationContextPhaseBuilding
	target.Status.Digest = digest.ComputeForIntegrationContext(context)

	return sdk.Update(target)
}

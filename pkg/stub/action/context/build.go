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
	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

func NewIntegrationContextBuildAction() IntegrationContextAction {
	return &integrationContextBuildAction{}
}

// start edit context
type integrationContextBuildAction struct {
}

func (action *integrationContextBuildAction) Name() string {
	return "Edit"
}

func (action *integrationContextBuildAction) CanHandle(context *v1alpha1.IntegrationContext) bool {
	// TODO: implement
	return false
}

func (action *integrationContextBuildAction) Handle(integration *v1alpha1.IntegrationContext) error {
	target := integration.DeepCopy()
	// TODO: implement
	return sdk.Update(target)
}

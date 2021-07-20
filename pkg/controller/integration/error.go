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

package integration

import (
	"context"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/digest"
)

// NewErrorAction creates a new error action for an integration
func NewErrorAction() Action {
	return &errorAction{}
}

type errorAction struct {
	baseAction
}

func (action *errorAction) Name() string {
	return "error"
}

func (action *errorAction) CanHandle(integration *v1.Integration) bool {
	return integration.Status.Phase == v1.IntegrationPhaseError
}

func (action *errorAction) Handle(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {
	hash, err := digest.ComputeForIntegration(integration)
	if err != nil {
		return nil, err
	}

	if hash != integration.Status.Digest {
		action.L.Info("Integration needs a rebuild")

		integration.Initialize()
		integration.Status.Digest = hash

		return integration, nil
	}

	err = setReadyCondition(ctx, action.client, integration)
	if err != nil {
		return nil, err
	}

	ready := integration.Status.GetCondition(v1.IntegrationConditionReady)
	if ready == nil || ready.Reason != v1.IntegrationConditionErrorReason {
		integration.Status.Phase = v1.IntegrationPhaseRunning
	}

	return nil, nil
}

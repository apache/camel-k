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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/digest"
)

// NewErrorAction creates a new error handling action for the kit
func NewErrorAction() Action {
	return &errorAction{}
}

type errorAction struct {
	baseAction
}

func (action *errorAction) Name() string {
	return "error"
}

func (action *errorAction) CanHandle(kit *v1.IntegrationKit) bool {
	return kit.Status.Phase == v1.IntegrationKitPhaseError ||
		kit.Status.Phase == v1.IntegrationKitPhaseCannotBuild
}

func (action *errorAction) Handle(ctx context.Context, kit *v1.IntegrationKit) (*v1.IntegrationKit, error) {
	hash, err := digest.ComputeForIntegrationKit(kit)
	if err != nil {
		return nil, err
	}
	if hash != kit.Status.Digest {
		action.L.Info("IntegrationKit needs a rebuild")

		kit.Status.Digest = hash
		kit.Status.Phase = v1.IntegrationKitPhaseInitialization

		return kit, nil
	}

	return nil, nil
}

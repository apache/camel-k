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

// NewMonitorAction creates a new monitoring handling action for the kit
func NewMonitorAction() Action {
	return &monitorAction{}
}

type monitorAction struct {
	baseAction
}

func (action *monitorAction) Name() string {
	return "monitor"
}

func (action *monitorAction) CanHandle(kit *v1.IntegrationKit) bool {
	return kit.Status.Phase == v1.IntegrationKitPhaseReady
}

func (action *monitorAction) Handle(ctx context.Context, kit *v1.IntegrationKit) (*v1.IntegrationKit, error) {
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
	if kit.Spec.Image != "" && kit.Spec.Image != kit.Status.Image {
		kit.Status.Phase = v1.IntegrationKitPhaseInitialization

		return kit, nil
	}

	return nil, nil
}

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

package integrationcontext

import (
	"context"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/digest"
)

// NewMonitorAction creates a new monitoring handling action for the context
func NewMonitorAction() Action {
	return &monitorAction{}
}

type monitorAction struct {
	baseAction
}

func (action *monitorAction) Name() string {
	return "monitor"
}

func (action *monitorAction) CanHandle(ictx *v1alpha1.IntegrationContext) bool {
	return ictx.Status.Phase == v1alpha1.IntegrationContextPhaseReady || ictx.Status.Phase == v1alpha1.IntegrationContextPhaseError
}

func (action *monitorAction) Handle(ctx context.Context, ictx *v1alpha1.IntegrationContext) error {
	hash, err := digest.ComputeForIntegrationContext(ictx)
	if err != nil {
		return err
	}
	if hash != ictx.Status.Digest {
		action.L.Info("IntegrationContext needs a rebuild")

		target := ictx.DeepCopy()
		target.Status.Digest = hash
		target.Status.Phase = v1alpha1.IntegrationContextPhaseBuildSubmitted

		action.L.Info("IntegrationContext state transition", "phase", target.Status.Phase)

		return action.client.Status().Update(ctx, target)
	}

	return nil
}

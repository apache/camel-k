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

package context

import (
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
)

// NewMonitorAction creates a new monitoring handling action for the context
func NewMonitorAction() Action {
	return &monitorAction{}
}

type monitorAction struct {
}

func (action *monitorAction) Name() string {
	return "monitor"
}

func (action *monitorAction) CanHandle(context *v1alpha1.IntegrationContext) bool {
	return context.Status.Phase == v1alpha1.IntegrationContextPhaseReady || context.Status.Phase == v1alpha1.IntegrationContextPhaseError
}

func (action *monitorAction) Handle(context *v1alpha1.IntegrationContext) error {
	hash, err := digest.ComputeForIntegrationContext(context)
	if err != nil {
		return err
	}
	if hash != context.Status.Digest {
		logrus.Info("IntegrationContext ", context.Name, " needs a rebuild")

		target := context.DeepCopy()
		target.Status.Digest = hash
		logrus.Info("Context ", target.Name, " transitioning to state ", v1alpha1.IntegrationContextPhaseBuilding)
		target.Status.Phase = v1alpha1.IntegrationContextPhaseBuilding
		return sdk.Update(target)
	}

	return nil
}

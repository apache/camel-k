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
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/sirupsen/logrus"
)

type MonitorAction struct {
}

func NewMonitorAction() *MonitorAction {
	return &MonitorAction{}
}

func (b *MonitorAction) Name() string {
	return "monitor"
}

func (a *MonitorAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseRunning ||
		integration.Status.Phase == v1alpha1.IntegrationPhaseError
}

func (a *MonitorAction) Handle(integration *v1alpha1.Integration) error {

	hash := digest.Compute(integration)
	if hash != integration.Status.Digest {
		logrus.Info("Integration ", integration.Name, " needs a rebuild")

		target := integration.DeepCopy()
		target.Status.Digest=hash
		target.Status.Phase=v1alpha1.IntegrationPhaseBuilding
		return sdk.Update(target)
	}

	// TODO check also if deployment matches (e.g. replicas)
	return nil
}

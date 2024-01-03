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
	"fmt"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/trait"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// NewMonitorSyntheticAction is an action used to monitor synthetic Integrations.
func NewMonitorSyntheticAction() Action {
	return &monitorSyntheticAction{}
}

type monitorSyntheticAction struct {
	monitorAction
}

func (action *monitorSyntheticAction) Name() string {
	return "monitor-synthetic"
}

func (action *monitorSyntheticAction) Handle(ctx context.Context, integration *v1.Integration) (*v1.Integration, error) {
	environment, err := trait.NewSyntheticEnvironment(ctx, action.client, integration, nil)
	if err != nil {
		// Importing application no longer available
		if k8serrors.IsNotFound(err) {
			// It could be a normal condition, don't report as an error
			integration.Status.Phase = v1.IntegrationPhaseImportMissing
			message := fmt.Sprintf(
				"import %s %s no longer available",
				integration.Annotations[v1.IntegrationImportedKindLabel],
				integration.Annotations[v1.IntegrationImportedNameLabel],
			)
			integration.SetReadyConditionError(message)
			zero := int32(0)
			integration.Status.Phase = v1.IntegrationPhaseImportMissing
			integration.Status.Replicas = &zero
			return integration, nil
		}
		// other reasons, likely some error to report
		integration.Status.Phase = v1.IntegrationPhaseError
		integration.SetReadyCondition(corev1.ConditionFalse, v1.IntegrationConditionImportingKindAvailableReason, err.Error())
		return integration, err
	}

	if environment == nil {
		// The application which generated the Integration has no longer the importing label. We may have missed the
		// delete event, therefore we need to perform the operation here.
		err := action.client.Delete(ctx, integration)
		action.L.Infof("Deleting synthetic Integration %s", integration.Name)
		if err != nil {
			return integration, err
		}
		return nil, nil
	}

	return action.monitorPods(ctx, environment, integration)
}

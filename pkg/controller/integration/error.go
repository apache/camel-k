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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

	if kubernetes.IsConditionTrue(integration, v1.IntegrationConditionDeploymentAvailable) {
		deployment, err := kubernetes.GetDeployment(ctx, action.client, integration.Name, integration.Namespace)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil, err
		}

		// if the integration is in error phase, check if the corresponding pod is running ok, the user may have updated the integration.
		deployAvailable := false
		progressingOk := false
		for _, c := range deployment.Status.Conditions {
			// first, check if the container is in available state
			if c.Type == appsv1.DeploymentAvailable {
				deployAvailable = c.Status == corev1.ConditionTrue
			}
			// second, check the progressing and the reasons
			if c.Type == appsv1.DeploymentProgressing {
				progressingOk = c.Status == corev1.ConditionTrue && (c.Reason == "NewReplicaSetAvailable" || c.Reason == "ReplicaSetUpdated")
			}
		}
		if deployAvailable && progressingOk {
			availableCondition := v1.IntegrationCondition{
				Type:   v1.IntegrationConditionReady, 
				Status: corev1.ConditionTrue,
				Reason: v1.IntegrationConditionReplicaSetReadyReason,
			}
			integration.Status.SetConditions(availableCondition)
			integration.Status.Phase = v1.IntegrationPhaseRunning
			return integration, nil
		}
	}

	// TODO check also if deployment matches (e.g. replicas)
	return nil, nil
}

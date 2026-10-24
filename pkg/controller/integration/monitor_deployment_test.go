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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	appsv1 "k8s.io/api/apps/v1"
)

func TestDeploymentController_setReadyCondition(t *testing.T) {
	tests := []struct {
		name            string
		conditions      []appsv1.DeploymentCondition
		expectedPhase   v1.IntegrationPhase
		expectedReady   corev1.ConditionStatus
		expectedReason  string
		expectedMessage string
	}{
		{
			name: "replica failure",
			conditions: []appsv1.DeploymentCondition{
				{
					Type:    appsv1.DeploymentReplicaFailure,
					Status:  corev1.ConditionTrue,
					Reason:  "FailedCreate",
					Message: "Failed to create replicas",
				},
			},
			expectedPhase:   v1.IntegrationPhaseError,
			expectedReady:   corev1.ConditionFalse,
			expectedMessage: "Failed to create replicas",
		},
		{
			name: "progress deadline exceeded",
			conditions: []appsv1.DeploymentCondition{
				{
					Type:    appsv1.DeploymentProgressing,
					Status:  corev1.ConditionFalse,
					Reason:  "ProgressDeadlineExceeded",
					Message: "Deployment exceeded its progress deadline",
				},
			},
			expectedPhase:   v1.IntegrationPhaseError,
			expectedReady:   corev1.ConditionFalse,
			expectedMessage: "Deployment exceeded its progress deadline",
		},
		{
			name: "progressing with another reason",
			conditions: []appsv1.DeploymentCondition{
				{
					Type:    appsv1.DeploymentProgressing,
					Status:  corev1.ConditionFalse,
					Reason:  "ReplicaSetUpdated",
					Message: "ReplicaSet is being updated",
				},
				{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionTrue,
				},
			},
			expectedReady:   corev1.ConditionTrue,
			expectedReason:  v1.IntegrationConditionDeploymentReadyReason,
			expectedMessage: "Ready",
		},
		{
			name: "replica failure condition is false",
			conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentReplicaFailure,
					Status: corev1.ConditionFalse,
				},
				{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionTrue,
				},
			},
			expectedReady:   corev1.ConditionTrue,
			expectedReason:  v1.IntegrationConditionDeploymentReadyReason,
			expectedMessage: "Ready",
		},
		{
			name: "deployment available",
			conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionTrue,
				},
			},
			expectedReady:   corev1.ConditionTrue,
			expectedReason:  v1.IntegrationConditionDeploymentReadyReason,
			expectedMessage: "Ready",
		},
		{
			name:            "deployment has no conditions",
			expectedReady:   corev1.ConditionFalse,
			expectedReason:  v1.IntegrationConditionDeploymentReadyReason,
			expectedMessage: "Deployment does not have minimum availability.",
		},
		{
			name: "deployment not available",
			conditions: []appsv1.DeploymentCondition{
				{
					Type:    appsv1.DeploymentAvailable,
					Status:  corev1.ConditionFalse,
					Message: "custom message",
				},
			},
			expectedReady:   corev1.ConditionFalse,
			expectedReason:  v1.IntegrationConditionDeploymentReadyReason,
			expectedMessage: "custom message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Conditions: tt.conditions,
				},
			}

			integration := &v1.Integration{}

			controller := &deploymentController{
				obj: deployment,
			}

			controller.setReadyCondition(integration)
			assert.Equal(t, tt.expectedPhase, integration.Status.Phase)

			ready := integration.Status.GetCondition(v1.IntegrationConditionReady)
			require.NotNil(t, ready)
			assert.Equal(t, tt.expectedReady, corev1.ConditionStatus(ready.Status))

			if tt.expectedReason != "" {
				assert.Equal(t, tt.expectedReason, ready.Reason)
			}

			if tt.expectedMessage != "" {
				assert.Equal(t, tt.expectedMessage, ready.Message)
			}
		})
	}
}

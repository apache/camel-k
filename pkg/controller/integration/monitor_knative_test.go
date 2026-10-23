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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/duck/knative/apis"
	servingv1 "github.com/apache/camel-k/v2/pkg/apis/duck/knative/serving/v1"
)

func TestKnativeServiceController_setReadyCondition(t *testing.T) {
	tests := []struct {
		name            string
		condition       *metav1.Condition
		expectedPhase   v1.IntegrationPhase
		expectedReady   corev1.ConditionStatus
		expectedReason  string
		expectedMessage string
	}{
		{
			name: "revision failed",
			condition: &metav1.Condition{
				Type:    servingv1.ServiceConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  "RevisionFailed",
				Message: "Revision failed to become ready",
			},
			expectedPhase:   v1.IntegrationPhaseError,
			expectedReady:   corev1.ConditionFalse,
			expectedMessage: "Revision failed to become ready",
		},
		{
			name: "service ready",
			condition: &metav1.Condition{
				Type:   servingv1.ServiceConditionReady,
				Status: metav1.ConditionTrue,
			},
			expectedReady:   corev1.ConditionTrue,
			expectedReason:  v1.IntegrationConditionKnativeServiceReadyReason,
			expectedMessage: "Ready",
		},
		{
			name: "service not ready for another reason",
			condition: &metav1.Condition{
				Type:    servingv1.ServiceConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  "Deploying",
				Message: "Revision is deploying",
			},
			expectedReady:   corev1.ConditionTrue,
			expectedReason:  v1.IntegrationConditionKnativeServiceReadyReason,
			expectedMessage: "Ready",
		},
		{
			name:            "no ready condition",
			expectedReady:   corev1.ConditionTrue,
			expectedReason:  v1.IntegrationConditionKnativeServiceReadyReason,
			expectedMessage: "Ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ksvc := &servingv1.Service{}

			if tt.condition != nil {
				ksvc.Status = servingv1.ServiceStatus{
					Status: apis.Status{
						Conditions: []metav1.Condition{*tt.condition},
					},
				}
			}

			integration := &v1.Integration{}
			controller := &knativeServiceController{
				obj: ksvc,
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

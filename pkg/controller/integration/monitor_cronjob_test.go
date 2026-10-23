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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	batchv1 "k8s.io/api/batch/v1"
)

func TestCronJobController_setReadyCondition(t *testing.T) {
	now := metav1.Now()
	past := metav1.NewTime(now.Add(-1 * time.Hour))

	tests := []struct {
		name            string
		lastSchedule    *metav1.Time
		lastSuccessful  *metav1.Time
		activeJobs      []corev1.ObjectReference
		initialPhase    v1.IntegrationPhase
		initialReady    *v1.IntegrationCondition
		expectedPhase   v1.IntegrationPhase
		expectedReady   *corev1.ConditionStatus
		expectedReason  string
		expectedMessage string
	}{
		{
			name:          "no scheduled job",
			initialPhase:  v1.IntegrationPhaseNone,
			expectedPhase: v1.IntegrationPhaseNone,
		},
		{
			name:           "active job",
			lastSchedule:   &now,
			lastSuccessful: &past,
			activeJobs:     []corev1.ObjectReference{{Name: "active-job"}},
			initialPhase:   v1.IntegrationPhaseNone,
			expectedPhase:  v1.IntegrationPhaseNone,
		},
		{
			name:            "last scheduled job has never succeeded",
			lastSchedule:    &now,
			initialPhase:    v1.IntegrationPhaseNone,
			expectedPhase:   v1.IntegrationPhaseError,
			expectedReady:   conditionStatusPtr(corev1.ConditionFalse),
			expectedReason:  v1.IntegrationConditionLastJobFailedReason,
			expectedMessage: "The last scheduled job has not succeeded",
		},
		{
			name:            "last scheduled job failed",
			lastSchedule:    &now,
			lastSuccessful:  &past,
			initialPhase:    v1.IntegrationPhaseNone,
			expectedPhase:   v1.IntegrationPhaseError,
			expectedReady:   conditionStatusPtr(corev1.ConditionFalse),
			expectedReason:  v1.IntegrationConditionLastJobFailedReason,
			expectedMessage: "The last scheduled job has not succeeded",
		},
		{
			name:            "last scheduled job succeeded",
			lastSchedule:    &past,
			lastSuccessful:  &now,
			initialPhase:    v1.IntegrationPhaseNone,
			expectedPhase:   v1.IntegrationPhaseNone,
			expectedReady:   conditionStatusPtr(corev1.ConditionTrue),
			expectedReason:  v1.IntegrationConditionLastJobSucceededReason,
			expectedMessage: "The last scheduled job succeeded",
		},
		{
			name:            "last successful time equals last schedule time",
			lastSchedule:    &now,
			lastSuccessful:  &now,
			expectedPhase:   v1.IntegrationPhaseNone,
			expectedReady:   conditionStatusPtr(corev1.ConditionTrue),
			expectedReason:  v1.IntegrationConditionLastJobSucceededReason,
			expectedMessage: "The last scheduled job succeeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cronJob := &batchv1.CronJob{
				Status: batchv1.CronJobStatus{
					LastScheduleTime:   tt.lastSchedule,
					LastSuccessfulTime: tt.lastSuccessful,
					Active:             tt.activeJobs,
				},
			}

			integration := &v1.Integration{
				Status: v1.IntegrationStatus{
					Phase: tt.initialPhase,
				},
			}

			if tt.initialReady != nil {
				integration.SetReadyCondition(
					corev1.ConditionStatus(tt.initialReady.Status),
					tt.initialReady.Reason,
					tt.initialReady.Message,
				)
			}

			controller := &cronJobController{obj: cronJob}
			controller.setReadyCondition(integration)
			assert.Equal(t, tt.expectedPhase, integration.Status.Phase)

			ready := integration.Status.GetCondition(v1.IntegrationConditionReady)
			if tt.expectedReady == nil {
				assert.Nil(t, ready)
				return
			}

			require.NotNil(t, ready)
			assert.Equal(t, *tt.expectedReady, corev1.ConditionStatus(ready.Status))
			assert.Equal(t, tt.expectedReason, ready.Reason)
			assert.Equal(t, tt.expectedMessage, ready.Message)
		})
	}
}

func conditionStatusPtr(status corev1.ConditionStatus) *corev1.ConditionStatus {
	return &status
}

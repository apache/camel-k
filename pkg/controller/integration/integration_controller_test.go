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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestIsIntegrationUpdated(t *testing.T) {
	now := metav1.Now()

	tests := []struct {
		name     string
		it       *v1.Integration
		previous *v1.IntegrationCondition
		next     *v1.IntegrationCondition
		expected bool
	}{
		{
			name: "should return true when transitioning to ready",
			it: &v1.Integration{
				Status: v1.IntegrationStatus{
					InitializationTimestamp: &now,
				},
			},
			previous: nil,
			next: &v1.IntegrationCondition{
				Status:          corev1.ConditionTrue,
				FirstTruthyTime: &now,
			},
			expected: true,
		},
		{
			name: "should return false when no InitializationTimestamp",
			it: &v1.Integration{
				Status: v1.IntegrationStatus{},
			},
			previous: nil,
			next: &v1.IntegrationCondition{
				Status:          corev1.ConditionTrue,
				FirstTruthyTime: &now,
			},
			expected: false,
		},
		{
			name: "should return false when already ready",
			it: &v1.Integration{
				Status: v1.IntegrationStatus{
					InitializationTimestamp: &now,
				},
			},
			previous: &v1.IntegrationCondition{
				Status:          corev1.ConditionTrue,
				FirstTruthyTime: &now,
			},
			next: &v1.IntegrationCondition{
				Status:          corev1.ConditionTrue,
				FirstTruthyTime: &now,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isIntegrationUpdated(tt.it, tt.previous, tt.next)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadinessTimestampCalculation(t *testing.T) {
	initTime := metav1.NewTime(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC))
	deployTime := metav1.NewTime(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	readyTime := metav1.NewTime(time.Date(2025, 1, 1, 12, 5, 0, 0, time.UTC))
	zeroTime := metav1.Time{}

	tests := []struct {
		name                    string
		initializationTimestamp *metav1.Time
		deploymentTimestamp     *metav1.Time
		firstTruthyTime         *metav1.Time
		expectedDuration        time.Duration
		description             string
	}{
		{
			name:                    "normal build - uses InitializationTimestamp",
			initializationTimestamp: &initTime,
			deploymentTimestamp:     nil,
			firstTruthyTime:         &readyTime,
			expectedDuration:        2*time.Hour + 5*time.Minute,
			description:             "Without DeploymentTimestamp, should use InitializationTimestamp",
		},
		{
			name:                    "dry build - uses DeploymentTimestamp",
			initializationTimestamp: &initTime,
			deploymentTimestamp:     &deployTime,
			firstTruthyTime:         &readyTime,
			expectedDuration:        5 * time.Minute,
			description:             "With DeploymentTimestamp, should use it instead of InitializationTimestamp",
		},
		{
			name:                    "DeploymentTimestamp is zero - falls back to InitializationTimestamp",
			initializationTimestamp: &initTime,
			deploymentTimestamp:     &zeroTime,
			firstTruthyTime:         &readyTime,
			expectedDuration:        2*time.Hour + 5*time.Minute,
			description:             "Zero DeploymentTimestamp should fall back to InitializationTimestamp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := &v1.Integration{
				Status: v1.IntegrationStatus{
					InitializationTimestamp: tt.initializationTimestamp,
					DeploymentTimestamp:     tt.deploymentTimestamp,
				},
			}

			startTime := it.Status.InitializationTimestamp.Time
			if it.Status.DeploymentTimestamp != nil && !it.Status.DeploymentTimestamp.IsZero() {
				startTime = it.Status.DeploymentTimestamp.Time
			}
			duration := tt.firstTruthyTime.Sub(startTime)
			assert.Equal(t, tt.expectedDuration, duration, tt.description)
		})
	}
}

func TestDeploymentTimestampIsSet(t *testing.T) {
	tests := []struct {
		name                      string
		hasDontRunAnnotation      bool
		expectedPhase             v1.IntegrationPhase
		expectDeploymentTimestamp bool
	}{
		{
			name:                      "normal build sets DeploymentTimestamp",
			hasDontRunAnnotation:      false,
			expectedPhase:             v1.IntegrationPhaseDeploying,
			expectDeploymentTimestamp: true,
		},
		{
			name:                      "dry build does not set DeploymentTimestamp yet",
			hasDontRunAnnotation:      true,
			expectedPhase:             v1.IntegrationPhaseBuildComplete,
			expectDeploymentTimestamp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := &v1.Integration{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Status: v1.IntegrationStatus{},
			}

			if tt.hasDontRunAnnotation {
				it.Annotations[v1.IntegrationDontRunAfterBuildAnnotation] = v1.IntegrationDontRunAfterBuildAnnotationTrueValue
			}

			if it.Annotations[v1.IntegrationDontRunAfterBuildAnnotation] == v1.IntegrationDontRunAfterBuildAnnotationTrueValue {
				it.Status.Phase = v1.IntegrationPhaseBuildComplete
			} else {
				now := metav1.Now().Rfc3339Copy()
				it.Status.DeploymentTimestamp = &now
				it.Status.Phase = v1.IntegrationPhaseDeploying
			}

			assert.Equal(t, tt.expectedPhase, it.Status.Phase)
			if tt.expectDeploymentTimestamp {
				assert.NotNil(t, it.Status.DeploymentTimestamp)
				assert.False(t, it.Status.DeploymentTimestamp.IsZero())
			} else {
				assert.Nil(t, it.Status.DeploymentTimestamp)
			}
		})
	}
}

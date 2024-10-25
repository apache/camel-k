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

package trait

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

const (
	TraitConfigurationReason = "TraitConfiguration"
	userDisabledMessage      = "explicitly disabled by the user"
	userEnabledMessage       = "explicitly enabled by the user"
	platformDisabledMessage  = "explicitly disabled by the platform"
)

// TraitCondition is used to get all information/warning about a trait configuration.
// It should either use an IntegrationConditionType or IntegrationKitConditionType.
type TraitCondition struct {
	traitID                  string
	integrationConditionType v1.IntegrationConditionType
	conditionStatus          corev1.ConditionStatus
	message                  string
	reason                   string
}

func NewIntegrationCondition(traitID string, ict v1.IntegrationConditionType, cs corev1.ConditionStatus, reason, message string) *TraitCondition {
	return &TraitCondition{
		traitID:                  traitID,
		integrationConditionType: ict,
		conditionStatus:          cs,
		reason:                   reason,
		message:                  message,
	}
}

func NewIntegrationConditionUserDisabled(traitID string) *TraitCondition {
	return NewIntegrationCondition(traitID, v1.IntegrationConditionTraitInfo, corev1.ConditionTrue, TraitConfigurationReason, userDisabledMessage)
}

func NewIntegrationConditionPlatformDisabledWithMessage(traitID string, message string) *TraitCondition {
	return NewIntegrationCondition(traitID, v1.IntegrationConditionTraitInfo, corev1.ConditionTrue, TraitConfigurationReason, fmt.Sprintf("%s: %s", platformDisabledMessage, message))
}

func (tc *TraitCondition) integrationCondition() (v1.IntegrationConditionType, corev1.ConditionStatus, string, string) {
	return v1.IntegrationConditionType(tc.typeForCondition()),
		tc.conditionStatus,
		tc.reason,
		tc.message
}

func (tc *TraitCondition) integrationKitCondition() (v1.IntegrationKitConditionType, corev1.ConditionStatus, string, string) {
	return v1.IntegrationKitConditionType(tc.typeForCondition()),
		tc.conditionStatus,
		tc.reason,
		tc.message
}

func (tc *TraitCondition) typeForCondition() string {
	conditionType := string(tc.integrationConditionType)
	if conditionType == "TraitInfo" {
		conditionType = fmt.Sprintf("%s%s", tc.traitID, tc.integrationConditionType)
	}
	return conditionType
}

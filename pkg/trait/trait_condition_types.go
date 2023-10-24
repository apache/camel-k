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
	traitConfigurationMessage = "Trait configuration"
	userDisabledMessage       = "explicitly disabled by the user"
	platformDisabledMessage   = "explicitly disabled by the platform"
)

// TraitCondition is used to get all information/warning about a trait configuration.
// It should either use an IntegrationConditionType or IntegrationKitConditionType.
type TraitCondition struct {
	integrationConditionType    v1.IntegrationConditionType
	integrationKitConditionType v1.IntegrationKitConditionType
	conditionStatus             corev1.ConditionStatus
	message                     string
	reason                      string
}

func NewIntegrationCondition(ict v1.IntegrationConditionType, cs corev1.ConditionStatus, message, reason string) *TraitCondition {
	return &TraitCondition{
		integrationConditionType: ict,
		conditionStatus:          cs,
		message:                  message,
		reason:                   reason,
	}
}

func NewIntegrationConditionUserDisabled() *TraitCondition {
	return NewIntegrationCondition(v1.IntegrationConditionTraitInfo, corev1.ConditionTrue, traitConfigurationMessage, userDisabledMessage)
}

func newIntegrationConditionPlatformDisabledWithReason(reason string) *TraitCondition {
	return NewIntegrationCondition(v1.IntegrationConditionTraitInfo, corev1.ConditionTrue, traitConfigurationMessage, fmt.Sprintf("%s: %s", platformDisabledMessage, reason))
}

func (tc *TraitCondition) integrationCondition() (v1.IntegrationConditionType, corev1.ConditionStatus, string, string) {
	return tc.integrationConditionType, tc.conditionStatus, tc.message, tc.reason
}

func (tc *TraitCondition) integrationKitCondition() (v1.IntegrationKitConditionType, corev1.ConditionStatus, string, string) {
	return tc.integrationKitConditionType, tc.conditionStatus, tc.message, tc.reason
}

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

package v1

import (
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewIntegrationPlatformList --
func NewIntegrationPlatformList() IntegrationPlatformList {
	return IntegrationPlatformList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationPlatformKind,
		},
	}
}

// NewIntegrationPlatform --
func NewIntegrationPlatform(namespace string, name string) IntegrationPlatform {
	return IntegrationPlatform{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationPlatformKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// TraitProfileByName returns the trait profile corresponding to the given name (case insensitive)
func TraitProfileByName(name string) TraitProfile {
	for _, p := range AllTraitProfiles {
		if strings.EqualFold(name, string(p)) {
			return p
		}
	}
	return ""
}

// Configurations --
func (in *IntegrationPlatformSpec) Configurations() []ConfigurationSpec {
	if in == nil {
		return []ConfigurationSpec{}
	}

	return in.Configuration
}

// SetOperatorID sets the given operator id as an annotation
func (in *IntegrationPlatform) SetOperatorID(operatorID string) {
	SetAnnotation(&in.ObjectMeta, OperatorIDAnnotation, operatorID)
}

// Configurations --
func (in *IntegrationPlatform) Configurations() []ConfigurationSpec {
	if in == nil {
		return []ConfigurationSpec{}
	}

	if len(in.Status.Configuration) > 0 {
		return in.Status.Configuration
	}

	return in.Spec.Configuration
}

// AddConfiguration --
func (in *IntegrationPlatform) AddConfiguration(confType string, confValue string) {
	in.Spec.Configuration = append(in.Spec.Configuration, ConfigurationSpec{
		Type:  confType,
		Value: confValue,
	})
}

// GetActualValue can be used to extract information the platform spec or its derived config in the status
func (in *IntegrationPlatform) GetActualValue(extractor func(spec IntegrationPlatformSpec) string) string {
	res := extractor(in.Status.IntegrationPlatformSpec)
	if res == "" {
		res = extractor(in.Spec)
	}
	return res
}

// ResyncStatusFullConfig copies the spec configuration into the status field.
func (in *IntegrationPlatform) ResyncStatusFullConfig() {
	cl := in.Spec.DeepCopy()
	in.Status.IntegrationPlatformSpec = *cl
}

// GetCondition returns the condition with the provided type.
func (in *IntegrationPlatformStatus) GetCondition(condType IntegrationPlatformConditionType) *IntegrationPlatformCondition {
	for i := range in.Conditions {
		c := in.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetCondition --
func (in *IntegrationPlatformStatus) SetCondition(condType IntegrationPlatformConditionType, status corev1.ConditionStatus, reason string, message string) {
	in.SetConditions(IntegrationPlatformCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

// SetErrorCondition --
func (in *IntegrationPlatformStatus) SetErrorCondition(condType IntegrationPlatformConditionType, reason string, err error) {
	in.SetConditions(IntegrationPlatformCondition{
		Type:               condType,
		Status:             corev1.ConditionFalse,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            err.Error(),
	})
}

// SetConditions updates the resource to include the provided conditions.
//
// If a condition that we are about to add already exists and has the same status and
// reason then we are not going to update.
func (in *IntegrationPlatformStatus) SetConditions(conditions ...IntegrationPlatformCondition) {
	for _, condition := range conditions {
		if condition.LastUpdateTime.IsZero() {
			condition.LastUpdateTime = metav1.Now()
		}
		if condition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = metav1.Now()
		}

		currentCond := in.GetCondition(condition.Type)

		if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
			return
		}
		// Do not update lastTransitionTime if the status of the condition doesn't change.
		if currentCond != nil && currentCond.Status == condition.Status {
			condition.LastTransitionTime = currentCond.LastTransitionTime
		}

		in.RemoveCondition(condition.Type)
		in.Conditions = append(in.Conditions, condition)
	}
}

// RemoveCondition removes the resource condition with the provided type.
func (in *IntegrationPlatformStatus) RemoveCondition(condType IntegrationPlatformConditionType) {
	newConditions := in.Conditions[:0]
	for _, c := range in.Conditions {
		if c.Type != condType {
			newConditions = append(newConditions, c)
		}
	}

	in.Conditions = newConditions
}

// IsOptionEnabled tells if provided option key is present in PublishStrategyOptions and enabled
func (b IntegrationPlatformBuildSpec) IsOptionEnabled(option string) bool {
	//Key defined in builder/kaniko.go
	if enabled, ok := b.PublishStrategyOptions[option]; ok {
		res, err := strconv.ParseBool(enabled)
		if err != nil {
			return false
		}
		return res
	}
	return false
}

// GetTimeout returns the specified duration or a default one
func (b IntegrationPlatformBuildSpec) GetTimeout() metav1.Duration {
	if b.Timeout == nil {
		return metav1.Duration{}
	}
	return *b.Timeout
}

var _ ResourceCondition = IntegrationPlatformCondition{}

// GetConditions --
func (in *IntegrationPlatformStatus) GetConditions() []ResourceCondition {
	res := make([]ResourceCondition, 0, len(in.Conditions))
	for _, c := range in.Conditions {
		res = append(res, c)
	}
	return res
}

// GetType --
func (c IntegrationPlatformCondition) GetType() string {
	return string(c.Type)
}

// GetStatus --
func (c IntegrationPlatformCondition) GetStatus() corev1.ConditionStatus {
	return c.Status
}

// GetLastUpdateTime --
func (c IntegrationPlatformCondition) GetLastUpdateTime() metav1.Time {
	return c.LastUpdateTime
}

// GetLastTransitionTime --
func (c IntegrationPlatformCondition) GetLastTransitionTime() metav1.Time {
	return c.LastTransitionTime
}

// GetReason --
func (c IntegrationPlatformCondition) GetReason() string {
	return c.Reason
}

// GetMessage --
func (c IntegrationPlatformCondition) GetMessage() string {
	return c.Message
}

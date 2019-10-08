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

package v1alpha1

import (
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
	for _, p := range allTraitProfiles {
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

// Configurations --
func (in *IntegrationPlatform) Configurations() []ConfigurationSpec {
	if in == nil {
		return []ConfigurationSpec{}
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

// IsKanikoCacheEnabled tells if the KanikoCache is enabled on the integration platform build spec
func (b IntegrationPlatformBuildSpec) IsKanikoCacheEnabled() bool {
	if b.KanikoBuildCache == nil {
		// Cache is enabled unless explicitly disabled
		return true
	}
	return *b.KanikoBuildCache
}

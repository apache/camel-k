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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewIntegrationPlatformList --.
func NewIntegrationPlatformList() IntegrationPlatformList {
	return IntegrationPlatformList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationPlatformKind,
		},
	}
}

// NewIntegrationPlatform --.
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

// Configurations --.
func (in *IntegrationPlatformSpec) Configurations() []ConfigurationSpec {
	if in == nil {
		return []ConfigurationSpec{}
	}

	return in.Configuration
}

// GetOperatorID returns the operator id as an annotation.
func (in *IntegrationPlatform) GetOperatorID() string {
	return GetAnnotation(OperatorIDAnnotation, in)
}

// SetOperatorID sets the given operator id as an annotation.
func (in *IntegrationPlatform) SetOperatorID(operatorID string) {
	SetAnnotation(&in.ObjectMeta, OperatorIDAnnotation, operatorID)
}

// Configurations --.
func (in *IntegrationPlatform) Configurations() []ConfigurationSpec {
	if in == nil {
		return []ConfigurationSpec{}
	}

	if len(in.Status.Configuration) > 0 {
		return in.Status.Configuration
	}

	return in.Spec.Configuration
}

// AddConfiguration --.
func (in *IntegrationPlatform) AddConfiguration(confType string, confValue string) {
	in.Spec.Configuration = append(in.Spec.Configuration, ConfigurationSpec{
		Type:  confType,
		Value: confValue,
	})
}

// GetActualValue can be used to extract information the platform spec or its derived config in the status.
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
func (in *IntegrationPlatformStatus) GetCondition(condType IntegrationPlatformConditionType) *metav1.Condition {
	return meta.FindStatusCondition(in.Conditions, string(condType))
}

// SetCondition sets the condition with the given status, reason, and message.
func (in *IntegrationPlatformStatus) SetCondition(condType IntegrationPlatformConditionType, status corev1.ConditionStatus, reason string, message string) {
	in.SetConditions(metav1.Condition{
		Type:    string(condType),
		Status:  metav1.ConditionStatus(status),
		Reason:  reason,
		Message: message,
	})
}

// SetErrorCondition sets the condition with the given reason and error message.
func (in *IntegrationPlatformStatus) SetErrorCondition(condType IntegrationPlatformConditionType, reason string, err error) {
	in.SetConditions(metav1.Condition{
		Type:    string(condType),
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: err.Error(),
	})
}

// SetConditions updates the resource to include the provided conditions.
//
// If a condition that we are about to add already exists and has the same status and
// reason then we are not going to update.
func (in *IntegrationPlatformStatus) SetConditions(conditions ...metav1.Condition) {
	for _, condition := range conditions {
		if condition.Reason == "" {
			// Reason is a required field for metav1.Condition.
			condition.Reason = string(condition.Status)
		}
		if condition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = metav1.Now()
		}

		meta.SetStatusCondition(&in.Conditions, condition)
	}
}

// RemoveCondition removes the resource condition with the provided type.
func (in *IntegrationPlatformStatus) RemoveCondition(condType IntegrationPlatformConditionType) {
	meta.RemoveStatusCondition(&in.Conditions, string(condType))
}

// GetTimeout returns the specified duration or a default one.
func (b *IntegrationPlatformBuildSpec) GetTimeout() metav1.Duration {
	if b.Timeout == nil {
		return metav1.Duration{}
	}

	return *b.Timeout
}

// GetConditions --.
func (in *IntegrationPlatformStatus) GetConditions() []ResourceCondition {
	res := make([]ResourceCondition, 0, len(in.Conditions))
	for i := range in.Conditions {
		res = append(res, (*conditionAdapter)(&in.Conditions[i]))
	}

	return res
}

// GetType --.

// GetStatus --.

// GetLastUpdateTime --.

// GetLastTransitionTime --.

// GetReason --.

// GetMessage --.

// Validate checks the strategy is supported.
func (b IntegrationPlatformBuildPublishStrategy) Validate() error {
	switch b {
	case IntegrationPlatformBuildPublishStrategyS2I, IntegrationPlatformBuildPublishStrategyJib:
		return nil
	default:
		return fmt.Errorf("invalid IntegrationPlatformBuildPublishStrategy: %q", b)
	}
}

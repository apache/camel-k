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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewIntegrationProfile --.
func NewIntegrationProfile(namespace string, name string) IntegrationProfile {
	return IntegrationProfile{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationProfileKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// SetOperatorID sets the given operator id as an annotation.
func (in *IntegrationProfile) SetOperatorID(operatorID string) {
	SetAnnotation(&in.ObjectMeta, OperatorIDAnnotation, operatorID)
}

// GetActualValue can be used to extract information the platform spec or its derived config in the status.
func (in *IntegrationProfile) GetActualValue(extractor func(spec IntegrationProfileSpec) string) string {
	res := extractor(in.Status.IntegrationProfileSpec)
	if res == "" {
		res = extractor(in.Spec)
	}
	return res
}

// ResyncStatusFullConfig copies the spec configuration into the status field.
func (in *IntegrationProfile) ResyncStatusFullConfig() {
	cl := in.Spec.DeepCopy()
	in.Status.IntegrationProfileSpec = *cl
}

// GetCondition returns the condition with the provided type.
func (in *IntegrationProfileStatus) GetCondition(condType IntegrationProfileConditionType) *IntegrationProfileCondition {
	for i := range in.Conditions {
		c := in.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetCondition sets the condition with the given status, reason, and message.
func (in *IntegrationProfileStatus) SetCondition(condType IntegrationProfileConditionType, status corev1.ConditionStatus, reason string, message string) {
	in.SetConditions(IntegrationProfileCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

// SetErrorCondition sets the condition with the given reason and error message.
func (in *IntegrationProfileStatus) SetErrorCondition(condType IntegrationProfileConditionType, reason string, err error) {
	in.SetConditions(IntegrationProfileCondition{
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
func (in *IntegrationProfileStatus) SetConditions(conditions ...IntegrationProfileCondition) {
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
func (in *IntegrationProfileStatus) RemoveCondition(condType IntegrationProfileConditionType) {
	newConditions := in.Conditions[:0]
	for _, c := range in.Conditions {
		if c.Type != condType {
			newConditions = append(newConditions, c)
		}
	}

	in.Conditions = newConditions
}

// GetTimeout returns the specified duration or a default one.
func (b IntegrationProfileBuildSpec) GetTimeout() metav1.Duration {
	if b.Timeout == nil {
		return metav1.Duration{}
	}
	return *b.Timeout
}

var _ ResourceCondition = IntegrationProfileCondition{}

// GetConditions --.
func (in *IntegrationProfileStatus) GetConditions() []ResourceCondition {
	res := make([]ResourceCondition, 0, len(in.Conditions))
	for _, c := range in.Conditions {
		res = append(res, c)
	}
	return res
}

// GetType --.
func (c IntegrationProfileCondition) GetType() string {
	return string(c.Type)
}

// GetStatus --.
func (c IntegrationProfileCondition) GetStatus() corev1.ConditionStatus {
	return c.Status
}

// GetLastUpdateTime --.
func (c IntegrationProfileCondition) GetLastUpdateTime() metav1.Time {
	return c.LastUpdateTime
}

// GetLastTransitionTime --.
func (c IntegrationProfileCondition) GetLastTransitionTime() metav1.Time {
	return c.LastTransitionTime
}

// GetReason --.
func (c IntegrationProfileCondition) GetReason() string {
	return c.Reason
}

// GetMessage --.
func (c IntegrationProfileCondition) GetMessage() string {
	return c.Message
}

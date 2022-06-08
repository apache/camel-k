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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewIntegrationKit(namespace string, name string) *IntegrationKit {
	return &IntegrationKit{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationKitKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func NewIntegrationKitList() IntegrationKitList {
	return IntegrationKitList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationKitKind,
		},
	}
}

func (in *IntegrationKitSpec) Configurations() []ConfigurationSpec {
	if in == nil {
		return []ConfigurationSpec{}
	}

	return in.Configuration
}

// SetOperatorID sets the given operator id as an annotation
func (in *IntegrationKit) SetOperatorID(operatorID string) {
	if in.Annotations == nil {
		in.Annotations = make(map[string]string)
	}

	in.Annotations[OperatorIDAnnotation] = operatorID
}

func (in *IntegrationKit) Configurations() []ConfigurationSpec {
	if in == nil {
		return []ConfigurationSpec{}
	}

	return in.Spec.Configuration
}

func (in *IntegrationKit) SetIntegrationPlatform(platform *IntegrationPlatform) {
	cs := corev1.ConditionTrue

	if platform.Status.Phase != IntegrationPlatformPhaseReady {
		cs = corev1.ConditionFalse
	}

	var message string
	if platform.Name != "" {
		message = "IntegrationPlatform (" + platform.Name + ")"
	}

	in.Status.SetCondition(IntegrationKitConditionPlatformAvailable, cs, IntegrationKitConditionPlatformAvailableReason, message)
	in.Status.Platform = platform.Name
}

func (in *IntegrationKit) HasHigherPriorityThan(kit *IntegrationKit) bool {
	p1 := 0
	p2 := 0
	if l, ok := in.Labels[IntegrationKitPriorityLabel]; ok {
		if p, err := strconv.Atoi(l); err == nil {
			p1 = p
		}
	}
	if l, ok := kit.Labels[IntegrationKitPriorityLabel]; ok {
		if p, err := strconv.Atoi(l); err == nil {
			p2 = p
		}
	}
	return p1 > p2
}

// GetCondition returns the condition with the provided type.
func (in *IntegrationKitStatus) GetCondition(condType IntegrationKitConditionType) *IntegrationKitCondition {
	for i := range in.Conditions {
		c := in.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

func (in *IntegrationKitStatus) SetCondition(condType IntegrationKitConditionType, status corev1.ConditionStatus, reason string, message string) {
	in.SetConditions(IntegrationKitCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

func (in *IntegrationKitStatus) SetErrorCondition(condType IntegrationKitConditionType, reason string, err error) {
	in.SetConditions(IntegrationKitCondition{
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
func (in *IntegrationKitStatus) SetConditions(conditions ...IntegrationKitCondition) {
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
func (in *IntegrationKitStatus) RemoveCondition(condType IntegrationKitConditionType) {
	newConditions := in.Conditions[:0]
	for _, c := range in.Conditions {
		if c.Type != condType {
			newConditions = append(newConditions, c)
		}
	}

	in.Conditions = newConditions
}

var _ ResourceCondition = IntegrationKitCondition{}

func (in *IntegrationKitStatus) GetConditions() []ResourceCondition {
	res := make([]ResourceCondition, 0, len(in.Conditions))
	for _, c := range in.Conditions {
		res = append(res, c)
	}
	return res
}

func (c IntegrationKitCondition) GetType() string {
	return string(c.Type)
}

func (c IntegrationKitCondition) GetStatus() corev1.ConditionStatus {
	return c.Status
}

func (c IntegrationKitCondition) GetLastUpdateTime() metav1.Time {
	return c.LastUpdateTime
}

func (c IntegrationKitCondition) GetLastTransitionTime() metav1.Time {
	return c.LastTransitionTime
}

func (c IntegrationKitCondition) GetReason() string {
	return c.Reason
}

func (c IntegrationKitCondition) GetMessage() string {
	return c.Message
}

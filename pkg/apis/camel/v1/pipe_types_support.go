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
	"bytes"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetConditions --.
func (in *PipeStatus) GetConditions() []ResourceCondition {
	res := make([]ResourceCondition, 0, len(in.Conditions))
	for _, c := range in.Conditions {
		res = append(res, c)
	}
	return res
}

// GetType --.
func (c PipeCondition) GetType() string {
	return string(c.Type)
}

// GetStatus --.
func (c PipeCondition) GetStatus() corev1.ConditionStatus {
	return c.Status
}

// GetLastUpdateTime --.
func (c PipeCondition) GetLastUpdateTime() metav1.Time {
	return c.LastUpdateTime
}

// GetLastTransitionTime --.
func (c PipeCondition) GetLastTransitionTime() metav1.Time {
	return c.LastTransitionTime
}

// GetReason --.
func (c PipeCondition) GetReason() string {
	return c.Reason
}

// GetMessage --.
func (c PipeCondition) GetMessage() string {
	return c.Message
}

// SetOperatorID sets the given operator id as an annotation.
func (in *Pipe) SetOperatorID(operatorID string) {
	SetAnnotation(&in.ObjectMeta, OperatorIDAnnotation, operatorID)
}

// GetCondition returns the condition with the provided type.
func (in *PipeStatus) GetCondition(condType PipeConditionType) *PipeCondition {
	for i := range in.Conditions {
		c := in.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetCondition --.
func (in *PipeStatus) SetCondition(condType PipeConditionType, status corev1.ConditionStatus, reason string, message string) {
	in.SetConditions(PipeCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

// SetErrorCondition --.
func (in *PipeStatus) SetErrorCondition(condType PipeConditionType, reason string, err error) {
	in.SetConditions(PipeCondition{
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
func (in *PipeStatus) SetConditions(conditions ...PipeCondition) {
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
func (in *PipeStatus) RemoveCondition(condType PipeConditionType) {
	newConditions := in.Conditions[:0]
	for _, c := range in.Conditions {
		if c.Type != condType {
			newConditions = append(newConditions, c)
		}
	}

	in.Conditions = newConditions
}

// GetPropertyMap returns the EndpointProperties as map.
func (p *EndpointProperties) GetPropertyMap() (map[string]string, error) {
	if p == nil {
		return nil, nil
	}
	if len(p.RawMessage) == 0 {
		return nil, nil
	}

	// Convert json property values to objects before getting their string representation.
	var props map[string]interface{}
	d := json.NewDecoder(bytes.NewReader(p.RawMessage))
	d.UseNumber()
	if err := d.Decode(&props); err != nil {
		return nil, err
	}
	stringProps := make(map[string]string, len(props))
	for k, v := range props {
		stringProps[k] = fmt.Sprintf("%v", v)
	}
	return stringProps, nil
}

// NewPipe --.
func NewPipe(namespace string, name string) Pipe {
	return Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// NewPipeList --.
func NewPipeList() PipeList {
	return PipeList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       PipeKind,
		},
	}
}

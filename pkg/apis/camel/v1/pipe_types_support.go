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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetConditions --.
func (in *PipeStatus) GetConditions() []ResourceCondition {
	res := make([]ResourceCondition, 0, len(in.Conditions))
	for i := range in.Conditions {
		res = append(res, (*conditionAdapter)(&in.Conditions[i]))
	}

	return res
}

// SetOperatorID sets the given operator id as an annotation.
func (in *Pipe) SetOperatorID(operatorID string) {
	SetAnnotation(&in.ObjectMeta, OperatorIDAnnotation, operatorID)
}

// SetTraits add the Trait specification to the Pipe.
func (in *Pipe) SetTraits(traits *Traits) {
	in.Spec.Traits = traits
}

// GetCondition returns the condition with the provided type.
func (in *PipeStatus) GetCondition(condType PipeConditionType) *metav1.Condition {
	return meta.FindStatusCondition(in.Conditions, string(condType))
}

// SetCondition sets the condition with the given status, reason, and message.
func (in *PipeStatus) SetCondition(condType PipeConditionType, status corev1.ConditionStatus, reason string, message string) {
	in.SetConditions(metav1.Condition{
		Type:    string(condType),
		Status:  metav1.ConditionStatus(status),
		Reason:  reason,
		Message: message,
	})
}

// SetErrorCondition sets the condition with the given reason and error message.
func (in *PipeStatus) SetErrorCondition(condType PipeConditionType, reason string, err error) {
	in.SetConditions(metav1.Condition{
		Type:    string(condType),
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: err.Error(),
	})
}

// SetConditions updates the resource conditions using metav1.Condition semantics.
func (in *PipeStatus) SetConditions(conditions ...metav1.Condition) {
	for _, condition := range conditions {
		if condition.Reason == "" {
			condition.Reason = string(condition.Status)
		}
		if condition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = metav1.Now()
		}
		meta.SetStatusCondition(&in.Conditions, condition)
	}
}

// RemoveCondition removes the resource condition with the provided type.
func (in *PipeStatus) RemoveCondition(condType PipeConditionType) {
	newConditions := in.Conditions[:0]
	for _, c := range in.Conditions {
		if c.Type != string(condType) {
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

	// Convert json property values to objects before getting their string representation
	var props map[string]any
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

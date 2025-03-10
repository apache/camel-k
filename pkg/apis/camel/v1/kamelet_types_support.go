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
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetConditions --.
func (in *KameletStatus) GetConditions() []ResourceCondition {
	res := make([]ResourceCondition, 0, len(in.Conditions))
	for _, c := range in.Conditions {
		res = append(res, &c)
	}
	return res
}

// GetType --.
func (c *KameletCondition) GetType() string {
	return string(c.Type)
}

// GetStatus --.
func (c *KameletCondition) GetStatus() corev1.ConditionStatus {
	return c.Status
}

// GetLastUpdateTime --.
func (c *KameletCondition) GetLastUpdateTime() metav1.Time {
	return c.LastUpdateTime
}

// GetLastTransitionTime --.
func (c *KameletCondition) GetLastTransitionTime() metav1.Time {
	return c.LastTransitionTime
}

// GetReason --.
func (c *KameletCondition) GetReason() string {
	return c.Reason
}

// GetMessage --.
func (c *KameletCondition) GetMessage() string {
	return c.Message
}

// GetCondition returns the condition with the provided type.
func (in *KameletStatus) GetCondition(condType KameletConditionType) *KameletCondition {
	for i := range in.Conditions {
		c := in.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetCondition --.
func (in *KameletStatus) SetCondition(condType KameletConditionType, status corev1.ConditionStatus, reason string, message string) {
	in.SetConditions(KameletCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

// SetErrorCondition --.
func (in *KameletStatus) SetErrorCondition(condType KameletConditionType, reason string, err error) {
	in.SetConditions(KameletCondition{
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
func (in *KameletStatus) SetConditions(conditions ...KameletCondition) {
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
func (in *KameletStatus) RemoveCondition(condType KameletConditionType) {
	newConditions := in.Conditions[:0]
	for _, c := range in.Conditions {
		if c.Type != condType {
			newConditions = append(newConditions, c)
		}
	}

	in.Conditions = newConditions
}

// SortedDefinitionPropertiesKeys returns the sorted keys of the Kamelet definition properties.
func (k *Kamelet) SortedDefinitionPropertiesKeys() []string {
	if k.Spec.Definition == nil {
		return []string{}
	}
	props := k.Spec.Definition.Properties
	if len(props) == 0 {
		return []string{}
	}
	res := make([]string, len(props))
	i := 0
	for key := range props {
		res[i] = key
		i++
	}
	sort.Strings(res)
	return res
}

// SortedTypesKeys returns the sorted keys of the Kamelet spec types.
func (k *Kamelet) SortedTypesKeys() []TypeSlot {
	types := k.Spec.DataTypes
	if len(types) == 0 {
		return []TypeSlot{}
	}
	strs := make([]string, len(types))
	i := 0
	for key := range types {
		strs[i] = string(key)
		i++
	}
	sort.Strings(strs)
	res := make([]TypeSlot, len(types))
	for i, s := range strs {
		res[i] = TypeSlot(s)
	}
	return res
}

// IsBundled returns true if the Kamelet is coming from a pre-bundled installation.
func (k *Kamelet) IsBundled() bool {
	return k.Labels != nil && k.Labels[KameletBundledLabel] == "true"
}

func ValidKameletName(name string) bool {
	return !reservedKameletNames[name]
}

// NewKamelet creates a new Kamelet.
func NewKamelet(namespace string, name string) Kamelet {
	return Kamelet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       KameletKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// NewKameletList creates a new list of Kamelets.
func NewKameletList() KameletList {
	return KameletList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       KameletKind,
		},
	}
}

// SetOperatorID sets the given operator id as an annotation.
func (k *Kamelet) SetOperatorID(operatorID string) {
	SetAnnotation(&k.ObjectMeta, OperatorIDAnnotation, operatorID)
}

// CloneWithVersion clones a Kamelet and set the main specification with any version provided.
// It also changes the name adding a suffix with the version provided.
func (k *Kamelet) CloneWithVersion(version string) (*Kamelet, error) {
	clone := k.DeepCopy()
	if version != "" {
		kameletVersionSpec, ok := k.Spec.Versions[version]
		if !ok {
			return nil, fmt.Errorf("could not find version %s for Kamelet %s/%s", version, k.Namespace, k.Name)
		}
		clone.Spec.KameletSpecBase = kameletVersionSpec
	}
	// Remove any existing version
	clone.Spec.Versions = nil

	return clone, nil
}

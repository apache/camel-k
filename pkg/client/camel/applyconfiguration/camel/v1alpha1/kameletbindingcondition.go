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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1alpha1"
	camelv1 "github.com/apache/camel-k/v2/pkg/client/camel/applyconfiguration/camel/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KameletBindingConditionApplyConfiguration represents an declarative configuration of the KameletBindingCondition type for use
// with apply.
type KameletBindingConditionApplyConfiguration struct {
	Type               *v1alpha1.KameletBindingConditionType    `json:"type,omitempty"`
	Status             *v1.ConditionStatus                      `json:"status,omitempty"`
	LastUpdateTime     *metav1.Time                             `json:"lastUpdateTime,omitempty"`
	LastTransitionTime *metav1.Time                             `json:"lastTransitionTime,omitempty"`
	Reason             *string                                  `json:"reason,omitempty"`
	Message            *string                                  `json:"message,omitempty"`
	Pods               []camelv1.PodConditionApplyConfiguration `json:"pods,omitempty"`
}

// KameletBindingConditionApplyConfiguration constructs an declarative configuration of the KameletBindingCondition type for use with
// apply.
func KameletBindingCondition() *KameletBindingConditionApplyConfiguration {
	return &KameletBindingConditionApplyConfiguration{}
}

// WithType sets the Type field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Type field is set to the value of the last call.
func (b *KameletBindingConditionApplyConfiguration) WithType(value v1alpha1.KameletBindingConditionType) *KameletBindingConditionApplyConfiguration {
	b.Type = &value
	return b
}

// WithStatus sets the Status field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Status field is set to the value of the last call.
func (b *KameletBindingConditionApplyConfiguration) WithStatus(value v1.ConditionStatus) *KameletBindingConditionApplyConfiguration {
	b.Status = &value
	return b
}

// WithLastUpdateTime sets the LastUpdateTime field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the LastUpdateTime field is set to the value of the last call.
func (b *KameletBindingConditionApplyConfiguration) WithLastUpdateTime(value metav1.Time) *KameletBindingConditionApplyConfiguration {
	b.LastUpdateTime = &value
	return b
}

// WithLastTransitionTime sets the LastTransitionTime field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the LastTransitionTime field is set to the value of the last call.
func (b *KameletBindingConditionApplyConfiguration) WithLastTransitionTime(value metav1.Time) *KameletBindingConditionApplyConfiguration {
	b.LastTransitionTime = &value
	return b
}

// WithReason sets the Reason field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Reason field is set to the value of the last call.
func (b *KameletBindingConditionApplyConfiguration) WithReason(value string) *KameletBindingConditionApplyConfiguration {
	b.Reason = &value
	return b
}

// WithMessage sets the Message field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Message field is set to the value of the last call.
func (b *KameletBindingConditionApplyConfiguration) WithMessage(value string) *KameletBindingConditionApplyConfiguration {
	b.Message = &value
	return b
}

// WithPods adds the given value to the Pods field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Pods field.
func (b *KameletBindingConditionApplyConfiguration) WithPods(values ...*camelv1.PodConditionApplyConfiguration) *KameletBindingConditionApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithPods")
		}
		b.Pods = append(b.Pods, *values[i])
	}
	return b
}

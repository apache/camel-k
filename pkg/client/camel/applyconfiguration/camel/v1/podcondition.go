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

package v1

import (
	v1 "k8s.io/api/core/v1"
)

// PodConditionApplyConfiguration represents an declarative configuration of the PodCondition type for use
// with apply.
type PodConditionApplyConfiguration struct {
	Name      *string                                 `json:"name,omitempty"`
	Condition *v1.PodCondition                        `json:"condition,omitempty"`
	Health    []HealthCheckResponseApplyConfiguration `json:"health,omitempty"`
}

// PodConditionApplyConfiguration constructs an declarative configuration of the PodCondition type for use with
// apply.
func PodCondition() *PodConditionApplyConfiguration {
	return &PodConditionApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *PodConditionApplyConfiguration) WithName(value string) *PodConditionApplyConfiguration {
	b.Name = &value
	return b
}

// WithCondition sets the Condition field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Condition field is set to the value of the last call.
func (b *PodConditionApplyConfiguration) WithCondition(value v1.PodCondition) *PodConditionApplyConfiguration {
	b.Condition = &value
	return b
}

// WithHealth adds the given value to the Health field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Health field.
func (b *PodConditionApplyConfiguration) WithHealth(values ...*HealthCheckResponseApplyConfiguration) *PodConditionApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithHealth")
		}
		b.Health = append(b.Health, *values[i])
	}
	return b
}

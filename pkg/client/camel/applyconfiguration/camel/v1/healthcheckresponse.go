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
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// HealthCheckResponseApplyConfiguration represents an declarative configuration of the HealthCheckResponse type for use
// with apply.
type HealthCheckResponseApplyConfiguration struct {
	Name   *string               `json:"name,omitempty"`
	Status *v1.HealthCheckStatus `json:"status,omitempty"`
	Data   *v1.RawMessage        `json:"data,omitempty"`
}

// HealthCheckResponseApplyConfiguration constructs an declarative configuration of the HealthCheckResponse type for use with
// apply.
func HealthCheckResponse() *HealthCheckResponseApplyConfiguration {
	return &HealthCheckResponseApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *HealthCheckResponseApplyConfiguration) WithName(value string) *HealthCheckResponseApplyConfiguration {
	b.Name = &value
	return b
}

// WithStatus sets the Status field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Status field is set to the value of the last call.
func (b *HealthCheckResponseApplyConfiguration) WithStatus(value v1.HealthCheckStatus) *HealthCheckResponseApplyConfiguration {
	b.Status = &value
	return b
}

// WithData sets the Data field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Data field is set to the value of the last call.
func (b *HealthCheckResponseApplyConfiguration) WithData(value v1.RawMessage) *HealthCheckResponseApplyConfiguration {
	b.Data = &value
	return b
}

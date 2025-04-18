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
	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

// IntegrationProfileStatusApplyConfiguration represents a declarative configuration of the IntegrationProfileStatus type for use
// with apply.
type IntegrationProfileStatusApplyConfiguration struct {
	IntegrationProfileSpecApplyConfiguration `json:",inline"`
	ObservedGeneration                       *int64                                          `json:"observedGeneration,omitempty"`
	Phase                                    *camelv1.IntegrationProfilePhase                `json:"phase,omitempty"`
	Conditions                               []IntegrationProfileConditionApplyConfiguration `json:"conditions,omitempty"`
}

// IntegrationProfileStatusApplyConfiguration constructs a declarative configuration of the IntegrationProfileStatus type for use with
// apply.
func IntegrationProfileStatus() *IntegrationProfileStatusApplyConfiguration {
	return &IntegrationProfileStatusApplyConfiguration{}
}

// WithBuild sets the Build field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Build field is set to the value of the last call.
func (b *IntegrationProfileStatusApplyConfiguration) WithBuild(value *IntegrationProfileBuildSpecApplyConfiguration) *IntegrationProfileStatusApplyConfiguration {
	b.IntegrationProfileSpecApplyConfiguration.Build = value
	return b
}

// WithTraits sets the Traits field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Traits field is set to the value of the last call.
func (b *IntegrationProfileStatusApplyConfiguration) WithTraits(value *TraitsApplyConfiguration) *IntegrationProfileStatusApplyConfiguration {
	b.IntegrationProfileSpecApplyConfiguration.Traits = value
	return b
}

// WithKamelet sets the Kamelet field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Kamelet field is set to the value of the last call.
func (b *IntegrationProfileStatusApplyConfiguration) WithKamelet(value *IntegrationProfileKameletSpecApplyConfiguration) *IntegrationProfileStatusApplyConfiguration {
	b.IntegrationProfileSpecApplyConfiguration.Kamelet = value
	return b
}

// WithObservedGeneration sets the ObservedGeneration field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ObservedGeneration field is set to the value of the last call.
func (b *IntegrationProfileStatusApplyConfiguration) WithObservedGeneration(value int64) *IntegrationProfileStatusApplyConfiguration {
	b.ObservedGeneration = &value
	return b
}

// WithPhase sets the Phase field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Phase field is set to the value of the last call.
func (b *IntegrationProfileStatusApplyConfiguration) WithPhase(value camelv1.IntegrationProfilePhase) *IntegrationProfileStatusApplyConfiguration {
	b.Phase = &value
	return b
}

// WithConditions adds the given value to the Conditions field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Conditions field.
func (b *IntegrationProfileStatusApplyConfiguration) WithConditions(values ...*IntegrationProfileConditionApplyConfiguration) *IntegrationProfileStatusApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithConditions")
		}
		b.Conditions = append(b.Conditions, *values[i])
	}
	return b
}

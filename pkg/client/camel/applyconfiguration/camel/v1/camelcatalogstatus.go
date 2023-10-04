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
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

// CamelCatalogStatusApplyConfiguration represents an declarative configuration of the CamelCatalogStatus type for use
// with apply.
type CamelCatalogStatusApplyConfiguration struct {
	ObservedGeneration *int64                                    `json:"observedGeneration,omitempty"`
	Phase              *v1.CamelCatalogPhase                     `json:"phase,omitempty"`
	Conditions         []CamelCatalogConditionApplyConfiguration `json:"conditions,omitempty"`
	Image              *string                                   `json:"image,omitempty"`
}

// CamelCatalogStatusApplyConfiguration constructs an declarative configuration of the CamelCatalogStatus type for use with
// apply.
func CamelCatalogStatus() *CamelCatalogStatusApplyConfiguration {
	return &CamelCatalogStatusApplyConfiguration{}
}

// WithObservedGeneration sets the ObservedGeneration field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ObservedGeneration field is set to the value of the last call.
func (b *CamelCatalogStatusApplyConfiguration) WithObservedGeneration(value int64) *CamelCatalogStatusApplyConfiguration {
	b.ObservedGeneration = &value
	return b
}

// WithPhase sets the Phase field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Phase field is set to the value of the last call.
func (b *CamelCatalogStatusApplyConfiguration) WithPhase(value v1.CamelCatalogPhase) *CamelCatalogStatusApplyConfiguration {
	b.Phase = &value
	return b
}

// WithConditions adds the given value to the Conditions field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Conditions field.
func (b *CamelCatalogStatusApplyConfiguration) WithConditions(values ...*CamelCatalogConditionApplyConfiguration) *CamelCatalogStatusApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithConditions")
		}
		b.Conditions = append(b.Conditions, *values[i])
	}
	return b
}

// WithImage sets the Image field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Image field is set to the value of the last call.
func (b *CamelCatalogStatusApplyConfiguration) WithImage(value string) *CamelCatalogStatusApplyConfiguration {
	b.Image = &value
	return b
}
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
	corev1 "k8s.io/api/core/v1"
)

// EndpointApplyConfiguration represents a declarative configuration of the Endpoint type for use
// with apply.
type EndpointApplyConfiguration struct {
	Ref        *corev1.ObjectReference                                  `json:"ref,omitempty"`
	URI        *string                                                  `json:"uri,omitempty"`
	Properties *EndpointPropertiesApplyConfiguration                    `json:"properties,omitempty"`
	DataTypes  map[camelv1.TypeSlot]DataTypeReferenceApplyConfiguration `json:"dataTypes,omitempty"`
}

// EndpointApplyConfiguration constructs a declarative configuration of the Endpoint type for use with
// apply.
func Endpoint() *EndpointApplyConfiguration {
	return &EndpointApplyConfiguration{}
}

// WithRef sets the Ref field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Ref field is set to the value of the last call.
func (b *EndpointApplyConfiguration) WithRef(value corev1.ObjectReference) *EndpointApplyConfiguration {
	b.Ref = &value
	return b
}

// WithURI sets the URI field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the URI field is set to the value of the last call.
func (b *EndpointApplyConfiguration) WithURI(value string) *EndpointApplyConfiguration {
	b.URI = &value
	return b
}

// WithProperties sets the Properties field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Properties field is set to the value of the last call.
func (b *EndpointApplyConfiguration) WithProperties(value *EndpointPropertiesApplyConfiguration) *EndpointApplyConfiguration {
	b.Properties = value
	return b
}

// WithDataTypes puts the entries into the DataTypes field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the DataTypes field,
// overwriting an existing map entries in DataTypes field with the same key.
func (b *EndpointApplyConfiguration) WithDataTypes(entries map[camelv1.TypeSlot]DataTypeReferenceApplyConfiguration) *EndpointApplyConfiguration {
	if b.DataTypes == nil && len(entries) > 0 {
		b.DataTypes = make(map[camelv1.TypeSlot]DataTypeReferenceApplyConfiguration, len(entries))
	}
	for k, v := range entries {
		b.DataTypes[k] = v
	}
	return b
}

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

// EndpointSubsetApplyConfiguration represents an declarative configuration of the EndpointSubset type for use
// with apply.
type EndpointSubsetApplyConfiguration struct {
	Addresses         []EndpointAddressApplyConfiguration `json:"addresses,omitempty"`
	NotReadyAddresses []EndpointAddressApplyConfiguration `json:"notReadyAddresses,omitempty"`
	Ports             []EndpointPortApplyConfiguration    `json:"ports,omitempty"`
}

// EndpointSubsetApplyConfiguration constructs an declarative configuration of the EndpointSubset type for use with
// apply.
func EndpointSubset() *EndpointSubsetApplyConfiguration {
	return &EndpointSubsetApplyConfiguration{}
}

// WithAddresses adds the given value to the Addresses field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Addresses field.
func (b *EndpointSubsetApplyConfiguration) WithAddresses(values ...*EndpointAddressApplyConfiguration) *EndpointSubsetApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithAddresses")
		}
		b.Addresses = append(b.Addresses, *values[i])
	}
	return b
}

// WithNotReadyAddresses adds the given value to the NotReadyAddresses field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the NotReadyAddresses field.
func (b *EndpointSubsetApplyConfiguration) WithNotReadyAddresses(values ...*EndpointAddressApplyConfiguration) *EndpointSubsetApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithNotReadyAddresses")
		}
		b.NotReadyAddresses = append(b.NotReadyAddresses, *values[i])
	}
	return b
}

// WithPorts adds the given value to the Ports field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Ports field.
func (b *EndpointSubsetApplyConfiguration) WithPorts(values ...*EndpointPortApplyConfiguration) *EndpointSubsetApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithPorts")
		}
		b.Ports = append(b.Ports, *values[i])
	}
	return b
}

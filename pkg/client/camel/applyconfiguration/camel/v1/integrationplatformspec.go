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

// IntegrationPlatformSpecApplyConfiguration represents a declarative configuration of the IntegrationPlatformSpec type for use
// with apply.
type IntegrationPlatformSpecApplyConfiguration struct {
	Cluster       *camelv1.IntegrationPlatformCluster               `json:"cluster,omitempty"`
	Profile       *camelv1.TraitProfile                             `json:"profile,omitempty"`
	Build         *IntegrationPlatformBuildSpecApplyConfiguration   `json:"build,omitempty"`
	Traits        *TraitsApplyConfiguration                         `json:"traits,omitempty"`
	Configuration []ConfigurationSpecApplyConfiguration             `json:"configuration,omitempty"`
	Kamelet       *IntegrationPlatformKameletSpecApplyConfiguration `json:"kamelet,omitempty"`
}

// IntegrationPlatformSpecApplyConfiguration constructs a declarative configuration of the IntegrationPlatformSpec type for use with
// apply.
func IntegrationPlatformSpec() *IntegrationPlatformSpecApplyConfiguration {
	return &IntegrationPlatformSpecApplyConfiguration{}
}

// WithCluster sets the Cluster field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Cluster field is set to the value of the last call.
func (b *IntegrationPlatformSpecApplyConfiguration) WithCluster(value camelv1.IntegrationPlatformCluster) *IntegrationPlatformSpecApplyConfiguration {
	b.Cluster = &value
	return b
}

// WithProfile sets the Profile field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Profile field is set to the value of the last call.
func (b *IntegrationPlatformSpecApplyConfiguration) WithProfile(value camelv1.TraitProfile) *IntegrationPlatformSpecApplyConfiguration {
	b.Profile = &value
	return b
}

// WithBuild sets the Build field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Build field is set to the value of the last call.
func (b *IntegrationPlatformSpecApplyConfiguration) WithBuild(value *IntegrationPlatformBuildSpecApplyConfiguration) *IntegrationPlatformSpecApplyConfiguration {
	b.Build = value
	return b
}

// WithTraits sets the Traits field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Traits field is set to the value of the last call.
func (b *IntegrationPlatformSpecApplyConfiguration) WithTraits(value *TraitsApplyConfiguration) *IntegrationPlatformSpecApplyConfiguration {
	b.Traits = value
	return b
}

// WithConfiguration adds the given value to the Configuration field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Configuration field.
func (b *IntegrationPlatformSpecApplyConfiguration) WithConfiguration(values ...*ConfigurationSpecApplyConfiguration) *IntegrationPlatformSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithConfiguration")
		}
		b.Configuration = append(b.Configuration, *values[i])
	}
	return b
}

// WithKamelet sets the Kamelet field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Kamelet field is set to the value of the last call.
func (b *IntegrationPlatformSpecApplyConfiguration) WithKamelet(value *IntegrationPlatformKameletSpecApplyConfiguration) *IntegrationPlatformSpecApplyConfiguration {
	b.Kamelet = value
	return b
}

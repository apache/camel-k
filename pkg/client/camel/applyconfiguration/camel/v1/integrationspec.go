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

// IntegrationSpecApplyConfiguration represents an declarative configuration of the IntegrationSpec type for use
// with apply.
type IntegrationSpecApplyConfiguration struct {
	Replicas       *int32                                `json:"replicas,omitempty"`
	Sources        []SourceSpecApplyConfiguration        `json:"sources,omitempty"`
	Flows          []FlowApplyConfiguration              `json:"flows,omitempty"`
	IntegrationKit *corev1.ObjectReference               `json:"integrationKit,omitempty"`
	Dependencies   []string                              `json:"dependencies,omitempty"`
	Profile        *camelv1.TraitProfile                 `json:"profile,omitempty"`
	Traits         *TraitsApplyConfiguration             `json:"traits,omitempty"`
	PodTemplate    *PodSpecTemplateApplyConfiguration    `json:"template,omitempty"`
	Configuration  []ConfigurationSpecApplyConfiguration `json:"configuration,omitempty"`
}

// IntegrationSpecApplyConfiguration constructs an declarative configuration of the IntegrationSpec type for use with
// apply.
func IntegrationSpec() *IntegrationSpecApplyConfiguration {
	return &IntegrationSpecApplyConfiguration{}
}

// WithReplicas sets the Replicas field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Replicas field is set to the value of the last call.
func (b *IntegrationSpecApplyConfiguration) WithReplicas(value int32) *IntegrationSpecApplyConfiguration {
	b.Replicas = &value
	return b
}

// WithSources adds the given value to the Sources field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Sources field.
func (b *IntegrationSpecApplyConfiguration) WithSources(values ...*SourceSpecApplyConfiguration) *IntegrationSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithSources")
		}
		b.Sources = append(b.Sources, *values[i])
	}
	return b
}

// WithFlows adds the given value to the Flows field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Flows field.
func (b *IntegrationSpecApplyConfiguration) WithFlows(values ...*FlowApplyConfiguration) *IntegrationSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithFlows")
		}
		b.Flows = append(b.Flows, *values[i])
	}
	return b
}

// WithIntegrationKit sets the IntegrationKit field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the IntegrationKit field is set to the value of the last call.
func (b *IntegrationSpecApplyConfiguration) WithIntegrationKit(value corev1.ObjectReference) *IntegrationSpecApplyConfiguration {
	b.IntegrationKit = &value
	return b
}

// WithDependencies adds the given value to the Dependencies field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Dependencies field.
func (b *IntegrationSpecApplyConfiguration) WithDependencies(values ...string) *IntegrationSpecApplyConfiguration {
	for i := range values {
		b.Dependencies = append(b.Dependencies, values[i])
	}
	return b
}

// WithProfile sets the Profile field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Profile field is set to the value of the last call.
func (b *IntegrationSpecApplyConfiguration) WithProfile(value camelv1.TraitProfile) *IntegrationSpecApplyConfiguration {
	b.Profile = &value
	return b
}

// WithTraits sets the Traits field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Traits field is set to the value of the last call.
func (b *IntegrationSpecApplyConfiguration) WithTraits(value *TraitsApplyConfiguration) *IntegrationSpecApplyConfiguration {
	b.Traits = value
	return b
}

// WithPodTemplate sets the PodTemplate field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PodTemplate field is set to the value of the last call.
func (b *IntegrationSpecApplyConfiguration) WithPodTemplate(value *PodSpecTemplateApplyConfiguration) *IntegrationSpecApplyConfiguration {
	b.PodTemplate = value
	return b
}

// WithConfiguration adds the given value to the Configuration field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Configuration field.
func (b *IntegrationSpecApplyConfiguration) WithConfiguration(values ...*ConfigurationSpecApplyConfiguration) *IntegrationSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithConfiguration")
		}
		b.Configuration = append(b.Configuration, *values[i])
	}
	return b
}

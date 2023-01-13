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
	v1 "github.com/apache/camel-k/pkg/client/camel/applyconfiguration/core/v1"
	corev1 "k8s.io/api/core/v1"
)

// PodSpecApplyConfiguration represents an declarative configuration of the PodSpec type for use
// with apply.
type PodSpecApplyConfiguration struct {
	Volumes                       []v1.VolumeApplyConfiguration                   `json:"volumes,omitempty"`
	InitContainers                []v1.ContainerApplyConfiguration                `json:"initContainers,omitempty"`
	Containers                    []v1.ContainerApplyConfiguration                `json:"containers,omitempty"`
	EphemeralContainers           []v1.EphemeralContainerApplyConfiguration       `json:"ephemeralContainers,omitempty"`
	RestartPolicy                 *corev1.RestartPolicy                           `json:"restartPolicy,omitempty"`
	TerminationGracePeriodSeconds *int64                                          `json:"terminationGracePeriodSeconds,omitempty"`
	ActiveDeadlineSeconds         *int64                                          `json:"activeDeadlineSeconds,omitempty"`
	DNSPolicy                     *corev1.DNSPolicy                               `json:"dnsPolicy,omitempty"`
	NodeSelector                  map[string]string                               `json:"nodeSelector,omitempty"`
	TopologySpreadConstraints     []v1.TopologySpreadConstraintApplyConfiguration `json:"topologySpreadConstraints,omitempty"`
	SecurityContext               *v1.PodSecurityContextApplyConfiguration        `json:"securityContext,omitempty"`
}

// PodSpecApplyConfiguration constructs an declarative configuration of the PodSpec type for use with
// apply.
func PodSpec() *PodSpecApplyConfiguration {
	return &PodSpecApplyConfiguration{}
}

// WithVolumes adds the given value to the Volumes field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Volumes field.
func (b *PodSpecApplyConfiguration) WithVolumes(values ...*v1.VolumeApplyConfiguration) *PodSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithVolumes")
		}
		b.Volumes = append(b.Volumes, *values[i])
	}
	return b
}

// WithInitContainers adds the given value to the InitContainers field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the InitContainers field.
func (b *PodSpecApplyConfiguration) WithInitContainers(values ...*v1.ContainerApplyConfiguration) *PodSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithInitContainers")
		}
		b.InitContainers = append(b.InitContainers, *values[i])
	}
	return b
}

// WithContainers adds the given value to the Containers field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Containers field.
func (b *PodSpecApplyConfiguration) WithContainers(values ...*v1.ContainerApplyConfiguration) *PodSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithContainers")
		}
		b.Containers = append(b.Containers, *values[i])
	}
	return b
}

// WithEphemeralContainers adds the given value to the EphemeralContainers field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the EphemeralContainers field.
func (b *PodSpecApplyConfiguration) WithEphemeralContainers(values ...*v1.EphemeralContainerApplyConfiguration) *PodSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithEphemeralContainers")
		}
		b.EphemeralContainers = append(b.EphemeralContainers, *values[i])
	}
	return b
}

// WithRestartPolicy sets the RestartPolicy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RestartPolicy field is set to the value of the last call.
func (b *PodSpecApplyConfiguration) WithRestartPolicy(value corev1.RestartPolicy) *PodSpecApplyConfiguration {
	b.RestartPolicy = &value
	return b
}

// WithTerminationGracePeriodSeconds sets the TerminationGracePeriodSeconds field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TerminationGracePeriodSeconds field is set to the value of the last call.
func (b *PodSpecApplyConfiguration) WithTerminationGracePeriodSeconds(value int64) *PodSpecApplyConfiguration {
	b.TerminationGracePeriodSeconds = &value
	return b
}

// WithActiveDeadlineSeconds sets the ActiveDeadlineSeconds field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ActiveDeadlineSeconds field is set to the value of the last call.
func (b *PodSpecApplyConfiguration) WithActiveDeadlineSeconds(value int64) *PodSpecApplyConfiguration {
	b.ActiveDeadlineSeconds = &value
	return b
}

// WithDNSPolicy sets the DNSPolicy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DNSPolicy field is set to the value of the last call.
func (b *PodSpecApplyConfiguration) WithDNSPolicy(value corev1.DNSPolicy) *PodSpecApplyConfiguration {
	b.DNSPolicy = &value
	return b
}

// WithNodeSelector puts the entries into the NodeSelector field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the NodeSelector field,
// overwriting an existing map entries in NodeSelector field with the same key.
func (b *PodSpecApplyConfiguration) WithNodeSelector(entries map[string]string) *PodSpecApplyConfiguration {
	if b.NodeSelector == nil && len(entries) > 0 {
		b.NodeSelector = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		b.NodeSelector[k] = v
	}
	return b
}

// WithTopologySpreadConstraints adds the given value to the TopologySpreadConstraints field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the TopologySpreadConstraints field.
func (b *PodSpecApplyConfiguration) WithTopologySpreadConstraints(values ...*v1.TopologySpreadConstraintApplyConfiguration) *PodSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithTopologySpreadConstraints")
		}
		b.TopologySpreadConstraints = append(b.TopologySpreadConstraints, *values[i])
	}
	return b
}

// WithSecurityContext sets the SecurityContext field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SecurityContext field is set to the value of the last call.
func (b *PodSpecApplyConfiguration) WithSecurityContext(value *v1.PodSecurityContextApplyConfiguration) *PodSpecApplyConfiguration {
	b.SecurityContext = value
	return b
}

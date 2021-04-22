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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// KameletBindingSpec --
type KameletBindingSpec struct {
	// Integration is an optional integration used to specify custom parameters
	Integration *v1.IntegrationSpec `json:"integration,omitempty"`
	// Source is the starting point of the integration defined by this binding
	Source Endpoint `json:"source,omitempty"`
	// Sink is the destination of the integration defined by this binding
	Sink Endpoint `json:"sink,omitempty"`
	// ErrorHandler is an optional handler called upon an error occuring in the integration
	ErrorHandler ErrorHandler `json:"errorHandler,omitempty"`
	// Steps contains an optional list of intermediate steps that are executed between the Source and the Sink
	Steps []Endpoint `json:"steps,omitempty"`
}

// Endpoint represents a source/sink external entity
type Endpoint struct {
	// Ref can be used to declare a Kubernetes resource as source/sink endpoint
	Ref *corev1.ObjectReference `json:"ref,omitempty"`
	// URI can alternatively be used to specify the (Camel) endpoint explicitly
	URI *string `json:"uri,omitempty"`
	// Properties are a key value representation of endpoint properties
	Properties *EndpointProperties `json:"properties,omitempty"`
	// Types defines the schema of the data produced/consumed by the endpoint
	Types map[EventSlot]EventTypeSpec `json:"types,omitempty"`
}

type EndpointType string

const (
	EndpointTypeSource       EndpointType = "source"
	EndpointTypeAction       EndpointType = "action"
	EndpointTypeSink         EndpointType = "sink"
	EndpointTypeErrorHandler EndpointType = "errorHandler"
)

type ErrorHandlerProperties struct {
	v1.RawMessage `json:",inline"`
}

// EndpointProperties is a key/value struct represented as JSON raw to allow numeric/boolean values
type EndpointProperties struct {
	v1.RawMessage `json:",inline"`
}

// KameletBindingStatus --
type KameletBindingStatus struct {
	// Phase --
	Phase KameletBindingPhase `json:"phase,omitempty"`
	// Conditions --
	Conditions []KameletBindingCondition `json:"conditions,omitempty"`
}

// KameletBindingCondition describes the state of a resource at a certain point.
type KameletBindingCondition struct {
	// Type of kameletBinding condition.
	Type KameletBindingConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

type KameletBindingConditionType string

const (
	// KameletBindingConditionReady --
	KameletBindingConditionReady KameletBindingConditionType = "Ready"
)

type KameletBindingPhase string

const (
	// KameletKind --
	KameletBindingKind string = "KameletBinding"

	// KameletBindingPhaseNone --
	KameletBindingPhaseNone KameletBindingPhase = ""
	// KameletBindingPhaseCreating --
	KameletBindingPhaseCreating KameletBindingPhase = "Creating"
	// KameletBindingPhaseError --
	KameletBindingPhaseError KameletBindingPhase = "Error"
	// KameletBindingPhaseReady --
	KameletBindingPhaseReady KameletBindingPhase = "Ready"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=kameletbindings,scope=Namespaced,shortName=klb,categories=kamel;camel
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The Kamelet Binding phase"

// KameletBinding is the Schema for the kamelets binding API
type KameletBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KameletBindingSpec   `json:"spec,omitempty"`
	Status KameletBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KameletBindingList contains a list of KameletBinding
type KameletBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KameletBinding `json:"items"`
}

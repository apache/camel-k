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

	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

const (
	// AnnotationIcon label used by icons
	AnnotationIcon = "camel.apache.org/kamelet.icon"
	// KameletBundledLabel label used by bundling
	KameletBundledLabel = "camel.apache.org/kamelet.bundled"
	// KameletReadOnlyLabel label used to identify readonly Kamelets
	KameletReadOnlyLabel = "camel.apache.org/kamelet.readonly"
	// KameletTypeLabel label used to identify Kamelet type
	KameletTypeLabel = "camel.apache.org/kamelet.type"
	// KameletGroupLabel label used to group Kamelets
	KameletGroupLabel = "camel.apache.org/kamelet.group"

	// KameletTypeSink type Sink
	KameletTypeSink = "sink"
	// KameletTypeSource type Source
	KameletTypeSource = "source"
	// KameletTypeAction type Action
	KameletTypeAction = "action"
)

var (
	// reservedKameletNames used to mark reserved names
	reservedKameletNames = map[string]bool{"source": true, "sink": true}
	// KameletIDProperty used to identify
	KameletIDProperty = "id"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=kamelets,scope=Namespaced,shortName=kl,categories=kamel;camel
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The Kamelet phase"

// Kamelet is the Schema for the kamelets API
type Kamelet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// the desired specification
	Spec KameletSpec `json:"spec,omitempty"`
	// the actual status of the resource
	Status KameletStatus `json:"status,omitempty"`
}

// KameletSpec specifies the configuration required to execute a Kamelet
type KameletSpec struct {
	// defines the formal configuration of the Kamelet
	Definition *JSONSchemaProps `json:"definition,omitempty"`
	// sources in any Camel DSL supported
	Sources []camelv1.SourceSpec `json:"sources,omitempty"`
	// the main source in YAML DSL
	Template *Template `json:"template,omitempty"`
	// Deprecated: use Template instead
	// the main source in YAML DSL
	Flow *camelv1.Flow `json:"flow,omitempty"`
	// Deprecated: unused
	Authorization *AuthorizationSpec `json:"authorization,omitempty"`
	// data specification types for the events consumed/produced by the Kamelet
	Types map[EventSlot]EventTypeSpec `json:"types,omitempty"`
	// Camel dependencies needed by the Kamelet
	Dependencies []string `json:"dependencies,omitempty"`
}

// Template is an unstructured object representing a Kamelet template in YAML/JSON DSL
type Template struct {
	// an unstructured raw message
	RawMessage `json:",inline"`
}

// EventSlot represent a kind of data (ie, input, output, ...)
type EventSlot string

const (
	// EventSlotIn is used for the input events
	EventSlotIn EventSlot = "in"
	// EventSlotOut is used for the output events
	EventSlotOut EventSlot = "out"
	// EventSlotError is used for the error events
	EventSlotError EventSlot = "error"
)

// EventTypeSpec represents a specification for an event type
type EventTypeSpec struct {
	// media type as expected for HTTP media types (ie, application/json)
	MediaType string `json:"mediaType,omitempty"`
	// the expected schema for the event
	Schema *JSONSchemaProps `json:"schema,omitempty"`
}

// AuthorizationSpec is TODO (oauth information)
// Deprecated: unused
type AuthorizationSpec struct {
}

// KameletStatus defines the observed state of Kamelet
type KameletStatus struct {
	// Phase --
	Phase KameletPhase `json:"phase,omitempty"`
	// Conditions --
	Conditions []KameletCondition `json:"conditions,omitempty"`
	// Properties --
	Properties []KameletProperty `json:"properties,omitempty"`
}

// KameletProperty specify the behavior of a property in a Kamelet
type KameletProperty struct {
	// the name of the property
	Name string `json:"name,omitempty"`
	// the default value of the property (if any)
	Default string `json:"default,omitempty"`
}

// KameletCondition describes the state of a resource at a certain point.
type KameletCondition struct {
	// Type of kamelet condition.
	Type KameletConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human-readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// KameletConditionType --
type KameletConditionType string

const (
	// KameletConditionReady --
	KameletConditionReady KameletConditionType = "Ready"
)

const (
	// KameletConditionReasonInvalidName --
	KameletConditionReasonInvalidName string = "InvalidName"
	// KameletConditionReasonInvalidProperty --
	KameletConditionReasonInvalidProperty string = "InvalidProperty"
	// KameletConditionReasonInvalidTemplate --
	KameletConditionReasonInvalidTemplate string = "InvalidTemplate"
)

// KameletPhase --
type KameletPhase string

const (
	// KameletKind --
	KameletKind string = "Kamelet"

	// KameletPhaseNone --
	KameletPhaseNone KameletPhase = ""
	// KameletPhaseReady --
	KameletPhaseReady KameletPhase = "Ready"
	// KameletPhaseError --
	KameletPhaseError KameletPhase = "Error"
)

// +kubebuilder:object:root=true

// KameletList contains a list of Kamelet
type KameletList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kamelet `json:"items"`
}

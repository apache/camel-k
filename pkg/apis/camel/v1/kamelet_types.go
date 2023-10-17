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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// AnnotationIcon label used by icons.
	AnnotationIcon = "camel.apache.org/kamelet.icon"
	// KameletBundledLabel label used by bundling.
	KameletBundledLabel = "camel.apache.org/kamelet.bundled"
	// KameletReadOnlyLabel label used to identify readonly Kamelets.
	KameletReadOnlyLabel = "camel.apache.org/kamelet.readonly"
	// KameletTypeLabel label used to identify Kamelet type.
	KameletTypeLabel = "camel.apache.org/kamelet.type"
	// KameletGroupLabel label used to group Kamelets.
	KameletGroupLabel = "camel.apache.org/kamelet.group"

	// KameletTypeSink type Sink.
	KameletTypeSink = "sink"
	// KameletTypeSource type Source.
	KameletTypeSource = "source"
	// KameletTypeAction type Action.
	KameletTypeAction = "action"
)

var (
	// reservedKameletNames used to mark reserved names.
	reservedKameletNames = map[string]bool{"source": true, "sink": true}
	// KameletIDProperty used to identify.
	KameletIDProperty = "id"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:path=kamelets,scope=Namespaced,shortName=kl,categories=kamel;camel
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The Kamelet phase"

// Kamelet is the Schema for the kamelets API.
type Kamelet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// the desired specification.
	Spec KameletSpec `json:"spec,omitempty"`
	// the actual status of the resource.
	Status KameletStatus `json:"status,omitempty"`
}

// KameletSpec specifies the configuration required to execute a Kamelet.
type KameletSpec struct {
	// defines the formal configuration of the Kamelet.
	Definition *JSONSchemaProps `json:"definition,omitempty"`
	// sources in any Camel DSL supported.
	Sources []SourceSpec `json:"sources,omitempty"`
	// the main source in YAML DSL.
	Template *Template `json:"template,omitempty"`
	// data specification types for the events consumed/produced by the Kamelet
	// Deprecated: In favor of using DataTypes.
	Types map[TypeSlot]EventTypeSpec `json:"types,omitempty"`
	// data specification types for the events consumed/produced by the Kamelet.
	DataTypes map[TypeSlot]DataTypesSpec `json:"dataTypes,omitempty"`
	// Camel dependencies needed by the Kamelet.
	Dependencies []string `json:"dependencies,omitempty"`
}

// Template is an unstructured object representing a Kamelet template in YAML/JSON DSL.
type Template struct {
	// an unstructured raw message.
	RawMessage `json:",inline"`
}

// TypeSlot represent a kind of data (ie, input, output, ...).
type TypeSlot string

const (
	// TypeSlotIn is used for the input events.
	TypeSlotIn TypeSlot = "in"
	// TypeSlotOut is used for the output events.
	TypeSlotOut TypeSlot = "out"
	// TypeSlotError is used for the error events.
	TypeSlotError TypeSlot = "error"
)

// EventTypeSpec represents a specification for an event type
// Deprecated: In favor of using DataTypeSpec.
type EventTypeSpec struct {
	// media type as expected for HTTP media types (ie, application/json).
	MediaType string `json:"mediaType,omitempty"`
	// the expected schema for the event.
	Schema *JSONSchemaProps `json:"schema,omitempty"`
}

// DataTypesSpec represents the specification for a set of data types.
type DataTypesSpec struct {
	// the default data type for this Kamelet.
	Default string `json:"default,omitempty"`
	// one to many data type specifications.
	Types map[string]DataTypeSpec `json:"types,omitempty"`
	// one to many header specifications.
	Headers map[string]HeaderSpec `json:"headers,omitempty"`
}

// DataTypeSpec represents the specification for a data type.
type DataTypeSpec struct {
	// the data type component scheme.
	Scheme string `json:"scheme,omitempty"`
	// the data type format name.
	Format string `json:"format,omitempty"`
	// optional description.
	Description string `json:"description,omitempty"`
	// media type as expected for HTTP media types (ie, application/json).
	MediaType string `json:"mediaType,omitempty"`
	// the list of Camel or Maven dependencies required by the data type.
	Dependencies []string `json:"dependencies,omitempty"`
	// one to many header specifications.
	Headers map[string]HeaderSpec `json:"headers,omitempty"`
	// the expected schema for the data type.
	Schema *JSONSchemaProps `json:"schema,omitempty"`
}

// DataTypeReference references to the specification of a data type by its scheme and format name.
type DataTypeReference struct {
	// the data type component scheme.
	Scheme string `json:"scheme,omitempty"`
	// the data type format name.
	Format string `json:"format,omitempty"`
}

// HeaderSpec represents the specification for a header used in the Kamelet.
type HeaderSpec struct {
	Type        string `json:"type,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Default     string `json:"default,omitempty"`
}

// KameletStatus defines the observed state of Kamelet.
type KameletStatus struct {
	// ObservedGeneration is the most recent generation observed for this Kamelet.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Phase --
	Phase KameletPhase `json:"phase,omitempty"`
	// Conditions --
	Conditions []KameletCondition `json:"conditions,omitempty"`
	// Properties --
	Properties []KameletProperty `json:"properties,omitempty"`
}

// KameletProperty specify the behavior of a property in a Kamelet.
type KameletProperty struct {
	// the name of the property.
	Name string `json:"name,omitempty"`
	// the default value of the property (if any).
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

// KameletConditionType --.
type KameletConditionType string

const (
	// KameletConditionReady --.
	KameletConditionReady KameletConditionType = "Ready"
)

const (
	// KameletConditionReasonInvalidName --.
	KameletConditionReasonInvalidName string = "InvalidName"
	// KameletConditionReasonInvalidProperty --.
	KameletConditionReasonInvalidProperty string = "InvalidProperty"
	// KameletConditionReasonInvalidTemplate --.
	KameletConditionReasonInvalidTemplate string = "InvalidTemplate"
)

// KameletPhase --.
type KameletPhase string

const (
	// KameletKind --.
	KameletKind string = "Kamelet"

	// KameletPhaseNone --.
	KameletPhaseNone KameletPhase = ""
	// KameletPhaseReady --.
	KameletPhaseReady KameletPhase = "Ready"
	// KameletPhaseError --.
	KameletPhaseError KameletPhase = "Error"
)

// +kubebuilder:object:root=true

// KameletList contains a list of Kamelet.
type KameletList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kamelet `json:"items"`
}

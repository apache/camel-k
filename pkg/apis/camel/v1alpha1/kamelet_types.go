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
	AnnotationIcon       = "camel.apache.org/kamelet.icon"
	KameletBundledLabel  = "camel.apache.org/kamelet.bundled"
	KameletReadOnlyLabel = "camel.apache.org/kamelet.readonly"
	KameletTypeLabel     = "camel.apache.org/kamelet.type"
	KameletGroupLabel    = "camel.apache.org/kamelet.group"

	KameletTypeSink   = "sink"
	KameletTypeSource = "source"
	KameletTypeAction = "action"
)

var (
	reservedKameletNames = map[string]bool{"source": true, "sink": true}
	KameletIDProperty    = "id"
)

// KameletSpec defines the desired state of Kamelet
type KameletSpec struct {
	Definition *JSONSchemaProps     `json:"definition,omitempty"`
	Sources    []camelv1.SourceSpec `json:"sources,omitempty"`
	Template   *camelv1.Template    `json:"template,omitempty"`
	// Deprecated: use template
	Flow          *camelv1.Flow               `json:"flow,omitempty"`
	Authorization *AuthorizationSpec          `json:"authorization,omitempty"`
	Types         map[EventSlot]EventTypeSpec `json:"types,omitempty"`
	Dependencies  []string                    `json:"dependencies,omitempty"`
}

type EventTypeSpec struct {
	MediaType string           `json:"mediaType,omitempty"`
	Schema    *JSONSchemaProps `json:"schema,omitempty"`
}

type EventSlot string

const (
	EventSlotIn    EventSlot = "in"
	EventSlotOut   EventSlot = "out"
	EventSlotError EventSlot = "error"
)

// AuthorizationSpec is TODO (oauth information)
type AuthorizationSpec struct {
}

// KameletStatus defines the observed state of Kamelet
type KameletStatus struct {
	Phase      KameletPhase       `json:"phase,omitempty"`
	Conditions []KameletCondition `json:"conditions,omitempty"`
	Properties []KameletProperty  `json:"properties,omitempty"`
}

type KameletProperty struct {
	Name    string `json:"name,omitempty"`
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
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

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

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=kamelets,scope=Namespaced,shortName=kl,categories=kamel;camel
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The Kamelet phase"

// Kamelet is the Schema for the kamelets API
type Kamelet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KameletSpec   `json:"spec,omitempty"`
	Status KameletStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KameletList contains a list of Kamelet
type KameletList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kamelet `json:"items"`
}

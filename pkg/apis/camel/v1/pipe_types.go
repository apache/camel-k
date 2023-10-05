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

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=pipes,scope=Namespaced,shortName=pp,categories=kamel;camel
// +kubebuilder:subresource:status
// +genclient:method=GetScale,verb=get,subresource=scale,result=k8s.io/api/autoscaling/v1.Scale
// +genclient:method=UpdateScale,verb=update,subresource=scale,input=k8s.io/api/autoscaling/v1.Scale,result=k8s.io/api/autoscaling/v1.Scale
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The Pipe phase"
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.status.replicas`,description="The number of pods"

// Pipe is the Schema for the Pipe API.
type Pipe struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// the specification of a Pipe
	Spec PipeSpec `json:"spec,omitempty"`
	// the status of a Pipe
	Status PipeStatus `json:"status,omitempty"`
}

// PipeSpec defines the binding between a source and a sink. It can include custom parameters and additional intermediate steps and error handling.
type PipeSpec struct {
	// Integration is an optional integration used to specify custom parameters
	// Deprecated don't use this. Use trait annotations if you need to change any cluster configuration.
	Integration *IntegrationSpec `json:"integration,omitempty"`
	// Source is the starting point of the integration defined by this Pipe
	Source Endpoint `json:"source,omitempty"`
	// Sink is the destination of the integration defined by this Pipe
	Sink Endpoint `json:"sink,omitempty"`
	// ErrorHandler is an optional handler called upon an error occurring in the integration
	ErrorHandler *ErrorHandlerSpec `json:"errorHandler,omitempty"`
	// Steps contains an optional list of intermediate steps that are executed between the Source and the Sink
	Steps []Endpoint `json:"steps,omitempty"`
	// Replicas is the number of desired replicas for the Pipe
	Replicas *int32 `json:"replicas,omitempty"`
	// Custom SA to use for the Pipe
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// Endpoint represents a source/sink external entity (could be any Kubernetes resource or Camel URI).
type Endpoint struct {
	// Ref can be used to declare a Kubernetes resource as source/sink endpoint
	Ref *corev1.ObjectReference `json:"ref,omitempty"`
	// URI can be used to specify the (Camel) endpoint explicitly
	URI *string `json:"uri,omitempty"`
	// Properties are a key value representation of endpoint properties
	Properties *EndpointProperties `json:"properties,omitempty"`
	// DataTypes defines the data type of the data produced/consumed by the endpoint and references a given data type specification.
	DataTypes map[TypeSlot]DataTypeReference `json:"dataTypes,omitempty"`
}

// EndpointType represents the type (ie, source or sink).
type EndpointType string

const (
	// EndpointTypeSource source endpoint.
	EndpointTypeSource EndpointType = "source"
	// EndpointTypeAction action endpoint.
	EndpointTypeAction EndpointType = "action"
	// EndpointTypeSink sink endpoint.
	EndpointTypeSink EndpointType = "sink"
	// EndpointTypeErrorHandler error handler endpoint.
	EndpointTypeErrorHandler EndpointType = "errorHandler"
)

// EndpointProperties is a key/value struct represented as JSON raw to allow numeric/boolean values.
type EndpointProperties struct {
	RawMessage `json:",inline"`
}

// PipeStatus specify the status of a Pipe.
type PipeStatus struct {
	// ObservedGeneration is the most recent generation observed for this Pipe.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Phase --
	Phase PipePhase `json:"phase,omitempty"`
	// Conditions --
	Conditions []PipeCondition `json:"conditions,omitempty"`
	// Replicas is the number of actual replicas of the pipe
	Replicas *int32 `json:"replicas,omitempty"`
	// Selector allows to identify pods belonging to the pipe
	Selector string `json:"selector,omitempty"`
}

// PipeCondition describes the state of a resource at a certain point.
type PipeCondition struct {
	// Type of pipe condition.
	Type PipeConditionType `json:"type"`
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
	// Pods collect health and conditions information from the owned PODs
	Pods []PodCondition `json:"pods,omitempty"`
}

// PipeConditionType --.
type PipeConditionType string

const (
	// PipeConditionReady --.
	PipeConditionReady PipeConditionType = "Ready"
	// PipeIntegrationConditionError is used to report the error on the generated Integration.
	PipeIntegrationConditionError PipeConditionType = "IntegrationError"
	// PipeIntegrationDeprecationNotice is used to report the usage of a deprecated resource.
	PipeIntegrationDeprecationNotice PipeConditionType = "DeprecationNotice"
)

// PipePhase --.
type PipePhase string

const (
	// PipeKind --.
	PipeKind string = "Pipe"

	// PipePhaseNone --.
	PipePhaseNone PipePhase = ""
	// PipePhaseCreating --.
	PipePhaseCreating PipePhase = "Creating"
	// PipePhaseError --.
	PipePhaseError PipePhase = "Error"
	// PipePhaseReady --.
	PipePhaseReady PipePhase = "Ready"
)

// +kubebuilder:object:root=true

// PipeList contains a list of Pipe.
type PipeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipe `json:"items"`
}

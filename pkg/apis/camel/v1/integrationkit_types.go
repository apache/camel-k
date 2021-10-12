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

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "make generate-deepcopy" to regenerate code after modifying this file

// IntegrationKitSpec defines the desired state of IntegrationKit
type IntegrationKitSpec struct {
	Image         string               `json:"image,omitempty"`
	Dependencies  []string             `json:"dependencies,omitempty"`
	Profile       TraitProfile         `json:"profile,omitempty"`
	Traits        map[string]TraitSpec `json:"traits,omitempty"`
	Configuration []ConfigurationSpec  `json:"configuration,omitempty"`
	Repositories  []string             `json:"repositories,omitempty"`
}

// IntegrationKitStatus defines the observed state of IntegrationKit
type IntegrationKitStatus struct {
	Phase           IntegrationKitPhase       `json:"phase,omitempty"`
	BaseImage       string                    `json:"baseImage,omitempty"`
	Image           string                    `json:"image,omitempty"`
	Digest          string                    `json:"digest,omitempty"`
	Artifacts       []Artifact                `json:"artifacts,omitempty"`
	Failure         *Failure                  `json:"failure,omitempty"`
	RuntimeVersion  string                    `json:"runtimeVersion,omitempty"`
	RuntimeProvider RuntimeProvider           `json:"runtimeProvider,omitempty"`
	Platform        string                    `json:"platform,omitempty"`
	Conditions      []IntegrationKitCondition `json:"conditions,omitempty"`
	Version         string                    `json:"version,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=integrationkits,scope=Namespaced,shortName=ik,categories=kamel;camel
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The integration kit phase"
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.metadata.labels.camel\.apache\.org\/kit\.type`,description="The integration kit type"
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.status.image`,description="The integration kit image"

// IntegrationKit is the Schema for the integrationkits API
type IntegrationKit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntegrationKitSpec   `json:"spec,omitempty"`
	Status IntegrationKitStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IntegrationKitList contains a list of IntegrationKit
type IntegrationKitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IntegrationKit `json:"items"`
}

// IntegrationKitPhase --
type IntegrationKitPhase string

// IntegrationKitConditionType --
type IntegrationKitConditionType string

const (
	// IntegrationKitKind --
	IntegrationKitKind string = "IntegrationKit"

	// IntegrationKitTypeLabel labels the kit type
	IntegrationKitTypeLabel = "camel.apache.org/kit.type"

	// IntegrationKitTypePlatform --
	IntegrationKitTypePlatform = "platform"
	// IntegrationKitTypeUser --
	IntegrationKitTypeUser = "user"
	// IntegrationKitTypeExternal --
	IntegrationKitTypeExternal = "external"

	// IntegrationKitLayoutLabel labels the kit layout
	IntegrationKitLayoutLabel = "camel.apache.org/kit.layout"

	// IntegrationKitLayoutFastJar labels a kit using the Quarkus fast-jar packaging
	IntegrationKitLayoutFastJar = "fast-jar"
	// IntegrationKitLayoutNative labels a kit using the Quarkus native packaging
	IntegrationKitLayoutNative = "native"

	// IntegrationKitPriorityLabel labels the kit priority
	IntegrationKitPriorityLabel = "camel.apache.org/kit.priority"

	// IntegrationKitPhaseNone --
	IntegrationKitPhaseNone IntegrationKitPhase = ""
	// IntegrationKitPhaseInitialization --
	IntegrationKitPhaseInitialization IntegrationKitPhase = "Initialization"
	// IntegrationKitPhaseWaitingForPlatform --
	IntegrationKitPhaseWaitingForPlatform IntegrationKitPhase = "Waiting For Platform"
	// IntegrationKitPhaseBuildSubmitted --
	IntegrationKitPhaseBuildSubmitted IntegrationKitPhase = "Build Submitted"
	// IntegrationKitPhaseBuildRunning --
	IntegrationKitPhaseBuildRunning IntegrationKitPhase = "Build Running"
	// IntegrationKitPhaseReady --
	IntegrationKitPhaseReady IntegrationKitPhase = "Ready"
	// IntegrationKitPhaseError --
	IntegrationKitPhaseError IntegrationKitPhase = "Error"

	// IntegrationKitConditionPlatformAvailable --
	IntegrationKitConditionPlatformAvailable IntegrationKitConditionType = "IntegrationPlatformAvailable"
	// IntegrationKitConditionPlatformAvailableReason --
	IntegrationKitConditionPlatformAvailableReason string = "IntegrationPlatformAvailable"
)

// IntegrationKitCondition describes the state of a resource at a certain point.
type IntegrationKitCondition struct {
	// Type of integration condition.
	Type IntegrationKitConditionType `json:"type"`
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

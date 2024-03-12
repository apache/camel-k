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

// IntegrationProfileSpec applies user defined settings to the IntegrationProfile.
type IntegrationProfileSpec struct {
	// specify how to build the Integration/IntegrationKits
	Build IntegrationProfileBuildSpec `json:"build,omitempty"`
	// list of traits to be executed for all the Integration/IntegrationKits built from this IntegrationProfile
	Traits Traits `json:"traits,omitempty"`
	// configuration to be executed to all Kamelets controlled by this IntegrationProfile
	Kamelet IntegrationProfileKameletSpec `json:"kamelet,omitempty"`
}

// IntegrationProfileStatus defines the observed state of IntegrationProfile.
type IntegrationProfileStatus struct {
	IntegrationProfileSpec `json:",inline"`
	// ObservedGeneration is the most recent generation observed for this IntegrationProfile.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// defines in what phase the IntegrationProfile is found
	Phase IntegrationProfilePhase `json:"phase,omitempty"`
	// which are the conditions met (particularly useful when in ERROR phase)
	Conditions []IntegrationProfileCondition `json:"conditions,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=integrationprofiles,scope=Namespaced,shortName=ipr,categories=kamel;camel
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The integration profile phase"
// +kubebuilder:printcolumn:name="Default runtime",type=string,JSONPath=`.status.build.runtimeVersion`,description="The default runtime version"

// IntegrationProfile is the resource used to apply user defined settings to the Camel K operator behavior.
// It defines the behavior of all Custom Resources (`IntegrationKit`, `Integration`, `Kamelet`) in the given namespace.
type IntegrationProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntegrationProfileSpec   `json:"spec,omitempty"`
	Status IntegrationProfileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IntegrationProfileList contains a list of IntegrationProfile.
type IntegrationProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IntegrationProfile `json:"items"`
}

// IntegrationProfileBuildSpec contains profile related build information.
// This configuration can be used to tune the behavior of the Integration/IntegrationKit image builds.
type IntegrationProfileBuildSpec struct {
	// the Camel K Runtime dependency version
	RuntimeVersion string `json:"runtimeVersion,omitempty"`
	// the runtime used. Likely Camel Quarkus (we used to have main runtime which has been discontinued since version 1.5)
	RuntimeProvider RuntimeProvider `json:"runtimeProvider,omitempty"`
	// a base image that can be used as base layer for all images.
	// It can be useful if you want to provide some custom base image with further utility software
	BaseImage string `json:"baseImage,omitempty"`
	// the image registry used to push/pull Integration images
	Registry RegistrySpec `json:"registry,omitempty"`
	// how much time to wait before time out the pipeline process
	Timeout *metav1.Duration `json:"timeout,omitempty"`
	// Maven configuration used to build the Camel/Camel-Quarkus applications
	Maven MavenSpec `json:"maven,omitempty"`
}

// IntegrationProfileKameletSpec define the behavior for all the Kamelets controller by the IntegrationProfile.
type IntegrationProfileKameletSpec struct {
	// remote repository used to retrieve Kamelet catalog
	Repositories []KameletRepositorySpec `json:"repositories,omitempty"`
}

// IntegrationProfilePhase is the phase of an IntegrationProfile.
type IntegrationProfilePhase string

// IntegrationProfileConditionType defines the type of condition.
type IntegrationProfileConditionType string

const (
	// IntegrationProfileKind is the Kind name of the IntegrationProfile CR.
	IntegrationProfileKind string = "IntegrationProfile"

	// IntegrationProfilePhaseNone when the IntegrationProfile does not exist.
	IntegrationProfilePhaseNone IntegrationProfilePhase = ""
	// IntegrationProfilePhaseReady when the IntegrationProfile is ready.
	IntegrationProfilePhaseReady IntegrationProfilePhase = "Ready"
	// IntegrationProfilePhaseError when the IntegrationProfile had some error (see Conditions).
	IntegrationProfilePhaseError IntegrationProfilePhase = "Error"

	// IntegrationProfileConditionTypeCreated is the condition if the IntegrationProfile has been created.
	IntegrationProfileConditionTypeCreated IntegrationProfileConditionType = "Created"

	// IntegrationProfileConditionCreatedReason represents the reason that the IntegrationProfile is created.
	IntegrationProfileConditionCreatedReason = "IntegrationProfileCreated"
)

// IntegrationProfileCondition describes the state of a resource at a certain point.
type IntegrationProfileCondition struct {
	// Type of integration condition.
	Type IntegrationProfileConditionType `json:"type"`
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

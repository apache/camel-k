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

	"github.com/apache/camel-k/pkg/apis/camel/v1/trait"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "make generate-deepcopy" to regenerate code after modifying this file

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=integrationkits,scope=Namespaced,shortName=ik,categories=kamel;camel
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The integration kit phase"
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.metadata.labels.camel\.apache\.org\/kit\.type`,description="The integration kit type"
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.status.image`,description="The integration kit image"

// IntegrationKit defines a container image and additional configuration needed to run an `Integration`.
// An `IntegrationKit` is a generic image generally built from the requirements of an `Integration`, but agnostic to it,
// in order to be reused by any other `Integration` which has the same required set of capabilities. An `IntegrationKit`
// may be used for other kits as a base container layer, when the `incremental` build option is enabled.
type IntegrationKit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// the desired configuration
	Spec IntegrationKitSpec `json:"spec,omitempty"`
	// the actual status
	Status IntegrationKitStatus `json:"status,omitempty"`
}

// IntegrationKitSpec defines a container image and additional configurations required to kick off an `Integration` with certain features
type IntegrationKitSpec struct {
	// the container image as identified in the container registry
	Image string `json:"image,omitempty"`
	// a list of Camel dependecies used by this kit
	Dependencies []string `json:"dependencies,omitempty"`
	// the profile which is expected by this kit
	Profile TraitProfile `json:"profile,omitempty"`
	// traits that the kit will execute
	Traits IntegrationKitTraits `json:"traits,omitempty"`
	// configuration used by the kit
	// TODO: we should deprecate in future releases in favour of mount, openapi or camel traits
	Configuration []ConfigurationSpec `json:"configuration,omitempty"`
	// Maven repositories that can be used by the kit
	Repositories []string `json:"repositories,omitempty"`
}

// IntegrationKitTraits defines traits assigned to an `IntegrationKit`
type IntegrationKitTraits struct {
	// The builder trait is internally used to determine the best strategy to build and configure IntegrationKits.
	Builder *trait.BuilderTrait `property:"builder" json:"builder,omitempty"`
	// The Quarkus trait configures the Quarkus runtime.
	// It's enabled by default.
	// NOTE: Compiling to a native executable, i.e. when using `package-type=native`, is only supported for kamelets, as well as YAML and XML integrations. It also requires at least 4GiB of memory, so the Pod running the native build, that is either the operator Pod, or the build Pod (depending on the build strategy configured for the platform), must have enough memory available.
	Quarkus *trait.QuarkusTrait `property:"quarkus" json:"quarkus,omitempty"`
	// The Registry trait sets up Maven to use the Image registry as a Maven repository.
	Registry *trait.RegistryTrait `property:"registry" json:"registry,omitempty"`

	// The collection of addon trait configurations
	Addons map[string]AddonTrait `json:"addons,omitempty"`
}

// IntegrationKitStatus defines the observed state of IntegrationKit
type IntegrationKitStatus struct {
	// ObservedGeneration is the most recent generation observed for this IntegrationKit.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// phase of the kit
	Phase IntegrationKitPhase `json:"phase,omitempty"`
	// base image used by the kit
	BaseImage string `json:"baseImage,omitempty"`
	// actual image name of the kit
	Image string `json:"image,omitempty"`
	// actual image digest of the kit
	Digest string `json:"digest,omitempty"`
	// list of artifacts used by the kit
	Artifacts []Artifact `json:"artifacts,omitempty"`
	// failure reason (if any)
	Failure *Failure `json:"failure,omitempty"`
	// the runtime version for which this kit was configured
	RuntimeVersion string `json:"runtimeVersion,omitempty"`
	// the runtime provider for which this kit was configured
	RuntimeProvider RuntimeProvider `json:"runtimeProvider,omitempty"`
	// the platform for which this kit was configured
	Platform string `json:"platform,omitempty"`
	// the Camel K operator version for which this kit was configured
	Version string `json:"version,omitempty"`
	// a list of conditions which happened for the events related the kit
	Conditions []IntegrationKitCondition `json:"conditions,omitempty"`
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

	// IntegrationKitTypePlatform identifies a kit created by the platform
	IntegrationKitTypePlatform = "platform"
	// IntegrationKitTypeUser identifies a kit created by the user
	IntegrationKitTypeUser = "user"
	// IntegrationKitTypeExternal identifies a kit created by any third party
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

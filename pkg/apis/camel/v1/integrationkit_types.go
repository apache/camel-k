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

	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "make generate-deepcopy" to regenerate code after modifying this file

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=integrationkits,scope=Namespaced,shortName=ik,categories=kamel;camel
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Alias",type=string,JSONPath=`.metadata.labels.camel\.apache\.org\/kit\.alias`,description="The integration kit alias"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The integration kit phase"
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.metadata.labels.camel\.apache\.org\/kit\.type`,description="The integration kit type"
// +kubebuilder:printcolumn:name="Layout",type=string,JSONPath=`.metadata.labels.camel\.apache\.org\/kit\.layout`,description="The integration kit layout"
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.status.image`,description="The integration kit image"
// +kubebuilder:printcolumn:name="Root",type=string,JSONPath=`.status.rootImage`,description="The integration kit root image"

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

// IntegrationKitSpec defines a container image and additional configurations required to kick off an `Integration` with certain features.
type IntegrationKitSpec struct {
	// the container image as identified in the container registry
	Image string `json:"image,omitempty"`
	// a list of Camel dependencies used by this kit
	Dependencies []string `json:"dependencies,omitempty"`
	// the profile which is expected by this kit
	Profile TraitProfile `json:"profile,omitempty"`
	// traits that the kit will execute
	Traits IntegrationKitTraits `json:"traits,omitempty"`
	// Deprecated:
	// Use camel trait (camel.properties) to manage properties
	// Use mount trait (mount.configs) to manage configs
	// Use mount trait (mount.resources) to manage resources
	// Use mount trait (mount.volumes) to manage volumes
	// configuration used by the kit
	Configuration []ConfigurationSpec `json:"configuration,omitempty"`
	// Maven repositories that can be used by the kit
	Repositories []string `json:"repositories,omitempty"`
	// the sources to add at build time
	Sources []SourceSpec `json:"sources,omitempty"`
	// features offered by the IntegrationKit
	Capabilities []string `json:"capabilities,omitempty"`
}

// IntegrationKitTraits defines traits assigned to an `IntegrationKit`.
type IntegrationKitTraits struct {
	// The builder trait is internally used to determine the best strategy to build and configure IntegrationKits.
	Builder *trait.BuilderTrait `property:"builder" json:"builder,omitempty"`
	// The Camel trait sets up Camel configuration.
	Camel *trait.CamelTrait `property:"camel" json:"camel,omitempty"`
	// The Quarkus trait configures the Quarkus runtime.
	// It's enabled by default.
	// NOTE: Compiling to a native executable, requires at least 4GiB of memory, so the Pod running the native build must have enough memory available.
	Quarkus *trait.QuarkusTrait `property:"quarkus" json:"quarkus,omitempty"`
	// The Registry trait sets up Maven to use the Image registry as a Maven repository (support removed since version 2.5.0).
	// Deprecated: use jvm trait or read documentation.
	Registry *trait.RegistryTrait `property:"registry" json:"registry,omitempty"`
	// The collection of addon trait configurations
	Addons map[string]AddonTrait `json:"addons,omitempty"`
}

// IntegrationKitStatus defines the observed state of IntegrationKit.
type IntegrationKitStatus struct {
	// ObservedGeneration is the most recent generation observed for this IntegrationKit.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// phase of the kit
	Phase IntegrationKitPhase `json:"phase,omitempty"`
	// root image used by the kit (the first image from which the incremental image has started, typically a JDK/JRE base image)
	RootImage string `json:"rootImage,omitempty"`
	// base image used by the kit (could be another IntegrationKit)
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
	// the catalog used to build/operate the IntegrationKit.
	Catalog *Catalog `json:"catalog,omitempty"`
	// the platform for which this kit was configured
	Platform string `json:"platform,omitempty"`
	// the Camel K operator version for which this kit was configured
	Version string `json:"version,omitempty"`
	// a list of conditions which happened for the events related the kit
	Conditions []IntegrationKitCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// IntegrationKitList contains a list of IntegrationKit.
type IntegrationKitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IntegrationKit `json:"items"`
}

// IntegrationKitPhase --.
type IntegrationKitPhase string

// IntegrationKitConditionType --.
type IntegrationKitConditionType string

const (
	// IntegrationKitKind --.
	IntegrationKitKind string = "IntegrationKit"

	// IntegrationKitTypeLabel labels the kit type.
	IntegrationKitTypeLabel = "camel.apache.org/kit.type"

	// IntegrationKitTypePlatform identifies a Kit created by the platform.
	IntegrationKitTypePlatform = "platform"
	// IntegrationKitTypeUser identifies a Kit created by the user.
	IntegrationKitTypeUser = "user"
	// IntegrationKitTypeExternal identifies a Kit created by any third party.
	IntegrationKitTypeExternal = "external"
	// Deprecated: synthetic Integration Kits are replaced by non managed build Integrations.
	// IntegrationKitTypeSynthetic identifies a synthetic Kit (generated for any container image for which the operator cannot make any assumption).
	IntegrationKitTypeSynthetic = "synthetic"

	// IntegrationKitLayoutLabel labels the kit layout.
	IntegrationKitLayoutLabel = "camel.apache.org/kit.layout"

	// IntegrationKitLayoutFastJar labels a kit using the Quarkus fast-jar packaging.
	IntegrationKitLayoutFastJar = "fast-jar"
	// IntegrationKitLayoutNative labels a kit using the Quarkus native packaging.
	// Deprecated: no longer in use.
	IntegrationKitLayoutNative = "native"
	// IntegrationKitLayoutNativeSources labels a kit using the Quarkus native-sources packaging.
	IntegrationKitLayoutNativeSources = "native-sources"

	// IntegrationKitPriorityLabel labels the kit priority.
	IntegrationKitPriorityLabel = "camel.apache.org/kit.priority"

	// IntegrationKitPhaseNone --.
	IntegrationKitPhaseNone IntegrationKitPhase = ""
	// IntegrationKitPhaseInitialization --.
	IntegrationKitPhaseInitialization IntegrationKitPhase = "Initialization"
	// IntegrationKitPhaseWaitingForPlatform --.
	IntegrationKitPhaseWaitingForPlatform IntegrationKitPhase = "Waiting For Platform"
	// IntegrationKitPhaseBuildSubmitted --.
	IntegrationKitPhaseBuildSubmitted IntegrationKitPhase = "Build Submitted"
	// IntegrationKitPhaseBuildRunning --.
	IntegrationKitPhaseBuildRunning IntegrationKitPhase = "Build Running"
	// IntegrationKitPhaseWaitingForCatalog --.
	IntegrationKitPhaseWaitingForCatalog IntegrationKitPhase = "Waiting For Catalog"
	// IntegrationKitPhaseReady --.
	IntegrationKitPhaseReady IntegrationKitPhase = "Ready"
	// IntegrationKitPhaseError --.
	IntegrationKitPhaseError IntegrationKitPhase = "Error"

	// IntegrationKitConditionPlatformAvailable --.
	IntegrationKitConditionPlatformAvailable IntegrationKitConditionType = "IntegrationPlatformAvailable"
	// IntegrationKitConditionCatalogAvailable --.
	IntegrationKitConditionCatalogAvailable IntegrationKitConditionType = "CamelCatalogAvailable"
	// IntegrationKitConditionPlatformAvailableReason --.
	IntegrationKitConditionPlatformAvailableReason string = "IntegrationPlatformAvailable"
	// IntegrationKitConditionTraitInfo --.
	IntegrationKitConditionTraitInfo IntegrationKitConditionType = "TraitInfo"
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

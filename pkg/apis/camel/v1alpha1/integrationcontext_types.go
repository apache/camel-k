package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IntegrationContextSpec defines the desired state of IntegrationContext
type IntegrationContextSpec struct {
	Image         string                          `json:"image,omitempty"`
	Dependencies  []string                        `json:"dependencies,omitempty"`
	Profile       TraitProfile                    `json:"profile,omitempty"`
	Traits        map[string]IntegrationTraitSpec `json:"traits,omitempty"`
	Configuration []ConfigurationSpec             `json:"configuration,omitempty"`
	Repositories  []string                        `json:"repositories,omitempty"`
}

// IntegrationContextStatus defines the observed state of IntegrationContext
type IntegrationContextStatus struct {
	Phase       IntegrationContextPhase `json:"phase,omitempty"`
	BaseImage   string                  `json:"baseImage,omitempty"`
	Image       string                  `json:"image,omitempty"`
	PublicImage string                  `json:"publicImage,omitempty"`
	Digest      string                  `json:"digest,omitempty"`
	Artifacts   []Artifact              `json:"artifacts,omitempty"`
	Failure     *Failure                `json:"failure,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IntegrationContext is the Schema for the integrationcontexts API
// +k8s:openapi-gen=true
type IntegrationContext struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntegrationContextSpec   `json:"spec,omitempty"`
	Status IntegrationContextStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IntegrationContextList contains a list of IntegrationContext
type IntegrationContextList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IntegrationContext `json:"items"`
}

// IntegrationContextPhase --
type IntegrationContextPhase string

const (
	// IntegrationContextKind --
	IntegrationContextKind string = "IntegrationContext"

	// IntegrationContextTypePlatform --
	IntegrationContextTypePlatform = "platform"

	// IntegrationContextTypeUser --
	IntegrationContextTypeUser = "user"

	// IntegrationContextTypeExternal --
	IntegrationContextTypeExternal = "external"

	// IntegrationContextPhaseBuilding --
	IntegrationContextPhaseBuilding IntegrationContextPhase = "Building"
	// IntegrationContextPhaseReady --
	IntegrationContextPhaseReady IntegrationContextPhase = "Ready"
	// IntegrationContextPhaseError --
	IntegrationContextPhaseError IntegrationContextPhase = "Error"
	// IntegrationContextPhaseBuildFailureRecovery --
	IntegrationContextPhaseBuildFailureRecovery IntegrationContextPhase = "Build Failure Recovery"
)

func init() {
	SchemeBuilder.Register(&IntegrationContext{}, &IntegrationContextList{})
}

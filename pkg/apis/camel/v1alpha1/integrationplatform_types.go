package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IntegrationPlatformSpec defines the desired state of IntegrationPlatform
type IntegrationPlatformSpec struct {
	Cluster IntegrationPlatformCluster   `json:"cluster,omitempty"`
	Profile TraitProfile                 `json:"profile,omitempty"`
	Build   IntegrationPlatformBuildSpec `json:"build,omitempty"`
}

// IntegrationPlatformStatus defines the observed state of IntegrationPlatform
type IntegrationPlatformStatus struct {
	Phase IntegrationPlatformPhase `json:"phase,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IntegrationPlatform is the Schema for the integrationplatforms API
// +k8s:openapi-gen=true
type IntegrationPlatform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntegrationPlatformSpec   `json:"spec,omitempty"`
	Status IntegrationPlatformStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IntegrationPlatformList contains a list of IntegrationPlatform
type IntegrationPlatformList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IntegrationPlatform `json:"items"`
}

// IntegrationPlatformCluster is the kind of orchestration cluster the platform is installed into
type IntegrationPlatformCluster string

const (
	// IntegrationPlatformClusterOpenShift is used when targeting a OpenShift cluster
	IntegrationPlatformClusterOpenShift = "OpenShift"
	// IntegrationPlatformClusterKubernetes is used when targeting a Kubernetes cluster
	IntegrationPlatformClusterKubernetes = "Kubernetes"
)

// TraitProfile represents lists of traits that are enabled for the specific installation/integration
type TraitProfile string

const (
	// TraitProfileOpenShift is used by default on OpenShift clusters
	TraitProfileOpenShift = "OpenShift"
	// TraitProfileKubernetes is used by default on Kubernetes clusters
	TraitProfileKubernetes = "Kubernetes"
	// TraitProfileKnative is used by default on OpenShift/Kubernetes clusters powered by Knative
	TraitProfileKnative = "Knative"
)

var allTraitProfiles = []TraitProfile{TraitProfileOpenShift, TraitProfileKubernetes, TraitProfileKnative}

// IntegrationPlatformBuildSpec contains platform related build information
type IntegrationPlatformBuildSpec struct {
	PublishStrategy IntegrationPlatformBuildPublishStrategy `json:"publishStrategy,omitempty"`
	Registry        string                                  `json:"registry,omitempty"`
	Organization    string                                  `json:"organization,omitempty"`
	PushSecret      string                                  `json:"pushSecret,omitempty"`
	CamelVersion    string                                  `json:"camelVersion,omitempty"`
	Properties      map[string]string                       `json:"properties,omitempty"`
	Repositories    []string                                `json:"repositories,omitempty"`
}

// IntegrationPlatformBuildPublishStrategy enumerates all implemented build strategies
type IntegrationPlatformBuildPublishStrategy string

const (
	// IntegrationPlatformBuildPublishStrategyS2I performs a OpenShift binary S2I build
	IntegrationPlatformBuildPublishStrategyS2I = "S2I"

	// IntegrationPlatformBuildPublishStrategyKaniko performs
	IntegrationPlatformBuildPublishStrategyKaniko = "Kaniko"
)

// IntegrationPlatformPhase --
type IntegrationPlatformPhase string

const (
	// IntegrationPlatformKind --
	IntegrationPlatformKind string = "IntegrationPlatform"

	// IntegrationPlatformPhaseCreating --
	IntegrationPlatformPhaseCreating IntegrationPlatformPhase = "Creating"
	// IntegrationPlatformPhaseStarting --
	IntegrationPlatformPhaseStarting IntegrationPlatformPhase = "Starting"
	// IntegrationPlatformPhaseReady --
	IntegrationPlatformPhaseReady IntegrationPlatformPhase = "Ready"
	// IntegrationPlatformPhaseError --
	IntegrationPlatformPhaseError IntegrationPlatformPhase = "Error"
	// IntegrationPlatformPhaseDuplicate --
	IntegrationPlatformPhaseDuplicate IntegrationPlatformPhase = "Duplicate"
)

func init() {
	SchemeBuilder.Register(&IntegrationPlatform{}, &IntegrationPlatformList{})
}

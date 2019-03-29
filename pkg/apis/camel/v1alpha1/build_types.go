package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BuildSpec defines the desired state of Build
type BuildSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Meta           metav1.ObjectMeta       `json:"meta,omitempty"`
	CamelVersion   string                  `json:"camelVersion,omitempty"`
	RuntimeVersion string                  `json:"runtimeVersion,omitempty"`
	Platform       IntegrationPlatformSpec `json:"platform,omitempty"`
	Dependencies   []string                `json:"dependencies,omitempty"`
	Repositories   []string                `json:"repositories,omitempty"`
	Steps          []string                `json:"steps,omitempty"`
	Image          string                  `json:"image,omitempty"`
	//Resources      []Resource              `json:"resources,omitempty"`
}

// BuildStatus defines the observed state of Build
type BuildStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	BaseImage   string     `json:"baseImage,omitempty"`
	Image       string     `json:"image,omitempty"`
	PublicImage string     `json:"publicImage,omitempty"`
	Error       string     `json:"error,omitempty"`
	Phase       BuildPhase `json:"phase,omitempty"`
	Artifacts   []Artifact `json:"artifacts,omitempty"`
	//Task        Task
}

// BuildPhase --
type BuildPhase string

const (
	// BuildPhaseInitial --
	BuildPhaseInitial BuildPhase = ""
	// BuildPhaseScheduling --
	BuildPhaseScheduling BuildPhase = "Scheduling"
	// BuildPhasePending --
	BuildPhasePending BuildPhase = "Pending"
	// BuildPhaseRunning --
	BuildPhaseRunning BuildPhase = "Running"
	// BuildPhaseSucceeded --
	BuildPhaseSucceeded BuildPhase = "Succeeded"
	// BuildPhaseFailed --
	BuildPhaseFailed BuildPhase = "Failed"
	// BuildPhaseInterrupted --
	BuildPhaseInterrupted = "Interrupted"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Build is the Schema for the builds API
// +k8s:openapi-gen=true
type Build struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildSpec   `json:"spec,omitempty"`
	Status BuildStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildList contains a list of Build
type BuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Build `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Build{}, &BuildList{})
}

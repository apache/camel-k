package v1alpha1

import (
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	openapi "github.com/go-openapi/spec"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KameletSpec defines the desired state of Kamelet
type KameletSpec struct {
	Info          KameletInfo         `json:"info,omitempty"`
	Parameters    []KameletParameter  `json:"parameters,omitempty"`
	Authorization AuthorizationSpec   `json:"authorization,omitempty"`
	Sources       *camelv1.SourceSpec `json:"sources,omitempty"`
	Flow          *camelv1.Flow       `json:"flow,omitempty"`
	Consumes      *EventType          `json:"consumes,omitempty"`
	Produces      *EventType          `json:"produces,omitempty"`
	Dependencies  []string            `json:"dependencies,omitempty"`
}

type EventType struct {
	Type   string      `json:"type,omitempty"`
	Schema *DataSchema `json:"schema,omitempty"`
}

type DataSchema struct {
	Ref     *corev1.ObjectReference `json:"ref,omitempty"`
	Content *openapi.Schema         `json:"schema,omitempty"`
}

type KameletInfo struct {
	DisplayName string      `json:"displayName,omitempty"`
	Description string      `json:"description,omitempty"`
	Group       string      `json:"group,omitempty"`
	Icon        KameletIcon `json:"icon,omitempty"`
}

type KameletIcon struct {
	Data      string `json:"data,omitempty"`
	MediaType string `json:"mediaType,omitempty"`
}

type KameletParameter struct {
	Name        string            `json:"name,omitempty"`
	Required    bool              `json:"required,omitempty"`
	Description string            `json:"description,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Schema      *openapi.Schema   `json:"schema,omitempty"`
}

// AuthorizationSpec is TODO (oauth information)
type AuthorizationSpec struct {
}

// KameletStatus defines the observed state of Kamelet
type KameletStatus struct {
	Phase      KameletPhase       `json:"phase,omitempty"`
	Conditions []KameletCondition `json:"conditions,omitempty"`
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

type KameletPhase string

const (
	// KameletKind --
	KameletKind string = "Kamelet"

	// KameletPhaseNone --
	KameletPhaseNone KameletPhase = ""
	// KameletPhaseReady --
	KameletPhaseReady KameletPhase = "Ready"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Kamelet is the Schema for the kamelets API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=kamelets,scope=Namespaced
type Kamelet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KameletSpec   `json:"spec,omitempty"`
	Status KameletStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KameletList contains a list of Kamelet
type KameletList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kamelet `json:"items"`
}

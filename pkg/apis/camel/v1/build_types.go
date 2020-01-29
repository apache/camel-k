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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BuildSpec defines the desired state of Build
type BuildSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Tasks []Task `json:"tasks,omitempty"`
}

// Task --
type Task struct {
	Builder *BuilderTask `json:"builder,omitempty"`
	Kaniko  *KanikoTask  `json:"kaniko,omitempty"`
}

// BaseTask --
type BaseTask struct {
	Name         string               `json:"name,omitempty"`
	Affinity     *corev1.Affinity     `json:"affinity,omitempty"`
	Volumes      []corev1.Volume      `json:"volumes,omitempty"`
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
}

// ImageTask --
type ImageTask struct {
	BaseTask `json:",inline"`
	Image    string          `json:"image,omitempty"`
	Args     []string        `json:"args,omitempty"`
	Env      []corev1.EnvVar `json:"env,omitempty"`
}

// KanikoTask --
type KanikoTask struct {
	ImageTask  `json:",inline"`
	BuiltImage string `json:"builtImage,omitempty"`
}

// BuilderTask --
type BuilderTask struct {
	BaseTask     `json:",inline"`
	Meta         metav1.ObjectMeta `json:"meta,omitempty"`
	Image        string            `json:"image,omitempty"`
	BaseImage    string            `json:"baseImage,omitempty"`
	Runtime      RuntimeSpec       `json:"runtime,omitempty"`
	Sources      []SourceSpec      `json:"sources,omitempty"`
	Resources    []ResourceSpec    `json:"resources,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
	Steps        []string          `json:"steps,omitempty"`
	Maven        MavenSpec         `json:"maven,omitempty"`
	BuildDir     string            `json:"buildDir,omitempty"`
	Properties   map[string]string `json:"properties,omitempty"`
	Timeout      metav1.Duration   `json:"timeout,omitempty"`
}

// BuildStatus defines the observed state of Build
type BuildStatus struct {
	Phase      BuildPhase       `json:"phase,omitempty"`
	Image      string           `json:"image,omitempty"`
	BaseImage  string           `json:"baseImage,omitempty"`
	Artifacts  []Artifact       `json:"artifacts,omitempty"`
	Error      string           `json:"error,omitempty"`
	Failure    *Failure         `json:"failure,omitempty"`
	StartedAt  metav1.Time      `json:"startedAt,omitempty"`
	Platform   string           `json:"platform,omitempty"`
	Conditions []BuildCondition `json:"conditions,omitempty"`
	// Change to Duration / ISO 8601 when CRD uses OpenAPI spec v3
	// https://github.com/OAI/OpenAPI-Specification/issues/845
	Duration string `json:"duration,omitempty"`
}

// BuildPhase --
type BuildPhase string

// BuildConditionType --
type BuildConditionType string

const (
	// BuildKind --
	BuildKind string = "Build"

	// BuildPhaseNone --
	BuildPhaseNone BuildPhase = ""
	// BuildPhaseInitialization --
	BuildPhaseInitialization BuildPhase = "Initialization"
	// BuildPhaseWaitingForPlatform --
	BuildPhaseWaitingForPlatform BuildPhase = "Waiting For Platform"
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
	// BuildPhaseError --
	BuildPhaseError BuildPhase = "Error"

	// BuildConditionPlatformAvailable --
	BuildConditionPlatformAvailable BuildConditionType = "IntegrationPlatformAvailable"
	// BuildConditionPlatformAvailableReason --
	BuildConditionPlatformAvailableReason string = "IntegrationPlatformAvailable"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Build is the Schema for the builds API
// +k8s:openapi-gen=true
// +genclient
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

// BuildCondition describes the state of a resource at a certain point.
type BuildCondition struct {
	// Type of integration condition.
	Type BuildConditionType `json:"type"`
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

func init() {
	SchemeBuilder.Register(&Build{}, &BuildList{})
}

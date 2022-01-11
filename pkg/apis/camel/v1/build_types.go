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

// BuildSpec defines the Build to be executed
type BuildSpec struct {
	// The sequence of Build tasks to be performed as part of the Build execution.
	Tasks []Task `json:"tasks,omitempty"`
	// The strategy that should be used to perform the Build.
	Strategy BuildStrategy `json:"strategy,omitempty"`
	// Timeout defines the Build maximum execution duration.
	// The Build deadline is set to the Build start time plus the Timeout duration.
	// If the Build deadline is exceeded, the Build context is canceled,
	// and its phase set to BuildPhaseFailed.
	// +kubebuilder:validation:Format=duration
	Timeout metav1.Duration `json:"timeout,omitempty"`
}

// Task --
type Task struct {
	Builder  *BuilderTask  `json:"builder,omitempty"`
	Buildah  *BuildahTask  `json:"buildah,omitempty"`
	Kaniko   *KanikoTask   `json:"kaniko,omitempty"`
	Spectrum *SpectrumTask `json:"spectrum,omitempty"`
	S2i      *S2iTask      `json:"s2i,omitempty"`
}

// BaseTask --
type BaseTask struct {
	Name string `json:"name,omitempty"`
}

// BuilderTask --
type BuilderTask struct {
	BaseTask     `json:",inline"`
	BaseImage    string         `json:"baseImage,omitempty"`
	Runtime      RuntimeSpec    `json:"runtime,omitempty"`
	Sources      []SourceSpec   `json:"sources,omitempty"`
	Resources    []ResourceSpec `json:"resources,omitempty"`
	Dependencies []string       `json:"dependencies,omitempty"`
	Steps        []string       `json:"steps,omitempty"`
	Maven        MavenBuildSpec `json:"maven,omitempty"`
	BuildDir     string         `json:"buildDir,omitempty"`
}

// MavenBuildSpec --
type MavenBuildSpec struct {
	MavenSpec `json:",inline"`
	// The Maven repositories.
	Repositories []Repository `json:"repositories,omitempty"`
}

// PublishTask --
type PublishTask struct {
	ContextDir string       `json:"contextDir,omitempty"`
	BaseImage  string       `json:"baseImage,omitempty"`
	Image      string       `json:"image,omitempty"`
	Registry   RegistrySpec `json:"registry,omitempty"`
}

// BuildahTask --
type BuildahTask struct {
	BaseTask    `json:",inline"`
	PublishTask `json:",inline"`
	Verbose     *bool `json:"verbose,omitempty"`
}

// KanikoTask --
type KanikoTask struct {
	BaseTask    `json:",inline"`
	PublishTask `json:",inline"`
	Verbose     *bool           `json:"verbose,omitempty"`
	Cache       KanikoTaskCache `json:"cache,omitempty"`
}

// KanikoTaskCache --
type KanikoTaskCache struct {
	Enabled               *bool  `json:"enabled,omitempty"`
	PersistentVolumeClaim string `json:"persistentVolumeClaim,omitempty"`
}

// SpectrumTask --
type SpectrumTask struct {
	BaseTask    `json:",inline"`
	PublishTask `json:",inline"`
}

// S2iTask --
type S2iTask struct {
	BaseTask   `json:",inline"`
	ContextDir string `json:"contextDir,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

// BuildStatus defines the observed state of Build
type BuildStatus struct {
	Phase      BuildPhase       `json:"phase,omitempty"`
	Image      string           `json:"image,omitempty"`
	Digest     string           `json:"digest,omitempty"`
	BaseImage  string           `json:"baseImage,omitempty"`
	Artifacts  []Artifact       `json:"artifacts,omitempty"`
	Error      string           `json:"error,omitempty"`
	Failure    *Failure         `json:"failure,omitempty"`
	StartedAt  *metav1.Time     `json:"startedAt,omitempty"`
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
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=builds,scope=Namespaced,shortName=ikb,categories=kamel;camel
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The build phase"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="The time at which the build was created"
// +kubebuilder:printcolumn:name="Started",type=date,JSONPath=`.status.startedAt`,description="The time at which the build was last (re-)started"
// Change format to 'duration' when CRD uses OpenAPI spec v3 (https://github.com/OAI/OpenAPI-Specification/issues/845)
// +kubebuilder:printcolumn:name="Duration",type=string,JSONPath=`.status.duration`,description="The build last execution duration"
// +kubebuilder:printcolumn:name="Attempts",type=integer,JSONPath=`.status.failure.recovery.attempt`,description="The number of execution attempts"

// Build is the Schema for the builds API
type Build struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildSpec   `json:"spec,omitempty"`
	Status BuildStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

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
	// A human-readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

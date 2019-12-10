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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IntegrationPlatformSpec defines the desired state of IntegrationPlatform
type IntegrationPlatformSpec struct {
	Cluster       IntegrationPlatformCluster       `json:"cluster,omitempty"`
	Profile       TraitProfile                     `json:"profile,omitempty"`
	Build         IntegrationPlatformBuildSpec     `json:"build,omitempty"`
	Resources     IntegrationPlatformResourcesSpec `json:"resources,omitempty"`
	Traits        map[string]TraitSpec             `json:"traits,omitempty"`
	Configuration []ConfigurationSpec              `json:"configuration,omitempty"`
}

// IntegrationPlatformResourcesSpec contains platform related resources
type IntegrationPlatformResourcesSpec struct {
	Kits []string `json:"kits,omitempty"`
}

// IntegrationPlatformStatus defines the observed state of IntegrationPlatform
type IntegrationPlatformStatus struct {
	IntegrationPlatformSpec `json:",inline"`

	Phase      IntegrationPlatformPhase       `json:"phase,omitempty"`
	Conditions []IntegrationPlatformCondition `json:"conditions,omitempty"`
	Version    string                         `json:"version,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IntegrationPlatform is the Schema for the integrationplatforms API
// +k8s:openapi-gen=true
// +genclient
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

// AllIntegrationPlatformClusters --
var AllIntegrationPlatformClusters = []IntegrationPlatformCluster{IntegrationPlatformClusterOpenShift, IntegrationPlatformClusterKubernetes}

// TraitProfile represents lists of traits that are enabled for the specific installation/integration
type TraitProfile string

const (
	// TraitProfileOpenShift is used by default on OpenShift clusters
	TraitProfileOpenShift = "OpenShift"
	// TraitProfileKubernetes is used by default on Kubernetes clusters
	TraitProfileKubernetes = "Kubernetes"
	// TraitProfileKnative is used by default on OpenShift/Kubernetes clusters powered by Knative
	TraitProfileKnative = "Knative"
	// DefaultTraitProfile is the trait profile used as default when no other profile is set
	DefaultTraitProfile = TraitProfileKubernetes
)

// AllTraitProfiles contains all allowed profiles
var AllTraitProfiles = []TraitProfile{TraitProfileKubernetes, TraitProfileKnative, TraitProfileOpenShift}

// IntegrationPlatformBuildSpec contains platform related build information
type IntegrationPlatformBuildSpec struct {
	BuildStrategy         IntegrationPlatformBuildStrategy        `json:"buildStrategy,omitempty"`
	PublishStrategy       IntegrationPlatformBuildPublishStrategy `json:"publishStrategy,omitempty"`
	CamelVersion          string                                  `json:"camelVersion,omitempty"`
	RuntimeVersion        string                                  `json:"runtimeVersion,omitempty"`
	RuntimeProvider       *RuntimeProvider                        `json:"runtimeProvider,omitempty"`
	BaseImage             string                                  `json:"baseImage,omitempty"`
	Properties            map[string]string                       `json:"properties,omitempty"`
	Registry              IntegrationPlatformRegistrySpec         `json:"registry,omitempty"`
	Timeout               *metav1.Duration                        `json:"timeout,omitempty"`
	PersistentVolumeClaim string                                  `json:"persistentVolumeClaim,omitempty"`
	Maven                 MavenSpec                               `json:"maven,omitempty"`
	HTTPProxySecret       string                                  `json:"httpProxySecret,omitempty"`
	KanikoBuildCache      *bool                                   `json:"kanikoBuildCache,omitempty"`
}

// IntegrationPlatformRegistrySpec --
type IntegrationPlatformRegistrySpec struct {
	Insecure     bool   `json:"insecure,omitempty"`
	Address      string `json:"address,omitempty"`
	Secret       string `json:"secret,omitempty"`
	Organization string `json:"organization,omitempty"`
}

// IntegrationPlatformBuildStrategy enumerates all implemented build strategies
type IntegrationPlatformBuildStrategy string

const (
	// IntegrationPlatformBuildStrategyRoutine performs the build in a routine
	IntegrationPlatformBuildStrategyRoutine = "routine"

	// IntegrationPlatformBuildStrategyPod performs the build in a pod
	IntegrationPlatformBuildStrategyPod = "pod"
)

// IntegrationPlatformBuildPublishStrategy enumerates all implemented publish strategies
type IntegrationPlatformBuildPublishStrategy string

const (
	// IntegrationPlatformBuildPublishStrategyS2I performs a OpenShift binary S2I build
	IntegrationPlatformBuildPublishStrategyS2I = "S2I"

	// IntegrationPlatformBuildPublishStrategyKaniko performs
	IntegrationPlatformBuildPublishStrategyKaniko = "Kaniko"
)

// IntegrationPlatformPhase --
type IntegrationPlatformPhase string

// IntegrationPlatformConditionType --
type IntegrationPlatformConditionType string

const (
	// IntegrationPlatformKind --
	IntegrationPlatformKind string = "IntegrationPlatform"

	// IntegrationPlatformPhaseCreating --
	IntegrationPlatformPhaseCreating IntegrationPlatformPhase = "Creating"
	// IntegrationPlatformPhaseWarming --
	IntegrationPlatformPhaseWarming IntegrationPlatformPhase = "Warming"
	// IntegrationPlatformPhaseReady --
	IntegrationPlatformPhaseReady IntegrationPlatformPhase = "Ready"
	// IntegrationPlatformPhaseError --
	IntegrationPlatformPhaseError IntegrationPlatformPhase = "Error"
	// IntegrationPlatformPhaseDuplicate --
	IntegrationPlatformPhaseDuplicate IntegrationPlatformPhase = "Duplicate"
)

// IntegrationPlatformCondition describes the state of a resource at a certain point.
type IntegrationPlatformCondition struct {
	// Type of integration condition.
	Type IntegrationPlatformConditionType `json:"type"`
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
	SchemeBuilder.Register(&IntegrationPlatform{}, &IntegrationPlatformList{})
}

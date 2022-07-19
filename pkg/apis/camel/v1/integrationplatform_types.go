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

// IntegrationPlatformSpec defines the desired state of IntegrationPlatform
type IntegrationPlatformSpec struct {
	Cluster       IntegrationPlatformCluster       `json:"cluster,omitempty"`
	Profile       TraitProfile                     `json:"profile,omitempty"`
	Build         IntegrationPlatformBuildSpec     `json:"build,omitempty"`
	Resources     IntegrationPlatformResourcesSpec `json:"resources,omitempty"`
	Traits        map[string]TraitSpec             `json:"traits,omitempty"`
	Configuration []ConfigurationSpec              `json:"configuration,omitempty"`
	Kamelet       IntegrationPlatformKameletSpec   `json:"kamelet,omitempty"`
}

// IntegrationPlatformResourcesSpec contains platform related resources
type IntegrationPlatformResourcesSpec struct {
}

// IntegrationPlatformStatus defines the observed state of IntegrationPlatform
type IntegrationPlatformStatus struct {
	IntegrationPlatformSpec `json:",inline"`

	Phase      IntegrationPlatformPhase       `json:"phase,omitempty"`
	Conditions []IntegrationPlatformCondition `json:"conditions,omitempty"`
	Version    string                         `json:"version,omitempty"`
	Info       map[string]string              `json:"info,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=integrationplatforms,scope=Namespaced,shortName=ip,categories=kamel;camel
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The integration platform phase"

// IntegrationPlatform is the Schema for the integrationplatforms API
type IntegrationPlatform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntegrationPlatformSpec   `json:"spec,omitempty"`
	Status IntegrationPlatformStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IntegrationPlatformList contains a list of IntegrationPlatform
type IntegrationPlatformList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IntegrationPlatform `json:"items"`
}

// IntegrationPlatformCluster is the kind of orchestration cluster the platform is installed into
type IntegrationPlatformCluster string

const (
	// IntegrationPlatformClusterOpenShift is used when targeting an OpenShift cluster
	IntegrationPlatformClusterOpenShift IntegrationPlatformCluster = "OpenShift"
	// IntegrationPlatformClusterKubernetes is used when targeting a Kubernetes cluster
	IntegrationPlatformClusterKubernetes IntegrationPlatformCluster = "Kubernetes"
)

// AllIntegrationPlatformClusters --
var AllIntegrationPlatformClusters = []IntegrationPlatformCluster{IntegrationPlatformClusterOpenShift, IntegrationPlatformClusterKubernetes}

// IntegrationPlatformBuildSpec contains platform related build information
type IntegrationPlatformBuildSpec struct {
	BuildStrategy         BuildStrategy                           `json:"buildStrategy,omitempty"`
	PublishStrategy       IntegrationPlatformBuildPublishStrategy `json:"publishStrategy,omitempty"`
	RuntimeVersion        string                                  `json:"runtimeVersion,omitempty"`
	RuntimeProvider       RuntimeProvider                         `json:"runtimeProvider,omitempty"`
	BaseImage             string                                  `json:"baseImage,omitempty"`
	Registry              RegistrySpec                            `json:"registry,omitempty"`
	Timeout               *metav1.Duration                        `json:"timeout,omitempty"`
	PersistentVolumeClaim string                                  `json:"persistentVolumeClaim,omitempty"`
	Maven                 MavenSpec                               `json:"maven,omitempty"`
	KanikoBuildCache      *bool                                   `json:"kanikoBuildCache,omitempty"`
}

// IntegrationPlatformKameletSpec --
type IntegrationPlatformKameletSpec struct {
	Repositories []IntegrationPlatformKameletRepositorySpec `json:"repositories,omitempty"`
}

// IntegrationPlatformKameletRepositorySpec --
type IntegrationPlatformKameletRepositorySpec struct {
	URI string `json:"uri,omitempty"`
}

// IntegrationPlatformBuildPublishStrategy enumerates all implemented publish strategies
type IntegrationPlatformBuildPublishStrategy string

const (
	// IntegrationPlatformBuildPublishStrategyBuildah --
	IntegrationPlatformBuildPublishStrategyBuildah IntegrationPlatformBuildPublishStrategy = "Buildah"
	// IntegrationPlatformBuildPublishStrategyKaniko --
	IntegrationPlatformBuildPublishStrategyKaniko IntegrationPlatformBuildPublishStrategy = "Kaniko"
	// IntegrationPlatformBuildPublishStrategyS2I --
	IntegrationPlatformBuildPublishStrategyS2I IntegrationPlatformBuildPublishStrategy = "S2I"
	// IntegrationPlatformBuildPublishStrategySpectrum --
	IntegrationPlatformBuildPublishStrategySpectrum IntegrationPlatformBuildPublishStrategy = "Spectrum"
)

// IntegrationPlatformBuildPublishStrategies --
var IntegrationPlatformBuildPublishStrategies = []IntegrationPlatformBuildPublishStrategy{
	IntegrationPlatformBuildPublishStrategyBuildah,
	IntegrationPlatformBuildPublishStrategyKaniko,
	IntegrationPlatformBuildPublishStrategyS2I,
	IntegrationPlatformBuildPublishStrategySpectrum,
}

// IntegrationPlatformPhase --
type IntegrationPlatformPhase string

// IntegrationPlatformConditionType --
type IntegrationPlatformConditionType string

const (
	// IntegrationPlatformKind --
	IntegrationPlatformKind string = "IntegrationPlatform"

	// IntegrationPlatformPhaseNone --
	IntegrationPlatformPhaseNone IntegrationPlatformPhase = ""
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
	// A human-readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

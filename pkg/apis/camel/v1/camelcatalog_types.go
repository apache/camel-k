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

const (
	// CamelCatalogKind -- .
	CamelCatalogKind string = "CamelCatalog"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=camelcatalogs,scope=Namespaced,shortName=cc,categories=kamel;camel
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Runtime Provider",type=string,JSONPath=`.spec.runtime.provider`,description="The Runtime provider"
// +kubebuilder:printcolumn:name="Runtime Version",type=string,JSONPath=`.spec.runtime.version`,description="The Runtime version"
// +kubebuilder:printcolumn:name="Runtime Camel Version",type=string,JSONPath=`.spec.runtime.metadata.camel\.version`,description="The Camel version"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The catalog phase"

// CamelCatalog represents the languages, components, data formats and capabilities enabled on a given runtime provider. The catalog may be statically generated.
type CamelCatalog struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// the actual state of the catalog
	Status CamelCatalogStatus `json:"status,omitempty" yaml:"status,omitempty"`
	// the desired state of the catalog
	Spec CamelCatalogSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// CamelCatalogList contains a list of CamelCatalog.
type CamelCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CamelCatalog `json:"items"`
}

// CamelCatalogSpec specify what features a Camel runtime provides.
type CamelCatalogSpec struct {
	// the runtime targeted for the catalog
	Runtime RuntimeSpec `json:"runtime" yaml:"runtime"`
	// artifacts required by this catalog
	Artifacts map[string]CamelArtifact `json:"artifacts" yaml:"artifacts"`
	// loaders required by this catalog
	Loaders map[string]CamelLoader `json:"loaders" yaml:"loaders"`
}

// CamelCatalogStatus defines the observed state of CamelCatalog.
type CamelCatalogStatus struct {
	// ObservedGeneration is the most recent generation observed for this Catalog.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// the actual phase
	Phase CamelCatalogPhase `json:"phase,omitempty"`
	// a list of events happened for the CamelCatalog
	Conditions []CamelCatalogCondition `json:"conditions,omitempty"`
	// the container image available for building an application with this catalog
	Image string `json:"image,omitempty"`
}

// CamelCatalogPhase -- .
type CamelCatalogPhase string

const (
	// CamelCatalogPhaseNone --.
	CamelCatalogPhaseNone CamelCatalogPhase = ""
	// CamelCatalogPhaseReady --.
	CamelCatalogPhaseReady CamelCatalogPhase = "Ready"
	// CamelCatalogPhaseError --.
	CamelCatalogPhaseError CamelCatalogPhase = "Error"
)

// CamelCatalogCondition describes the state of a resource at a certain point.
type CamelCatalogCondition struct {
	// Type of CamelCatalog condition.
	Type CamelCatalogConditionType `json:"type"`
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

// CamelCatalogConditionType --.
type CamelCatalogConditionType string

const (
	// CamelCatalogConditionReady --.
	CamelCatalogConditionReady CamelCatalogConditionType = "Ready"
)

// CamelScheme represents the scheme used to identify a component in a URI (ie, timer in a timer:xyz endpoint URI).
type CamelScheme struct {
	// the ID (ie, timer in a timer:xyz URI)
	ID string `json:"id" yaml:"id"`
	// is a passive scheme
	Passive bool `json:"passive" yaml:"passive"`
	// is a HTTP based scheme
	HTTP bool `json:"http" yaml:"http"`
	// required scope for consumer
	Consumer CamelSchemeScope `json:"consumer,omitempty" yaml:"consumer,omitempty"`
	// required scope for producers
	Producer CamelSchemeScope `json:"producer,omitempty" yaml:"producer,omitempty"`
}

// CamelSchemeScope contains scoped information about a scheme.
type CamelSchemeScope struct {
	// list of dependencies needed for this scope
	Dependencies []CamelArtifactDependency `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// CamelArtifactExclusion represents an exclusion clause.
type CamelArtifactExclusion struct {
	// Maven Group
	GroupID string `json:"groupId" yaml:"groupId"`
	// Maven Artifact
	ArtifactID string `json:"artifactId" yaml:"artifactId"`
}

// CamelArtifactDependency represent a maven's dependency.
type CamelArtifactDependency struct {
	// the maven dependency
	MavenArtifact `json:",inline" yaml:",inline"`
	// provide a list of artifacts to exclude for this dependency
	Exclusions []CamelArtifactExclusion `json:"exclusions,omitempty" yaml:"exclusions,omitempty"`
}

// CamelArtifact represent the configuration for a feature offered by Camel.
type CamelArtifact struct {
	// Base Camel Artifact dependency
	CamelArtifactDependency `json:",inline" yaml:",inline"`
	// accepted URI schemes
	Schemes []CamelScheme `json:"schemes,omitempty" yaml:"schemes,omitempty"`
	// accepted languages
	Languages []string `json:"languages,omitempty" yaml:"languages,omitempty"`
	// accepted data formats
	DataFormats []string `json:"dataformats,omitempty" yaml:"dataformats,omitempty"`
	// required dependencies
	Dependencies []CamelArtifactDependency `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	// the Java types used by the artifact feature (ie, component, data format, ...)
	JavaTypes []string `json:"javaTypes,omitempty" yaml:"javaTypes,omitempty"`
}

// CamelLoader represents the configuration required to load a DSL.
type CamelLoader struct {
	// the base Maven artifact required
	MavenArtifact `json:",inline" yaml:",inline"`
	// a list of DSLs supported
	Languages []string `json:"languages,omitempty" yaml:"languages,omitempty"`
	// a list of additional dependencies required beside the base one
	Dependencies []MavenArtifact `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	// the metadata of the loader
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

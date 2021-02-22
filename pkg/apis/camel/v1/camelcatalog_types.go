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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CamelScheme --
type CamelScheme struct {
	ID       string           `json:"id" yaml:"id"`
	Passive  bool             `json:"passive" yaml:"passive"`
	HTTP     bool             `json:"http" yaml:"http"`
	Consumer CamelSchemeScope `json:"consumer,omitempty" yaml:"consumer,omitempty"`
	Producer CamelSchemeScope `json:"producer,omitempty" yaml:"producer,omitempty"`
}

// CamelSchemeScope contains scoped information about a scheme
type CamelSchemeScope struct {
	Dependencies []CamelArtifactDependency `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// CamelArtifactExclusion --
type CamelArtifactExclusion struct {
	GroupID    string `json:"groupId" yaml:"groupId"`
	ArtifactID string `json:"artifactId" yaml:"artifactId"`
}

// CamelArtifactDependency represent a maven's dependency
type CamelArtifactDependency struct {
	MavenArtifact `json:",inline" yaml:",inline"`
	Exclusions    []CamelArtifactExclusion `json:"exclusions,omitempty" yaml:"exclusions,omitempty"`
}

// CamelArtifact --
type CamelArtifact struct {
	CamelArtifactDependency `json:",inline" yaml:",inline"`
	Schemes                 []CamelScheme             `json:"schemes,omitempty" yaml:"schemes,omitempty"`
	Languages               []string                  `json:"languages,omitempty" yaml:"languages,omitempty"`
	DataFormats             []string                  `json:"dataformats,omitempty" yaml:"dataformats,omitempty"`
	Dependencies            []CamelArtifactDependency `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	JavaTypes               []string                  `json:"javaTypes,omitempty" yaml:"javaTypes,omitempty"`
}

// CamelLoader --
type CamelLoader struct {
	MavenArtifact `json:",inline" yaml:",inline"`
	Languages     []string        `json:"languages,omitempty" yaml:"languages,omitempty"`
	Dependencies  []MavenArtifact `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// CamelCatalogSpec defines the desired state of CamelCatalog
type CamelCatalogSpec struct {
	Runtime   RuntimeSpec              `json:"runtime" yaml:"runtime"`
	Artifacts map[string]CamelArtifact `json:"artifacts" yaml:"artifacts"`
	Loaders   map[string]CamelLoader   `json:"loaders" yaml:"loaders"`
}

// CamelCatalogStatus defines the observed state of CamelCatalog
type CamelCatalogStatus struct {
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=camelcatalogs,scope=Namespaced,shortName=cc,categories=kamel;camel
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Runtime Version",type=string,JSONPath=`.spec.runtime.version`,description="The Camel K Runtime version"
// +kubebuilder:printcolumn:name="Runtime Provider",type=string,JSONPath=`.spec.runtime.provider`,description="The Camel K Runtime provider"

// CamelCatalog is the Schema for the camelcatalogs API
type CamelCatalog struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Status CamelCatalogStatus `json:"status,omitempty" yaml:"status,omitempty"`
	Spec   CamelCatalogSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// CamelCatalogList contains a list of CamelCatalog
type CamelCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CamelCatalog `json:"items"`
}

const (
	// CamelCatalogKind --
	CamelCatalogKind string = "CamelCatalog"
)

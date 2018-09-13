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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IntegrationList --
type IntegrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Integration `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Integration --
type Integration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              IntegrationSpec   `json:"spec"`
	Status            IntegrationStatus `json:"status,omitempty"`
}

// IntegrationSpec --
type IntegrationSpec struct {
	Replicas     *int32         `json:"replicas,omitempty"`
	Source       SourceSpec     `json:"source,omitempty"`
	Context      string         `json:"context,omitempty"`
	Dependencies []string       `json:"dependencies,omitempty"`
	Properties   []PropertySpec `json:"properties,omitempty"`
}

// SourceSpec --
type SourceSpec struct {
	Name     string `json:"name,omitempty"`
	Content  string `json:"content,omitempty"`
	Language string `json:"language,omitempty"`
}

// IntegrationStatus --
type IntegrationStatus struct {
	Phase  IntegrationPhase `json:"phase,omitempty"`
	Digest string           `json:"digest,omitempty"`
	Image  string           `json:"image,omitempty"`
}

// IntegrationPhase --
type IntegrationPhase string

const (
	// IntegrationKind --
	IntegrationKind string = "Integration"

	// IntegrationPhaseBuilding --
	IntegrationPhaseBuilding IntegrationPhase = "Building"
	// IntegrationPhaseDeploying --
	IntegrationPhaseDeploying IntegrationPhase = "Deploying"
	// IntegrationPhaseRunning --
	IntegrationPhaseRunning IntegrationPhase = "Running"
	// IntegrationPhaseError --
	IntegrationPhaseError IntegrationPhase = "Error"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IntegrationContextList --
type IntegrationContextList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []IntegrationContext `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IntegrationContext --
type IntegrationContext struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              IntegrationContextSpec   `json:"spec"`
	Status            IntegrationContextStatus `json:"status,omitempty"`
}

// IntegrationContextSpec --
type IntegrationContextSpec struct {
	Dependencies []string          `json:"dependencies,omitempty"`
	Properties   []PropertySpec    `json:"properties,omitempty"`
	Environment  []EnvironmentSpec `json:"environment,omitempty"`
}

// PropertySpec --
type PropertySpec struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// EnvironmentSpec --
type EnvironmentSpec struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// IntegrationContextStatus --
type IntegrationContextStatus struct {
	Phase  IntegrationContextPhase `json:"phase,omitempty"`
	Image  string                  `json:"image,omitempty"`
	Digest string                  `json:"digest,omitempty"`
}

// IntegrationContextPhase --
type IntegrationContextPhase string

const (
	// IntegrationContextKind --
	IntegrationContextKind string = "IntegrationContext"

	// IntegrationContextPhaseBuilding --
	IntegrationContextPhaseBuilding IntegrationContextPhase = "Building"
	// IntegrationContextPhaseDeploying --
	IntegrationContextPhaseDeploying IntegrationContextPhase = "Deploying"
	// IntegrationContextPhaseReady --
	IntegrationContextPhaseReady IntegrationContextPhase = "Ready"
	// IntegrationContextPhaseError --
	IntegrationContextPhaseError IntegrationContextPhase = "Error"
)

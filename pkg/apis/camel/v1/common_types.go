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

// ConfigurationSpec --
type ConfigurationSpec struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Artifact --
type Artifact struct {
	ID       string `json:"id" yaml:"id"`
	Location string `json:"location,omitempty" yaml:"location,omitempty"`
	Target   string `json:"target,omitempty" yaml:"target,omitempty"`
	Checksum string `json:"checksum,omitempty" yaml:"checksum,omitempty"`
}

// Failure --
type Failure struct {
	Reason   string          `json:"reason"`
	Time     metav1.Time     `json:"time"`
	Recovery FailureRecovery `json:"recovery"`
}

// FailureRecovery --
type FailureRecovery struct {
	Attempt    int `json:"attempt"`
	AttemptMax int `json:"attemptMax"`
	// +optional
	AttemptTime metav1.Time `json:"attemptTime"`
}

// A TraitSpec contains the configuration of a trait
type TraitSpec struct {
	// TraitConfiguration --
	Configuration TraitConfiguration `json:"configuration"`
}

type TraitConfiguration struct {
	RawMessage `json:",inline"`
}

// RawMessage is a raw encoded JSON value.
// It implements Marshaler and Unmarshaler and can
// be used to delay JSON decoding or precompute a JSON encoding.
// +kubebuilder:validation:Type=object
// +kubebuilder:validation:Format=""
// +kubebuilder:pruning:PreserveUnknownFields
type RawMessage []byte

// +kubebuilder:object:generate=false
// Configurable --
type Configurable interface {
	Configurations() []ConfigurationSpec
}

// +kubebuilder:object:generate=false
// PlatformInjectable --
type PlatformInjectable interface {
	SetIntegrationPlatform(platform *IntegrationPlatform)
}

// MavenSpec --
type MavenSpec struct {
	LocalRepository string           `json:"localRepository,omitempty"`
	Settings        ValueSource      `json:"settings,omitempty"`
	Timeout         *metav1.Duration `json:"timeout,omitempty"`
}

// ValueSource --
type ValueSource struct {
	// Selects a key of a ConfigMap.
	ConfigMapKeyRef *corev1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
	// Selects a key of a secret.
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// MavenArtifact --
type MavenArtifact struct {
	GroupID    string `json:"groupId" yaml:"groupId"`
	ArtifactID string `json:"artifactId" yaml:"artifactId"`
	Version    string `json:"version,omitempty" yaml:"version,omitempty"`
}

// RuntimeSpec --
type RuntimeSpec struct {
	Version          string                `json:"version" yaml:"version"`
	Provider         RuntimeProvider       `json:"provider" yaml:"provider"`
	ApplicationClass string                `json:"applicationClass" yaml:"applicationClass"`
	Dependencies     []MavenArtifact       `json:"dependencies" yaml:"dependencies"`
	Metadata         map[string]string     `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Capabilities     map[string]Capability `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
}

// Capability --
type Capability struct {
	Dependencies []MavenArtifact   `json:"dependencies" yaml:"dependencies"`
	Metadata     map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

const (
	// ServiceTypeUser --
	ServiceTypeUser = "user"

	// CapabilityRest --
	CapabilityRest = "rest"
	// CapabilityHealth --
	CapabilityHealth = "health"
	// CapabilityCron --
	CapabilityCron = "cron"
	// CapabilityPlatformHTTP --
	CapabilityPlatformHTTP = "platform-http"
	// CapabilityCircuitBreaker
	CapabilityCircuitBreaker = "circuit-breaker"
	// CapabilityTracing --
	CapabilityTracing = "tracing"
	// CapabilityMaster --
	CapabilityMaster = "master"
)

// +kubebuilder:object:generate=false
// ResourceCondition is a common type for all conditions
type ResourceCondition interface {
	GetType() string
	GetStatus() corev1.ConditionStatus
	GetLastUpdateTime() metav1.Time
	GetLastTransitionTime() metav1.Time
	GetReason() string
	GetMessage() string
}

// Flow is an unstructured object representing a Camel Flow in YAML/JSON DSL
type Flow struct {
	RawMessage `json:",inline"`
}

// RuntimeProvider --
type RuntimeProvider string

const (
	// RuntimeProviderQuarkus --
	RuntimeProviderQuarkus RuntimeProvider = "quarkus"
)

// ResourceType --
type ResourceType string

// ResourceSpec --
type ResourceSpec struct {
	DataSpec  `json:",inline"`
	Type      ResourceType `json:"type,omitempty"`
	MountPath string       `json:"mountPath,omitempty"`
}

const (
	// ResourceTypeData --
	ResourceTypeData ResourceType = "data"
	// ResourceTypeOpenAPI --
	ResourceTypeOpenAPI ResourceType = "openapi"
)

// SourceSpec --
type SourceSpec struct {
	DataSpec `json:",inline"`
	Language Language `json:"language,omitempty"`
	// Loader is an optional id of the org.apache.camel.k.RoutesLoader that will
	// interpret this source at runtime
	Loader string `json:"loader,omitempty"`
	// Interceptors are optional identifiers the org.apache.camel.k.RoutesLoader
	// uses to pre/post process sources
	Interceptors []string `json:"interceptors,omitempty"`
	// Type defines the kind of source described by this object
	Type SourceType `json:"type,omitempty"`
	// List of property names defined in the source (e.g. if type is "template")
	PropertyNames []string `json:"property-names,omitempty"`
}

type SourceType string

const (
	SourceTypeDefault  SourceType = ""
	SourceTypeTemplate SourceType = "template"
)

// DataSpec --
type DataSpec struct {
	Name        string `json:"name,omitempty"`
	Content     string `json:"content,omitempty"`
	RawContent  []byte `json:"rawContent,omitempty"`
	ContentRef  string `json:"contentRef,omitempty"`
	ContentKey  string `json:"contentKey,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Compression bool   `json:"compression,omitempty"`
}

// Language --
type Language string

const (
	// LanguageJavaSource --
	LanguageJavaSource Language = "java"
	// LanguageGroovy --
	LanguageGroovy Language = "groovy"
	// LanguageJavaScript --
	LanguageJavaScript Language = "js"
	// LanguageXML --
	LanguageXML Language = "xml"
	// LanguageKotlin --
	LanguageKotlin Language = "kts"
	// LanguageYaml --
	LanguageYaml Language = "yaml"
)

// Languages is the list of all supported languages
var Languages = []Language{
	LanguageJavaSource,
	LanguageGroovy,
	LanguageJavaScript,
	LanguageXML,
	LanguageKotlin,
	LanguageYaml,
}

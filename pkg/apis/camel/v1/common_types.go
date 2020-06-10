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
	"encoding/json"

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
	Attempt     int         `json:"attempt"`
	AttemptMax  int         `json:"attemptMax"`
	AttemptTime metav1.Time `json:"attemptTime"`
}

// A TraitSpec contains the configuration of a trait
type TraitSpec struct {
	Configuration map[string]string `json:"configuration,omitempty"`
}

// Configurable --
type Configurable interface {
	Configurations() []ConfigurationSpec
}

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
)

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
type Flow json.RawMessage

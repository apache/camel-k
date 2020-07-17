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
	Attempt    int `json:"attempt"`
	AttemptMax int `json:"attemptMax"`
	// +optional
	AttemptTime metav1.Time `json:"attemptTime"`
}

// Traits --
type Traits struct {
	Affinity       *AffinityTrait       `property:"affinity" json:"affinity,omitempty"`
	Builder        *BuilderTrait        `property:"builder" json:"builder,omitempty"`
	Camel          *CamelTrait          `property:"camel" json:"camel,omitempty"`
	Container      *ContainerTrait      `property:"container" json:"container,omitempty"`
	Cron           *CronTrait           `property:"cron" json:"cron,omitempty"`
	Dependencies   *DependenciesTrait   `property:"dependencies" json:"dependencies,omitempty"`
	Deployer       *DeployerTrait       `property:"deployer" json:"deployer,omitempty"`
	Deployment     *DeploymentTrait     `property:"deployment" json:"deployment,omitempty"`
	Environment    *EnvironmentTrait    `property:"environment" json:"environment,omitempty"`
	GC             *GCTrait             `property:"gc" json:"gc,omitempty"`
	Ingress        *IngressTrait        `property:"ingress" json:"ingress,omitempty"`
	Istio          *IstioTrait          `property:"istio" json:"istio,omitempty"`
	Jolokia        *JolokiaTrait        `property:"jolokia" json:"jolokia,omitempty"`
	JVM            *JvmTrait            `property:"jvm" json:"jvm,omitempty"`
	Knative        *KnativeTrait        `property:"knative" json:"knative,omitempty"`
	KnativeService *KnativeServiceTrait `property:"knative-service" json:"knative-service,omitempty"`
	OpenAPI        *OpenAPITrait        `property:"openapi" json:"openapi,omitempty"`
	Owner          *OwnerTrait          `property:"owner" json:"owner,omitempty"`
	Platform       *PlatformTrait       `property:"platform" json:"platform,omitempty"`
	Prometheus     *PrometheusTrait     `property:"prometheus" json:"prometheus,omitempty"`
	PullSecret     *PullSecretTrait     `property:"pull-secret" json:"pull-secret,omitempty"`
	Quarkus        *QuarkusTrait        `property:"quarkus" json:"quarkus,omitempty"`
	Route          *RouteTrait          `property:"route" json:"route,omitempty"`
	Service        *ServiceTrait        `property:"service" json:"service,omitempty"`
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
// +kubebuilder:validation:Type=object
type Flow struct {
	json.RawMessage `json:",inline"`
}

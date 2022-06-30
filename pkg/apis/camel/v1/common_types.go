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

	"github.com/apache/camel-k/pkg/apis/camel/v1/trait"
)

const (
	// TraitAnnotationPrefix represents the prefix used for traits annotations
	TraitAnnotationPrefix = "trait.camel.apache.org/"
	// OperatorIDAnnotation operator id annotation label
	OperatorIDAnnotation = "camel.apache.org/operator.id"
	// SecondaryPlatformAnnotation secondary platform annotation label
	SecondaryPlatformAnnotation = "camel.apache.org/secondary.platform"
	// PlatformSelectorAnnotation platform id annotation label
	PlatformSelectorAnnotation = "camel.apache.org/platform.id"
)

// BuildStrategy specifies how the Build should be executed.
// It will trigger a Maven process that will take care of producing the expected Camel/Camel-Quarkus runtime.
// +kubebuilder:validation:Enum=routine;pod
type BuildStrategy string

const (
	// BuildStrategyRoutine performs the build in a routine (will be executed as a process inside the same Camel K operator `Pod`).
	// A routine may be preferred to a `pod` strategy since it reuse the Maven repository dependency cached locally. It is executed as
	// a parallel process, so you may need to consider the quantity of concurrent build process running simultaneously.
	BuildStrategyRoutine BuildStrategy = "routine"
	// BuildStrategyPod performs the build in a `Pod` (will schedule a new builder ephemeral `Pod` which will take care of the build action).
	// This strategy has the limitation that every build will have to download all the dependencies required by the Maven build.
	BuildStrategyPod BuildStrategy = "pod"
)

// BuildStrategies is a list of strategies allowed for the build
var BuildStrategies = []BuildStrategy{
	BuildStrategyRoutine,
	BuildStrategyPod,
}

// ConfigurationSpec represents a generic configuration specification
type ConfigurationSpec struct {
	// represents the type of configuration, ie: property, configmap, secret, ...
	Type string `json:"type"`
	// the value to assign to the configuration (syntax may vary depending on the `Type`)
	Value string `json:"value"`
	// Deprecated: no longer used
	ResourceType string `json:"resourceType,omitempty"`
	// Deprecated: no longer used
	ResourceMountPoint string `json:"resourceMountPoint,omitempty"`
	// Deprecated: no longer used
	ResourceKey string `json:"resourceKey,omitempty"`
}

// Artifact represents a materialized artifact (a jar dependency or in general a file used by the build)
type Artifact struct {
	// the identification (GAV for maven dependencies or file name for other file types)
	ID string `json:"id" yaml:"id"`
	// where it is located in the builder `Pod`
	Location string `json:"location,omitempty" yaml:"location,omitempty"`
	// the expected location in the runtime
	Target string `json:"target,omitempty" yaml:"target,omitempty"`
	// a checksum (SHA1) of the content
	Checksum string `json:"checksum,omitempty" yaml:"checksum,omitempty"`
}

// Failure represent a message specifying the reason and the time of an event failure
type Failure struct {
	// a short text specifying the reason
	Reason string `json:"reason"`
	// the time when the failure has happened
	Time metav1.Time `json:"time"`
	// the recovery attempted for this failure
	Recovery FailureRecovery `json:"recovery"`
}

// FailureRecovery defines the attempts to recover a failure
type FailureRecovery struct {
	// attempt number
	Attempt int `json:"attempt"`
	// maximum number of attempts
	AttemptMax int `json:"attemptMax"`
	// time of the attempt execution
	// +optional
	AttemptTime metav1.Time `json:"attemptTime"`
}

// Traits represents the collection of trait configurations
type Traits struct {
	// The configuration of Affinity trait
	Affinity *trait.AffinityTrait `property:"affinity" json:"affinity,omitempty"`
	// The configuration of Builder trait
	Builder *trait.BuilderTrait `property:"builder" json:"builder,omitempty"`
	// The configuration of Camel trait
	Camel *trait.CamelTrait `property:"camel" json:"camel,omitempty"`
	// The configuration of Container trait
	Container *trait.ContainerTrait `property:"container" json:"container,omitempty"`
	// The configuration of Cron trait
	Cron *trait.CronTrait `property:"cron" json:"cron,omitempty"`
	// The configuration of Dependencies trait
	Dependencies *trait.DependenciesTrait `property:"dependencies" json:"dependencies,omitempty"`
	// The configuration of Deployer trait
	Deployer *trait.DeployerTrait `property:"deployer" json:"deployer,omitempty"`
	// The configuration of Deployment trait
	Deployment *trait.DeploymentTrait `property:"deployment" json:"deployment,omitempty"`
	// The configuration of Environment trait
	Environment *trait.EnvironmentTrait `property:"environment" json:"environment,omitempty"`
	// The configuration of Error Handler trait
	ErrorHandler *trait.ErrorHandlerTrait `property:"error-handler" json:"error-handler,omitempty"`
	// The configuration of GC trait
	GC *trait.GCTrait `property:"gc" json:"gc,omitempty"`
	// The configuration of Health trait
	Health *trait.HealthTrait `property:"health" json:"health,omitempty"`
	// The configuration of Ingress trait
	Ingress *trait.IngressTrait `property:"ingress" json:"ingress,omitempty"`
	// The configuration of Istio trait
	Istio *trait.IstioTrait `property:"istio" json:"istio,omitempty"`
	// The configuration of Jolokia trait
	Jolokia *trait.JolokiaTrait `property:"jolokia" json:"jolokia,omitempty"`
	// The configuration of JVM trait
	JVM *trait.JVMTrait `property:"jvm" json:"jvm,omitempty"`
	// The configuration of Kamelets trait
	Kamelets *trait.KameletsTrait `property:"kamelets" json:"kamelets,omitempty"`
	// The configuration of Knative trait
	Knative *trait.KnativeTrait `property:"knative" json:"knative,omitempty"`
	// The configuration of Knative Service trait
	KnativeService *trait.KnativeServiceTrait `property:"knative-service" json:"knative-service,omitempty"`
	// The configuration of Logging trait
	Logging *trait.LoggingTrait `property:"logging" json:"logging,omitempty"`
	// The configuration of Mount trait
	Mount *trait.MountTrait `property:"mount" json:"mount,omitempty"`
	// The configuration of OpenAPI trait
	OpenAPI *trait.OpenAPITrait `property:"openapi" json:"openapi,omitempty"`
	// The configuration of Owner trait
	Owner *trait.OwnerTrait `property:"owner" json:"owner,omitempty"`
	// The configuration of PDB trait
	PDB *trait.PDBTrait `property:"pdb" json:"pdb,omitempty"`
	// The configuration of Platform trait
	Platform *trait.PlatformTrait `property:"platform" json:"platform,omitempty"`
	// The configuration of Pod trait
	Pod *trait.PodTrait `property:"pod" json:"pod,omitempty"`
	// The configuration of Prometheus trait
	Prometheus *trait.PrometheusTrait `property:"prometheus" json:"prometheus,omitempty"`
	// The configuration of Pull Secret trait
	PullSecret *trait.PullSecretTrait `property:"pull-secret" json:"pull-secret,omitempty"`
	// The configuration of Quarkus trait
	Quarkus *trait.QuarkusTrait `property:"quarkus" json:"quarkus,omitempty"`
	// The configuration of Registry trait
	Registry *trait.RegistryTrait `property:"registry" json:"registry,omitempty"`
	// The configuration of Route trait
	Route *trait.RouteTrait `property:"route" json:"route,omitempty"`
	// The configuration of Service trait
	Service *trait.ServiceTrait `property:"service" json:"service,omitempty"`
	// The configuration of Service Binding trait
	ServiceBinding *trait.ServiceBindingTrait `property:"service-binding" json:"service-binding,omitempty"`
	// The configuration of Toleration trait
	Toleration *trait.TolerationTrait `property:"toleration" json:"toleration,omitempty"`

	// The extension point with addon traits
	Addons map[string]AddonTrait `json:"addons,omitempty"`

	// Deprecated: for backward compatibility.
	Keda *TraitSpec `property:"keda" json:"keda,omitempty"`
	// Deprecated: for backward compatibility.
	Master *TraitSpec `property:"master" json:"master,omitempty"`
	// Deprecated: for backward compatibility.
	Strimzi *TraitSpec `property:"strimzi" json:"strimzi,omitempty"`
	// Deprecated: for backward compatibility.
	ThreeScale *TraitSpec `property:"3scale" json:"3scale,omitempty"`
	// Deprecated: for backward compatibility.
	Tracing *TraitSpec `property:"tracing" json:"tracing,omitempty"`
}

// AddonTrait represents the configuration of an addon trait
type AddonTrait struct {
	// Generic raw message, typically a map containing the keys (trait parameters) and the values (either single text or array)
	RawMessage `json:",inline"`
}

// A TraitSpec contains the configuration of a trait
// Deprecated: superceded by each Trait type, left for backward compatibility.
type TraitSpec struct {
	// TraitConfiguration parameters configuration
	Configuration TraitConfiguration `json:"configuration"`
}

// TraitConfiguration represents the expected configuration for a given trait parameter
// Deprecated: superceded by each Trait type, left for backward compatibility.
type TraitConfiguration struct {
	// generic raw message, typically a map containing the keys (trait parameters) and the values (either single text or array)
	RawMessage `json:",inline"`
}

// +kubebuilder:validation:Type=object
// +kubebuilder:validation:Format=""
// +kubebuilder:pruning:PreserveUnknownFields

// RawMessage is a raw encoded JSON value.
// It implements Marshaler and Unmarshaler and can
// be used to delay JSON decoding or precompute a JSON encoding.
type RawMessage []byte

// +kubebuilder:object:generate=false

// Configurable --
type Configurable interface {
	Configurations() []ConfigurationSpec
}

// RegistrySpec provides the configuration for the container registry
type RegistrySpec struct {
	// if the container registry is insecure (ie, http only)
	Insecure bool `json:"insecure,omitempty"`
	// the URI to access
	Address string `json:"address,omitempty"`
	// the secret where credentials are stored
	Secret string `json:"secret,omitempty"`
	// the configmap which stores the Certificate Authority
	CA string `json:"ca,omitempty"`
	// the registry organization
	Organization string `json:"organization,omitempty"`
}

// ValueSource --
type ValueSource struct {
	// Selects a key of a ConfigMap.
	ConfigMapKeyRef *corev1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
	// Selects a key of a secret.
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// RuntimeSpec represents the configuration for the Java runtime in charge to execute the Camel application
type RuntimeSpec struct {
	// Camel K Runtime version
	Version string `json:"version" yaml:"version"`
	// Camel main application provider, ie, Camel Quarkus
	Provider RuntimeProvider `json:"provider" yaml:"provider"`
	// application entry point (main) to be executed
	ApplicationClass string `json:"applicationClass" yaml:"applicationClass"`
	// list of dependencies needed to run the application
	Dependencies []MavenArtifact `json:"dependencies" yaml:"dependencies"`
	// set of metadata
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	// features offered by this runtime
	Capabilities map[string]Capability `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
}

// Capability is a particular feature which requires a well known set of dependencies
type Capability struct {
	Dependencies []MavenArtifact `json:"dependencies" yaml:"dependencies"`
	// Deprecated: not in use
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

const (
	// ServiceTypeUser service user type label marker
	ServiceTypeUser = "user"

	// CapabilityRest defines the REST API service exposure capability
	CapabilityRest = "rest"
	// CapabilityHealth defines the health monitoring capability
	CapabilityHealth = "health"
	// CapabilityCron defines the cron execution capability
	CapabilityCron = "cron"
	// CapabilityPlatformHTTP defines the http service exposure capability
	CapabilityPlatformHTTP = "platform-http"
	// CapabilityCircuitBreaker defines the circuit breaker capability
	CapabilityCircuitBreaker = "circuit-breaker"
	// CapabilityTracing defines the tracing (opentracing) capability
	CapabilityTracing = "tracing"
	// CapabilityMaster defines the master capability
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

// RuntimeProvider is the provider chosen for the runtime
type RuntimeProvider string

const (
	// RuntimeProviderQuarkus Camel Quarkus runtime
	RuntimeProviderQuarkus RuntimeProvider = "quarkus"
)

// ResourceSpec represent an attached resource which will be materialized as a file on the running `Pod`
// TODO: we should deprecate in future releases in favour of mount, openapi or camel traits
type ResourceSpec struct {
	// the content of the resource
	DataSpec `json:",inline"`
	// the kind of data to expect
	Type ResourceType `json:"type,omitempty"`
	// the mount path on destination `Pod`
	MountPath string `json:"mountPath,omitempty"`
}

// ResourceType defines a kind of resource
type ResourceType string

const (
	// ResourceTypeData represents a generic data resource (text or binary)
	ResourceTypeData ResourceType = "data"
	// ResourceTypeConfig represents a configuration resource (text only)
	ResourceTypeConfig ResourceType = "config"
	// ResourceTypeOpenAPI represents an OpenAPI specification resource
	ResourceTypeOpenAPI ResourceType = "openapi"
)

// SourceSpec defines the configuration for one or more routes to be executed in a certain Camel DSL language
type SourceSpec struct {
	// contains configuration related to the source code
	DataSpec `json:",inline"`
	// specify which is the language (Camel DSL) used to interpret this source code
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

// SourceType represents an available source type
type SourceType string

const (
	// SourceTypeDefault is used to represent a source code
	SourceTypeDefault SourceType = ""
	// SourceTypeTemplate is used to represent a template
	SourceTypeTemplate SourceType = "template"
	// SourceTypeErrorHandler is used to represent an error handler
	SourceTypeErrorHandler SourceType = "errorHandler"
)

// DataSpec represents the way the source is materialized in the running `Pod`
type DataSpec struct {
	// the name of the specification
	Name string `json:"name,omitempty"`
	// the path where the file is stored
	Path string `json:"path,omitempty"`
	// the source code (plain text)
	Content string `json:"content,omitempty"`
	// the source code (binary)
	RawContent []byte `json:"rawContent,omitempty"`
	// the confimap reference holding the source content
	ContentRef string `json:"contentRef,omitempty"`
	// the confimap key holding the source content
	ContentKey string `json:"contentKey,omitempty"`
	// the content type (tipically text or binary)
	ContentType string `json:"contentType,omitempty"`
	// if the content is compressed (base64 encrypted)
	Compression bool `json:"compression,omitempty"`
}

// Language represents a supported language (Camel DSL)
type Language string

const (
	// LanguageJavaSource used for Java
	LanguageJavaSource Language = "java"
	// LanguageGroovy used for Groovy
	LanguageGroovy Language = "groovy"
	// LanguageJavaScript  used for Javascript
	LanguageJavaScript Language = "js"
	// LanguageXML used for XML
	LanguageXML Language = "xml"
	// LanguageKotlin used for Kotlin
	LanguageKotlin Language = "kts"
	// LanguageYaml used for YAML
	LanguageYaml Language = "yaml"
	// LanguageKamelet used for Kamelets
	LanguageKamelet Language = "kamelet"
)

// Languages is the list of all supported languages
var Languages = []Language{
	LanguageJavaSource,
	LanguageGroovy,
	LanguageJavaScript,
	LanguageXML,
	LanguageKotlin,
	LanguageYaml,
	LanguageKamelet,
}

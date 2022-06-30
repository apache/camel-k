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

// +genclient
// +genclient:method=GetScale,verb=get,subresource=scale,result=k8s.io/api/autoscaling/v1.Scale
// +genclient:method=UpdateScale,verb=update,subresource=scale,input=k8s.io/api/autoscaling/v1.Scale,result=k8s.io/api/autoscaling/v1.Scale
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=integrations,scope=Namespaced,shortName=it,categories=kamel;camel
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The integration phase"
// +kubebuilder:printcolumn:name="Kit",type=string,JSONPath=`.status.integrationKit.name`,description="The integration kit"
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.status.replicas`,description="The number of pods"

// Integration is the Schema for the integrations API
type Integration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// the desired Integration specification
	Spec IntegrationSpec `json:"spec,omitempty"`
	// the status of the Integration
	Status IntegrationStatus `json:"status,omitempty"`
}

// IntegrationSpec specifies the configuration of an Integration.
// The Integration will be watched by the operator which will be in charge to run the related application, according to the configuration specified.
type IntegrationSpec struct {
	// the number of `Pods` needed for the running Integration
	Replicas *int32 `json:"replicas,omitempty"`
	// the sources which contain the Camel routes to run
	Sources []SourceSpec `json:"sources,omitempty"`
	// a source in YAML DSL language which contain the routes to run
	Flows []Flow `json:"flows,omitempty"`
	// Deprecated:
	// Use container trait (container.resources) to manage resources
	// Use openapi trait (openapi.configmaps) to manage OpenAPIs specifications
	Resources []ResourceSpec `json:"resources,omitempty"`
	// the reference of the `IntegrationKit` which is used for this Integration
	IntegrationKit *corev1.ObjectReference `json:"integrationKit,omitempty"`
	// the list of Camel or Maven dependencies required by the Integration
	Dependencies []string `json:"dependencies,omitempty"`
	// the profile needed to run this Integration
	Profile TraitProfile `json:"profile,omitempty"`
	// the traits needed to run this Integration
	Traits Traits `json:"traits,omitempty"`
	// Pod template customization
	PodTemplate *PodSpecTemplate `json:"template,omitempty"`
	// Deprecated:
	// Use camel trait (camel.properties) to manage properties
	// Use container trait (mount.configs) to manage configs
	// Use container trait (mount.resources) to manage resources
	// Use container trait (mount.volumes) to manage volumes
	Configuration []ConfigurationSpec `json:"configuration,omitempty"`
	// additional Maven repositories to be used
	Repositories []string `json:"repositories,omitempty"`
	// custom SA to use for the Integration
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// IntegrationStatus defines the observed state of Integration
type IntegrationStatus struct {
	// ObservedGeneration is the most recent generation observed for this Integration.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// the actual phase
	Phase IntegrationPhase `json:"phase,omitempty"`
	// the digest calculated for this Integration
	Digest string `json:"digest,omitempty"`
	// the container image used
	Image string `json:"image,omitempty"`
	// a list of dependencies needed by the application
	Dependencies []string `json:"dependencies,omitempty"`
	// the profile needed to run this Integration
	Profile TraitProfile `json:"profile,omitempty"`
	// the reference of the `IntegrationKit` which is used for this Integration
	IntegrationKit *corev1.ObjectReference `json:"integrationKit,omitempty"`
	// The IntegrationPlatform watching this Integration
	Platform string `json:"platform,omitempty"`
	// a list of sources generated for this Integration
	GeneratedSources []SourceSpec `json:"generatedSources,omitempty"`
	// Deprecated:
	// a list of resources generated for this Integration
	GeneratedResources []ResourceSpec `json:"generatedResources,omitempty"`
	// the runtime version targeted for this Integration
	RuntimeVersion string `json:"runtimeVersion,omitempty"`
	// the runtime provider targeted for this Integration
	RuntimeProvider RuntimeProvider `json:"runtimeProvider,omitempty"`
	// Deprecated:
	// a list of configuration specification
	Configuration []ConfigurationSpec `json:"configuration,omitempty"`
	// a list of events happened for the Integration
	Conditions []IntegrationCondition `json:"conditions,omitempty"`
	// the operator version
	Version string `json:"version,omitempty"`
	// the number of replicas
	Replicas *int32 `json:"replicas,omitempty"`
	// label selector
	Selector string `json:"selector,omitempty"`
	// features offered by the Integration
	Capabilities []string `json:"capabilities,omitempty"`
	// the timestamp representing the last time when this integration was initialized.
	InitializationTimestamp *metav1.Time `json:"lastInitTimestamp,omitempty"`
}

// +kubebuilder:object:root=true

// IntegrationList contains a list of Integration
type IntegrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Integration `json:"items"`
}

// IntegrationPhase --
type IntegrationPhase string

// IntegrationConditionType --
type IntegrationConditionType string

const (
	// IntegrationKind --
	IntegrationKind string = "Integration"

	// IntegrationPhaseNone --
	IntegrationPhaseNone IntegrationPhase = ""
	// IntegrationPhaseInitialization --
	IntegrationPhaseInitialization IntegrationPhase = "Initialization"
	// IntegrationPhaseWaitingForPlatform --
	IntegrationPhaseWaitingForPlatform IntegrationPhase = "Waiting For Platform"
	// IntegrationPhaseBuildingKit --
	IntegrationPhaseBuildingKit IntegrationPhase = "Building Kit"
	// IntegrationPhaseDeploying --
	IntegrationPhaseDeploying IntegrationPhase = "Deploying"
	// IntegrationPhaseRunning --
	IntegrationPhaseRunning IntegrationPhase = "Running"
	// IntegrationPhaseError --
	IntegrationPhaseError IntegrationPhase = "Error"

	// IntegrationConditionKitAvailable --
	IntegrationConditionKitAvailable IntegrationConditionType = "IntegrationKitAvailable"
	// IntegrationConditionPlatformAvailable --
	IntegrationConditionPlatformAvailable IntegrationConditionType = "IntegrationPlatformAvailable"
	// IntegrationConditionDeploymentAvailable --
	IntegrationConditionDeploymentAvailable IntegrationConditionType = "DeploymentAvailable"
	// IntegrationConditionServiceAvailable --
	IntegrationConditionServiceAvailable IntegrationConditionType = "ServiceAvailable"
	// IntegrationConditionKnativeServiceAvailable --
	IntegrationConditionKnativeServiceAvailable IntegrationConditionType = "KnativeServiceAvailable"
	// IntegrationConditionCronJobAvailable --
	IntegrationConditionCronJobAvailable IntegrationConditionType = "CronJobAvailable"
	// IntegrationConditionExposureAvailable --
	IntegrationConditionExposureAvailable IntegrationConditionType = "ExposureAvailable"
	// IntegrationConditionPrometheusAvailable --
	IntegrationConditionPrometheusAvailable IntegrationConditionType = "PrometheusAvailable"
	// IntegrationConditionJolokiaAvailable --
	IntegrationConditionJolokiaAvailable IntegrationConditionType = "JolokiaAvailable"
	// IntegrationConditionProbesAvailable --
	IntegrationConditionProbesAvailable IntegrationConditionType = "ProbesAvailable"
	// IntegrationConditionReady --
	IntegrationConditionReady IntegrationConditionType = "Ready"

	// IntegrationConditionKitAvailableReason --
	IntegrationConditionKitAvailableReason string = "IntegrationKitAvailable"
	// IntegrationConditionPlatformAvailableReason --
	IntegrationConditionPlatformAvailableReason string = "IntegrationPlatformAvailable"
	// IntegrationConditionDeploymentAvailableReason --
	IntegrationConditionDeploymentAvailableReason string = "DeploymentAvailable"
	// IntegrationConditionDeploymentNotAvailableReason --
	IntegrationConditionDeploymentNotAvailableReason string = "DeploymentNotAvailable"
	// IntegrationConditionServiceAvailableReason --
	IntegrationConditionServiceAvailableReason string = "ServiceAvailable"
	// IntegrationConditionServiceNotAvailableReason --
	IntegrationConditionServiceNotAvailableReason string = "ServiceNotAvailable"
	// IntegrationConditionContainerNotAvailableReason --
	IntegrationConditionContainerNotAvailableReason string = "ContainerNotAvailable"
	// IntegrationConditionRouteAvailableReason --
	IntegrationConditionRouteAvailableReason string = "RouteAvailable"
	// IntegrationConditionRouteNotAvailableReason --
	IntegrationConditionRouteNotAvailableReason string = "RouteNotAvailable"
	// IntegrationConditionIngressAvailableReason --
	IntegrationConditionIngressAvailableReason string = "IngressAvailable"
	// IntegrationConditionIngressNotAvailableReason --
	IntegrationConditionIngressNotAvailableReason string = "IngressNotAvailable"
	// IntegrationConditionKnativeServiceAvailableReason --
	IntegrationConditionKnativeServiceAvailableReason string = "KnativeServiceAvailable"
	// IntegrationConditionKnativeServiceNotAvailableReason --
	IntegrationConditionKnativeServiceNotAvailableReason string = "KnativeServiceNotAvailable"
	// IntegrationConditionCronJobAvailableReason --
	IntegrationConditionCronJobAvailableReason string = "CronJobAvailableReason"
	// IntegrationConditionCronJobNotAvailableReason --
	IntegrationConditionCronJobNotAvailableReason string = "CronJobNotAvailableReason"
	// IntegrationConditionPrometheusAvailableReason --
	IntegrationConditionPrometheusAvailableReason string = "PrometheusAvailable"
	// IntegrationConditionJolokiaAvailableReason --
	IntegrationConditionJolokiaAvailableReason string = "JolokiaAvailable"
	// IntegrationConditionProbesAvailableReason --
	IntegrationConditionProbesAvailableReason string = "ProbesAvailable"

	// IntegrationConditionKnativeServiceReadyReason --
	IntegrationConditionKnativeServiceReadyReason string = "KnativeServiceReady"
	// IntegrationConditionDeploymentReadyReason --
	IntegrationConditionDeploymentReadyReason string = "DeploymentReady"
	// IntegrationConditionDeploymentProgressingReason --
	IntegrationConditionDeploymentProgressingReason string = "DeploymentProgressing"
	// IntegrationConditionCronJobCreatedReason --
	IntegrationConditionCronJobCreatedReason string = "CronJobCreated"
	// IntegrationConditionCronJobActiveReason --
	IntegrationConditionCronJobActiveReason string = "CronJobActive"
	// IntegrationConditionLastJobSucceededReason --
	IntegrationConditionLastJobSucceededReason string = "LastJobSucceeded"
	// IntegrationConditionLastJobFailedReason --
	IntegrationConditionLastJobFailedReason string = "LastJobFailed"
	// IntegrationConditionRuntimeNotReadyReason --
	IntegrationConditionRuntimeNotReadyReason string = "RuntimeNotReady"
	// IntegrationConditionErrorReason --
	IntegrationConditionErrorReason string = "Error"

	// IntegrationConditionUnsupportedLanguageReason --
	IntegrationConditionUnsupportedLanguageReason string = "UnsupportedLanguage"

	// IntegrationConditionKameletsAvailable --
	IntegrationConditionKameletsAvailable IntegrationConditionType = "KameletsAvailable"
	// IntegrationConditionKameletsAvailableReason --
	IntegrationConditionKameletsAvailableReason string = "KameletsAvailable"
	// IntegrationConditionKameletsNotAvailableReason --
	IntegrationConditionKameletsNotAvailableReason string = "KameletsNotAvailable"
)

// IntegrationCondition describes the state of a resource at a certain point.
type IntegrationCondition struct {
	// Type of integration condition.
	Type IntegrationConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// First time the condition status transitioned to True.
	FirstTruthyTime *metav1.Time `json:"firstTruthyTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human-readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// PodSpecTemplate represent a template used to deploy an Integration `Pod`
type PodSpecTemplate struct {
	// the specification
	Spec PodSpec `json:"spec,omitempty"`
}

// PodSpec defines a group of Kubernetes resources
type PodSpec struct {
	// Volumes
	Volumes []corev1.Volume `json:"volumes,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name" protobuf:"bytes,1,rep,name=volumes"`
	// InitContainers
	InitContainers []corev1.Container `json:"initContainers,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,20,rep,name=initContainers"`
	// Containers
	Containers []corev1.Container `json:"containers" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=containers"`
	// EphemeralContainers
	EphemeralContainers []corev1.EphemeralContainer `json:"ephemeralContainers,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,34,rep,name=ephemeralContainers"`
	// RestartPolicy
	RestartPolicy corev1.RestartPolicy `json:"restartPolicy,omitempty" protobuf:"bytes,3,opt,name=restartPolicy,casttype=RestartPolicy"`
	// TerminationGracePeriodSeconds
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty" protobuf:"varint,4,opt,name=terminationGracePeriodSeconds"`
	// ActiveDeadlineSeconds
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty" protobuf:"varint,5,opt,name=activeDeadlineSeconds"`
	// DNSPolicy
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty" protobuf:"bytes,6,opt,name=dnsPolicy,casttype=DNSPolicy"`
	// NodeSelector
	NodeSelector map[string]string `json:"nodeSelector,omitempty" protobuf:"bytes,7,rep,name=nodeSelector"`
	// TopologySpreadConstraints
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty" patchStrategy:"merge" patchMergeKey:"topologyKey" protobuf:"bytes,33,opt,name=topologySpreadConstraints"`
	// PodSecurityContext
	SecurityContext corev1.PodSecurityContext `json:"securityContext,omitempty" protobuf:"bytes,34,opt,name=securityContext"`
}

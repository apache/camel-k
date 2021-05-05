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

// IntegrationSpec defines the desired state of Integration
type IntegrationSpec struct {
	Replicas  *int32         `json:"replicas,omitempty"`
	Sources   []SourceSpec   `json:"sources,omitempty"`
	Flows     []Flow         `json:"flows,omitempty"`
	Resources []ResourceSpec `json:"resources,omitempty"`
	// Deprecated: use the IntegrationKit field
	Kit                string                  `json:"kit,omitempty"`
	IntegrationKit     *corev1.ObjectReference `json:"integrationKit,omitempty"`
	Dependencies       []string                `json:"dependencies,omitempty"`
	Profile            TraitProfile            `json:"profile,omitempty"`
	Traits             map[string]TraitSpec    `json:"traits,omitempty"`
	Configuration      []ConfigurationSpec     `json:"configuration,omitempty"`
	Repositories       []string                `json:"repositories,omitempty"`
	ServiceAccountName string                  `json:"serviceAccountName,omitempty"`
}

// IntegrationStatus defines the observed state of Integration
type IntegrationStatus struct {
	Phase        IntegrationPhase `json:"phase,omitempty"`
	Digest       string           `json:"digest,omitempty"`
	Image        string           `json:"image,omitempty"`
	Dependencies []string         `json:"dependencies,omitempty"`
	Profile      TraitProfile     `json:"profile,omitempty"`
	// Deprecated: use the IntegrationKit field
	Kit                string                  `json:"kit,omitempty"`
	IntegrationKit     *corev1.ObjectReference `json:"integrationKit,omitempty"`
	Platform           string                  `json:"platform,omitempty"`
	GeneratedSources   []SourceSpec            `json:"generatedSources,omitempty"`
	GeneratedResources []ResourceSpec          `json:"generatedResources,omitempty"`
	Failure            *Failure                `json:"failure,omitempty"`
	RuntimeVersion     string                  `json:"runtimeVersion,omitempty"`
	RuntimeProvider    RuntimeProvider         `json:"runtimeProvider,omitempty"`
	Configuration      []ConfigurationSpec     `json:"configuration,omitempty"`
	Conditions         []IntegrationCondition  `json:"conditions,omitempty"`
	Version            string                  `json:"version,omitempty"`
	Replicas           *int32                  `json:"replicas,omitempty"`
	Selector           string                  `json:"selector,omitempty"`
	Capabilities       []string                `json:"capabilities,omitempty"`
	// The timestamp representing the last time when this integration was initialized.
	InitializationTimestamp *metav1.Time `json:"lastInitTimestamp,omitempty"`
}

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

	Spec   IntegrationSpec   `json:"spec,omitempty"`
	Status IntegrationStatus `json:"status,omitempty"`
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
	// IntegrationPhaseResolvingKit --
	IntegrationPhaseResolvingKit IntegrationPhase = "Resolving Kit"
	// IntegrationPhaseWaitingForBindings e.g Service Bindings --
	IntegrationPhaseWaitingForBindings IntegrationPhase = "Waiting for Bindings"
	// IntegrationPhaseDeploying --
	IntegrationPhaseDeploying IntegrationPhase = "Deploying"
	// IntegrationPhaseRunning --
	IntegrationPhaseRunning IntegrationPhase = "Running"
	// IntegrationPhaseUpdating is a phase where the operator is not supposed to interact with the resource
	IntegrationPhaseUpdating IntegrationPhase = "Updating"
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
	// IntegrationConditionServiceBindingsCollectionReady --
	IntegrationConditionServiceBindingsCollectionReady IntegrationConditionType = "ServiceBindingsCollectionReady"
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
	// IntegrationConditionErrorReason --
	IntegrationConditionErrorReason string = "Error"
	// IntegrationConditionCronJobCreatedReason --
	IntegrationConditionCronJobCreatedReason string = "CronJobCreated"
	// IntegrationConditionReplicaSetReadyReason --
	IntegrationConditionReplicaSetReadyReason string = "ReplicaSetReady"
	// IntegrationConditionReplicaSetNotReadyReason --
	IntegrationConditionReplicaSetNotReadyReason string = "ReplicaSetNotReady"
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
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:onlyVerbs=get,list,watch
// +genclient:noStatus
// +kubebuilder:object:root=true

// ScaledObject is a specification for a ScaledObject resource
type ScaledObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ScaledObjectSpec `json:"spec"`
}

// ScaledObjectSpec is the spec for a ScaledObject resource
type ScaledObjectSpec struct {
	ScaleTargetRef *v1.ObjectReference `json:"scaleTargetRef"`
	// +optional
	PollingInterval *int32 `json:"pollingInterval,omitempty"`
	// +optional
	CooldownPeriod *int32 `json:"cooldownPeriod,omitempty"`
	// +optional
	IdleReplicaCount *int32 `json:"idleReplicaCount,omitempty"`
	// +optional
	MinReplicaCount *int32 `json:"minReplicaCount,omitempty"`
	// +optional
	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`

	Triggers []ScaleTriggers `json:"triggers"`
}

// ScaleTriggers reference the scaler that will be used
type ScaleTriggers struct {
	Type string `json:"type"`
	// +optional
	Name     string            `json:"name,omitempty"`
	Metadata map[string]string `json:"metadata"`
	// +optional
	AuthenticationRef *ScaledObjectAuthRef `json:"authenticationRef,omitempty"`
	// +optional
	FallbackReplicas *int32 `json:"fallback,omitempty"`
}

// ScaledObjectAuthRef points to the TriggerAuthentication or ClusterTriggerAuthentication object that
// is used to authenticate the scaler with the environment
type ScaledObjectAuthRef struct {
	Name string `json:"name"`
	// Kind of the resource being referred to. Defaults to TriggerAuthentication.
	// +optional
	Kind string `json:"kind,omitempty"`
}

// +kubebuilder:object:root=true

// ScaledObjectList contains a list of ScaledObject.
type ScaledObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScaledObject `json:"items"`
}

// +genclient
// +genclient:onlyVerbs=get,list,watch
// +genclient:noStatus
// +kubebuilder:object:root=true

// TriggerAuthentication defines how a trigger can authenticate
type TriggerAuthentication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TriggerAuthenticationSpec `json:"spec"`
}

// TriggerAuthenticationSpec defines the various ways to authenticate
type TriggerAuthenticationSpec struct {
	// +optional
	SecretTargetRef []AuthSecretTargetRef `json:"secretTargetRef,omitempty"`
}

// AuthSecretTargetRef is used to authenticate using a reference to a secret
type AuthSecretTargetRef struct {
	Parameter string `json:"parameter"`
	Name      string `json:"name"`
	Key       string `json:"key"`
}

// +kubebuilder:object:root=true

// TriggerAuthenticationList contains a list of TriggerAuthentication
type TriggerAuthenticationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []TriggerAuthentication `json:"items"`
}

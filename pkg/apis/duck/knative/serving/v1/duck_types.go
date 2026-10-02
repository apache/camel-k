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

	"github.com/apache/camel-k/v2/pkg/apis/duck/knative/apis"
)

// ServiceConditionReady is set when the service is configured and has available backends ready to receive traffic.
const ServiceConditionReady = "Ready"

// +kubebuilder:object:root=true

// Service is a partial schema of the Knative Serving Service resource, carrying
// the fields Camel K produces and consumes.
type Service struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceSpec   `json:"spec,omitempty"`
	Status ServiceStatus `json:"status,omitempty"`
}

// ServiceSpec represents the configuration for the Service object.
type ServiceSpec struct {
	// ConfigurationSpec holds the desired state of the Configuration.
	ConfigurationSpec `json:",inline"`
}

// ConfigurationSpec holds the desired state of the Configuration.
type ConfigurationSpec struct {
	// Template holds the latest specification for the Revision to be stamped out.
	Template RevisionTemplateSpec `json:"template"`
}

// RevisionTemplateSpec describes the data a revision should have when created from a template.
type RevisionTemplateSpec struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RevisionSpec `json:"spec,omitempty"`
}

// RevisionSpec holds the desired state of the Revision.
type RevisionSpec struct {
	corev1.PodSpec `json:",inline"`

	// TimeoutSeconds is the maximum duration in seconds that the request instance is allowed to respond to a request.
	// +optional
	TimeoutSeconds *int64 `json:"timeoutSeconds,omitempty"`
}

// ServiceStatus represents the status of the Service resource.
type ServiceStatus struct {
	apis.Status `json:",inline"`

	// RouteStatusFields represents the current Route.
	RouteStatusFields `json:",inline"`
}

// RouteStatusFields holds the fields of the Route status.
type RouteStatusFields struct {
	// URL holds the url that will distribute traffic over the provided traffic targets.
	// +optional
	URL *apis.URL `json:"url,omitempty"`

	// Address holds the information needed for a Route to be the target of an event.
	// +optional
	Address *apis.Addressable `json:"address,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceList contains a list of Service.
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Service `json:"items"`
}

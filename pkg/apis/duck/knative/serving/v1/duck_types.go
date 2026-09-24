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
const ServiceConditionReady = apis.ConditionReady

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

	// RouteSpec holds the desired state of the Route.
	RouteSpec `json:",inline"`
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

	// ContainerConcurrency specifies the maximum allowed in-flight (concurrent) requests per container of the Revision.
	// +optional
	ContainerConcurrency *int64 `json:"containerConcurrency,omitempty"`

	// TimeoutSeconds is the maximum duration in seconds that the request instance is allowed to respond to a request.
	// +optional
	TimeoutSeconds *int64 `json:"timeoutSeconds,omitempty"`
}

// RouteSpec holds the desired state of the Route.
type RouteSpec struct {
	// Traffic specifies how to distribute traffic over a collection of revisions and configurations.
	// +optional
	Traffic []TrafficTarget `json:"traffic,omitempty"`
}

// TrafficTarget holds a single entry of the routing table for a Route.
type TrafficTarget struct {
	// Tag is optionally used to expose a dedicated url for referencing this target exclusively.
	// +optional
	Tag string `json:"tag,omitempty"`

	// RevisionName of a specific revision to which to send this portion of traffic.
	// +optional
	RevisionName string `json:"revisionName,omitempty"`

	// ConfigurationName of a configuration to whose latest revision we will send this portion of traffic.
	// +optional
	ConfigurationName string `json:"configurationName,omitempty"`

	// LatestRevision indicates that the latest ready Revision of the Configuration should be used for this traffic target.
	// +optional
	LatestRevision *bool `json:"latestRevision,omitempty"`

	// Percent indicates the percent of traffic that is to be routed to this Revision or Configuration.
	// +optional
	Percent *int64 `json:"percent,omitempty"`

	// URL displays the URL for accessing named traffic targets. It is only present in the status.
	// +optional
	URL *apis.URL `json:"url,omitempty"`
}

// ServiceStatus represents the status of the Service resource.
type ServiceStatus struct {
	apis.Status `json:",inline"`

	// ConfigurationStatusFields represents the current Configuration.
	ConfigurationStatusFields `json:",inline"`

	// RouteStatusFields represents the current Route.
	RouteStatusFields `json:",inline"`
}

// ConfigurationStatusFields holds the fields of the Configuration status.
type ConfigurationStatusFields struct {
	// LatestReadyRevisionName holds the name of the latest Revision that has become ready.
	// +optional
	LatestReadyRevisionName string `json:"latestReadyRevisionName,omitempty"`

	// LatestCreatedRevisionName is the last revision that was created from this Configuration.
	// +optional
	LatestCreatedRevisionName string `json:"latestCreatedRevisionName,omitempty"`
}

// RouteStatusFields holds the fields of the Route status.
type RouteStatusFields struct {
	// URL holds the url that will distribute traffic over the provided traffic targets.
	// +optional
	URL *apis.URL `json:"url,omitempty"`

	// Address holds the information needed for a Route to be the target of an event.
	// +optional
	Address *apis.Addressable `json:"address,omitempty"`

	// Traffic holds the configured traffic distribution.
	// +optional
	Traffic []TrafficTarget `json:"traffic,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceList contains a list of Service.
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Service `json:"items"`
}

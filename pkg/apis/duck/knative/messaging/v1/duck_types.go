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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/v2/pkg/apis/duck/knative/apis"
)

// +kubebuilder:object:root=true

// Subscription is a partial schema of the Knative Eventing Subscription resource,
// routing the events received on a Channel to a Subscriber.
type Subscription struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   SubscriptionSpec   `json:"spec"`
	Status SubscriptionStatus `json:"status,omitempty"`
}

// SubscriptionSpec specifies the Channel for incoming events and the Subscriber processing them.
type SubscriptionSpec struct {
	// Channel is a reference to the channel that will be used to create the subscription.
	Channel apis.KReference `json:"channel"`

	// Subscriber is reference to the function for processing events.
	// +optional
	Subscriber *apis.Destination `json:"subscriber,omitempty"`

	// Reply specifies how to handle events returned from the Subscriber target.
	// +optional
	Reply *apis.Destination `json:"reply,omitempty"`
}

// SubscriptionStatus represents the current state of a Subscription.
type SubscriptionStatus struct {
	apis.Status `json:",inline"`

	// PhysicalSubscription is the fully resolved values that this Subscription represents.
	PhysicalSubscription SubscriptionStatusPhysicalSubscription `json:"physicalSubscription,omitempty"`
}

// SubscriptionStatusPhysicalSubscription represents the fully resolved values for this Subscription.
type SubscriptionStatusPhysicalSubscription struct {
	// SubscriberURI is the fully resolved URI for spec.subscriber.
	// +optional
	SubscriberURI *apis.URL `json:"subscriberUri,omitempty"`

	// ReplyURI is the fully resolved URI for the spec.reply.
	// +optional
	ReplyURI *apis.URL `json:"replyUri,omitempty"`
}

// +kubebuilder:object:root=true

// SubscriptionList contains a list of Subscription.
type SubscriptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Subscription `json:"items"`
}

// +kubebuilder:object:root=true

// Channel is a partial schema of the Knative Eventing Channel resource.
type Channel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChannelSpec   `json:"spec,omitempty"`
	Status ChannelStatus `json:"status,omitempty"`
}

// ChannelSpec holds the desired state of the Channel.
type ChannelSpec struct {
	// ChannelTemplate specifies which Channel CRD to use to create the backing Channel.
	// +optional
	ChannelTemplate *ChannelTemplateSpec `json:"channelTemplate,omitempty"`
}

// ChannelTemplateSpec identifies the underlying Channel implementation.
type ChannelTemplateSpec struct {
	metav1.TypeMeta `json:",inline"`
}

// ChannelStatus represents the current state of a Channel.
type ChannelStatus struct {
	apis.Status `json:",inline"`

	// AddressStatus is the part where the Channel fulfills the Addressable contract.
	apis.AddressStatus `json:",inline"`
}

// +kubebuilder:object:root=true

// ChannelList contains a list of Channel.
type ChannelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Channel `json:"items"`
}

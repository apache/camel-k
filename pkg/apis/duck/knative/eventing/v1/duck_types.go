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

// Trigger is a partial schema of the Knative Eventing Trigger resource: a request
// to have events delivered to a Subscriber from a Broker's event pool.
type Trigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TriggerSpec   `json:"spec,omitempty"`
	Status TriggerStatus `json:"status,omitempty"`
}

// TriggerSpec defines the desired state of the Trigger.
type TriggerSpec struct {
	// Broker is the broker that this trigger receives events from.
	Broker string `json:"broker,omitempty"`

	// Filter is the filter to apply against all events from the Broker.
	// +optional
	Filter *TriggerFilter `json:"filter,omitempty"`

	// Subscriber is the addressable that receives events from the Broker that pass the Filter.
	Subscriber apis.Destination `json:"subscriber"`
}

// TriggerFilter allows filtering events destined for a Trigger's Subscriber.
type TriggerFilter struct {
	// Attributes filters events by exact match on event context attributes.
	// +optional
	Attributes TriggerFilterAttributes `json:"attributes,omitempty"`
}

// TriggerFilterAttributes is a map of context attribute names to values for filtering by equality.
type TriggerFilterAttributes map[string]string

// TriggerStatus represents the current state of a Trigger.
type TriggerStatus struct {
	apis.Status `json:",inline"`
}

// +kubebuilder:object:root=true

// TriggerList contains a list of Trigger.
type TriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Trigger `json:"items"`
}

// +kubebuilder:object:root=true

// Broker is a partial schema of the Knative Eventing Broker resource: a pool of
// events that are consumable using Triggers.
type Broker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BrokerSpec   `json:"spec,omitempty"`
	Status BrokerStatus `json:"status,omitempty"`
}

// BrokerSpec defines the desired state of the Broker.
type BrokerSpec struct {
	// Config is a KReference to the configuration that specifies configuration options for this Broker.
	// +optional
	Config *apis.KReference `json:"config,omitempty"`
}

// BrokerStatus represents the current state of a Broker.
type BrokerStatus struct {
	apis.Status `json:",inline"`

	// AddressStatus is the part where the Broker fulfills the Addressable contract.
	apis.AddressStatus `json:",inline"`
}

// +kubebuilder:object:root=true

// BrokerList contains a list of Broker.
type BrokerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Broker `json:"items"`
}

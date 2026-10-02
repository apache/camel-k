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

	Spec TriggerSpec `json:"spec,omitempty"`
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
	Attributes map[string]string `json:"attributes,omitempty"`
}

// +kubebuilder:object:root=true

// TriggerList contains a list of Trigger.
type TriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Trigger `json:"items"`
}

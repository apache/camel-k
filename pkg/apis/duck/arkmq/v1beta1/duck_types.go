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

package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	ArkMQGroup       = "broker.amq.io"
	ArkMQVersion     = "v1beta1"
	ArkMQKindAddress = "ActiveMQArtemisAddress"
	ArkMQKindBroker  = "ActiveMQArtemis"

	ArkMQBrokerLabel = "ActiveMQArtemis"
)

// +genclient
// +genclient:onlyVerbs=get,list,watch
// +genclient:noStatus
// +kubebuilder:object:root=true

// ActiveMQArtemisAddress is the duck of an ActiveMQArtemisAddress.
type ActiveMQArtemisAddress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ActiveMQArtemisAddressSpec `json:"spec,omitempty"`
}

// ActiveMQArtemisAddressSpec is the duck of an ActiveMQArtemisAddress spec.
type ActiveMQArtemisAddressSpec struct {
	AddressName string `json:"addressName,omitempty"`
	QueueName   string `json:"queueName,omitempty"`
	RoutingType string `json:"routingType,omitempty"`
	ApplyTo     string `json:"applyTo,omitempty"`
}

// +kubebuilder:object:root=true

// ActiveMQArtemisAddressList contains a list of ActiveMQArtemisAddress.
type ActiveMQArtemisAddressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ActiveMQArtemisAddress `json:"items"`
}

// +genclient
// +genclient:onlyVerbs=get,list,watch
// +genclient:noStatus
// +kubebuilder:object:root=true

// ActiveMQArtemis is the duck of an ActiveMQArtemis broker.
type ActiveMQArtemis struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActiveMQArtemisSpec   `json:"spec,omitempty"`
	Status ActiveMQArtemisStatus `json:"status,omitempty"`
}

// ActiveMQArtemisSpec contains relevant spec of the ActiveMQArtemis.
type ActiveMQArtemisSpec struct {
	DeploymentPlan ActiveMQArtemisDeploymentPlan `json:"deploymentPlan,omitempty"`
}

// ActiveMQArtemisDeploymentPlan contains deployment plan.
type ActiveMQArtemisDeploymentPlan struct {
	Size int32 `json:"size,omitempty"`
}

// ActiveMQArtemisStatus contains the relevant info of the ActiveMQArtemis status.
type ActiveMQArtemisStatus struct {
	PortStatus []ActiveMQArtemisPortStatus `json:"portStatus,omitempty"`
}

// ActiveMQArtemisPortStatus contains port information.
type ActiveMQArtemisPortStatus struct {
	Name string `json:"name,omitempty"`
	Port int32  `json:"port,omitempty"`
}

// +kubebuilder:object:root=true

// ActiveMQArtemisList contains a list of ActiveMQArtemis.
type ActiveMQArtemisList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ActiveMQArtemis `json:"items"`
}

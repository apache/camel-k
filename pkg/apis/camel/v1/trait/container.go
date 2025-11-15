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

package trait

import corev1 "k8s.io/api/core/v1"

// The Container trait can be used to configure properties of the container where the integration will run.
//
// It also provides configuration for Services associated to the container.
//
// +camel-k:trait=container.
type ContainerTrait struct {
	PlatformBaseTrait `json:",inline" property:",squash"`

	// To automatically enable the trait
	Auto *bool `json:"auto,omitempty" property:"auto"`
	// The minimum amount of CPU required (default 125 millicores).
	RequestCPU string `json:"requestCPU,omitempty" property:"request-cpu"`
	// The minimum amount of memory required (default 128 Mi).
	RequestMemory string `json:"requestMemory,omitempty" property:"request-memory"`
	// The maximum amount of CPU to be provided (default 500 millicores).
	LimitCPU string `json:"limitCPU,omitempty" property:"limit-cpu"`
	// The maximum amount of memory to be provided (default 512 Mi).
	LimitMemory string `json:"limitMemory,omitempty" property:"limit-memory"`
	// List of container ports available in the container (syntax: <port-name>;<port-number>[;port-protocol]).
	// When omitted, `port-protocol` (admitted values `TCP`, `UDP` or `SCTP`) is `TCP`.
	// Don't use this for the primary http managed port (for which case you need to use `portName` and `port`).
	// Don't use in Knative based environments.
	Ports []string `json:"ports,omitempty" property:"ports"`
	// Can be used to enable/disable http exposure via kubernetes Service.
	Expose *bool `json:"expose,omitempty" property:"expose"`
	// To configure a different http port exposed by the container (default `8080`).
	Port int32 `json:"port,omitempty" property:"port"`
	// To configure a different http port name for the port exposed by the container.
	// It defaults to `http` only when the `expose` parameter is true.
	PortName string `json:"portName,omitempty" property:"port-name"`
	// To configure under which service port the http container port is to be exposed (default `80`).
	ServicePort int32 `json:"servicePort,omitempty" property:"service-port"`
	// To configure under which service port name the http container port is to be exposed (default `http`).
	ServicePortName string `json:"servicePortName,omitempty" property:"service-port-name"`
	// The main container name. It's named `integration` by default.
	Name string `json:"name,omitempty" property:"name"`
	// The main container image to use for the Integration. When using this parameter the operator will create a synthetic IntegrationKit which
	// won't be able to execute traits requiring CamelCatalog. If the container image you're using is coming from an IntegrationKit, use instead
	// Integration `.spec.integrationKit` parameter. If you're moving the Integration across environments, you will also need to create an "external" IntegrationKit.
	Image string `json:"image,omitempty" property:"image"`
	// The pull policy: Always|Never|IfNotPresent
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty" property:"image-pull-policy"`
	// Security Context RunAsUser configuration (default none): this value is automatically retrieved in Openshift clusters when not explicitly set.
	RunAsUser *int64 `json:"runAsUser,omitempty" property:"run-as-user"`
	// Security Context RunAsNonRoot configuration (default false).
	RunAsNonRoot *bool `json:"runAsNonRoot,omitempty" property:"run-as-non-root"`
	// Security Context SeccompProfileType configuration (default RuntimeDefault).
	// +kubebuilder:validation:Enum=Unconfined;RuntimeDefault
	SeccompProfileType corev1.SeccompProfileType `json:"seccompProfileType,omitempty" property:"seccomp-profile-type"`
	// Security Context AllowPrivilegeEscalation configuration (default false).
	AllowPrivilegeEscalation *bool `json:"allowPrivilegeEscalation,omitempty" property:"allow-privilege-escalation"`
	// Security Context Capabilities Drop configuration (default ALL).
	CapabilitiesDrop []corev1.Capability `json:"capabilitiesDrop,omitempty" property:"capabilities-drop"`
	// Security Context Capabilities Add configuration (default none).
	CapabilitiesAdd []corev1.Capability `json:"capabilitiesAdd,omitempty" property:"capabilities-add"`
}

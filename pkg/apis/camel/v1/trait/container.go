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
// +camel-k:trait=container
// nolint: tagliatelle
type ContainerTrait struct {
	Trait `property:",squash" json:",inline"`

	// To automatically enable the trait
	Auto *bool `property:"auto" json:"auto,omitempty"`

	// The minimum amount of CPU required.
	RequestCPU string `property:"request-cpu" json:"requestCPU,omitempty"`
	// The minimum amount of memory required.
	RequestMemory string `property:"request-memory" json:"requestMemory,omitempty"`
	// The maximum amount of CPU required.
	LimitCPU string `property:"limit-cpu" json:"limitCPU,omitempty"`
	// The maximum amount of memory required.
	LimitMemory string `property:"limit-memory" json:"limitMemory,omitempty"`

	// Can be used to enable/disable exposure via kubernetes Service.
	Expose *bool `property:"expose" json:"expose,omitempty"`
	// To configure a different port exposed by the container (default `8080`).
	Port int `property:"port" json:"port,omitempty"`
	// To configure a different port name for the port exposed by the container. It defaults to `http` only when the `expose` parameter is true.
	PortName string `property:"port-name" json:"portName,omitempty"`
	// To configure under which service port the container port is to be exposed (default `80`).
	ServicePort int `property:"service-port" json:"servicePort,omitempty"`
	// To configure under which service port name the container port is to be exposed (default `http`).
	ServicePortName string `property:"service-port-name" json:"servicePortName,omitempty"`
	// The main container name. It's named `integration` by default.
	Name string `property:"name" json:"name,omitempty"`
	// The main container image
	Image string `property:"image" json:"image,omitempty"`
	// The pull policy: Always|Never|IfNotPresent
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	ImagePullPolicy corev1.PullPolicy `property:"image-pull-policy" json:"imagePullPolicy,omitempty"`
}

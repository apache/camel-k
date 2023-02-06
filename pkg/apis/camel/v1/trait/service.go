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

// The Service trait exposes the integration with a Service resource so that it can be accessed by other applications
// (or integrations) in the same namespace.
//
// It's enabled by default if the integration depends on a Camel component that can expose a HTTP endpoint.
//
// +camel-k:trait=service.
type ServiceTrait struct {
	Trait `property:",squash" json:",inline"`
	// To automatically detect from the code if a Service needs to be created.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// Enable Service to be exposed as NodePort (default `false`).
	// Deprecated: Use service type instead.
	NodePort *bool `property:"node-port" json:"nodePort,omitempty"`
	// The type of service to be used, either 'ClusterIP', 'NodePort' or 'LoadBalancer'.
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	Type *ServiceType `property:"type" json:"type,omitempty"`
}

type ServiceType string

const (
	// ServiceTypeClusterIP means a service will only be accessible inside the
	// cluster, via the cluster IP.
	ServiceTypeClusterIP ServiceType = "ClusterIP"

	// ServiceTypeNodePort means a service will be exposed on one port of
	// every node, in addition to 'ClusterIP' type.
	ServiceTypeNodePort ServiceType = "NodePort"

	// ServiceTypeLoadBalancer means a service will be exposed via an
	// external load balancer (if the cloud provider supports it), in addition
	// to 'NodePort' type.
	ServiceTypeLoadBalancer ServiceType = "LoadBalancer"
)

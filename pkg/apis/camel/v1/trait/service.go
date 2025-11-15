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

// The Service trait exposes the Integration with a Service resource so that it can be accessed by other applications
// (or Integrations) in the same namespace.
//
// NOTE: this trait is automatically disabled if the Knative Service trait is enabled.
//
// It's enabled by default if the integration depends on a Camel component that can expose a HTTP endpoint.
//
// +camel-k:trait=service.
type ServiceTrait struct {
	Trait `json:",inline" property:",squash"`

	// To automatically detect from the code if a Service needs to be created.
	Auto *bool `json:"auto,omitempty" property:"auto"`
	// Enable Service to be exposed as NodePort (default `false`).
	// Deprecated: Use service type instead.
	NodePort *bool `json:"nodePort,omitempty" property:"node-port"`
	// The type of service to be used, either 'ClusterIP', 'NodePort' or 'LoadBalancer'.
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	Type *ServiceType `json:"type,omitempty" property:"type"`
	// The annotations added to the Service object.
	Annotations map[string]string `json:"annotations,omitempty" property:"annotations"`
	// The labels added to the Service object.
	Labels map[string]string `json:"labels,omitempty" property:"labels"`
	// List of container ports available in the container to expose
	// (syntax: <port-name>;<port-number>;<container-port-number>[;<port-protocol]).
	// When omitted, `port-protocol` (admitted values `TCP`, `UDP` or `SCTP`) is `TCP`.
	// Don't use this for the primary http managed port (which is managed by container trait).
	// Don't use in Knative based environments.
	Ports []string `json:"ports,omitempty" property:"ports"`
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

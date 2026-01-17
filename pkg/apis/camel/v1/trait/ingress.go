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

import networkingv1 "k8s.io/api/networking/v1"

// The Ingress trait can be used to expose the service associated with the integration
// to the outside world with a Kubernetes Ingress.
//
// It's enabled by default whenever a Service is added to the integration (through the `service` trait).
//
// +camel-k:trait=ingress.
//
//nolint:godoclint
type IngressTrait struct {
	Trait `json:",inline" property:",squash"`

	// The Ingress class name as defined by the Ingress spec
	// See https://kubernetes.io/docs/concepts/services-networking/ingress/
	IngressClassName string `json:"ingressClassName,omitempty" property:"ingress-class-name"`
	// The annotations added to the ingress.
	// This can be used to set controller specific annotations, e.g., when using the NGINX Ingress controller:
	// See https://github.com/kubernetes/ingress-nginx/blob/main/docs/user-guide/nginx-configuration/annotations.md
	Annotations map[string]string `json:"annotations,omitempty" property:"annotations"`
	// To configure the host exposed by the ingress.
	Host string `json:"host,omitempty" property:"host"`
	// To configure the path exposed by the ingress (default `/`).
	//
	// Deprecated: In favor of `paths` - left for backward compatibility.
	Path string `json:"path,omitempty" property:"path"`
	// To configure the paths exposed by the ingress (default `['/']`).
	Paths []string `json:"paths,omitempty" property:"paths"`
	// To configure the path type exposed by the ingress.
	// One of `Exact`, `Prefix`, `ImplementationSpecific` (default to `Prefix`).
	// +kubebuilder:validation:Enum=Exact;Prefix;ImplementationSpecific
	PathType *networkingv1.PathType `json:"pathType,omitempty" property:"path-type"`
	// To automatically add an ingress whenever the integration uses an HTTP endpoint consumer.
	Auto *bool `json:"auto,omitempty" property:"auto"`
	// To configure tls hosts
	TLSHosts []string `json:"tlsHosts,omitempty" property:"tls-hosts"`
	// To configure tls secret name
	TLSSecretName string `json:"tlsSecretName,omitempty" property:"tls-secret-name"`
}

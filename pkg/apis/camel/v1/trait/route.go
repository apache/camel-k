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

// WARNING: The Route trait is **deprecated** and will removed in future release versions: use Ingress trait istead.
//
// The Route trait can be used to configure the creation of OpenShift routes for the integration.
//
// The certificate and key contents may be sourced either from the local filesystem or in a OpenShift `secret` object.
// The user may use the parameters ending in `-secret` (example: `tls-certificate-secret`) to reference a certificate stored in a `secret`.
// Parameters ending in `-secret` have higher priorities and in case the same route parameter is set, for example: `tls-key-secret` and `tls-key`,
// then `tls-key-secret` is used.
// The recommended approach to set the key and certificates is to use `secrets` to store their contents and use the
// following parameters to reference them: `tls-certificate-secret`, `tls-key-secret`, `tls-ca-certificate-secret`, `tls-destination-ca-certificate-secret`
// See the examples section at the end of this page to see the setup options.
//
// +camel-k:trait=route.
// +camel-k:deprecated=2.6.0.
type RouteTrait struct {
	Trait `json:",inline" property:",squash"`

	// The annotations added to route.
	// This can be used to set route specific annotations
	// For annotations options see https://docs.openshift.com/container-platform/3.11/architecture/networking/routes.html#route-specific-annotations
	// CLI usage example: -t "route.annotations.'haproxy.router.openshift.io/balance'=true"
	Annotations map[string]string `json:"annotations,omitempty" property:"annotations"`
	// To configure the host exposed by the route.
	Host string `json:"host,omitempty" property:"host"`
	// The TLS termination type, like `edge`, `passthrough` or `reencrypt`.
	//
	// Refer to the OpenShift route documentation for additional information.
	// +kubebuilder:validation:Enum=edge;reencrypt;passthrough
	TLSTermination string `json:"tlsTermination,omitempty" property:"tls-termination"`
	// The TLS certificate contents.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSCertificate string `json:"tlsCertificate,omitempty" property:"tls-certificate"`
	// The secret name and key reference to the TLS certificate. The format is "secret-name[/key-name]", the value represents the secret name, if there is only one key in the secret it will be read, otherwise you can set a key name separated with a "/".
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSCertificateSecret string `json:"tlsCertificateSecret,omitempty" property:"tls-certificate-secret"`
	// The TLS certificate key contents.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSKey string `json:"tlsKey,omitempty" property:"tls-key"`
	// The secret name and key reference to the TLS certificate key. The format is "secret-name[/key-name]", the value represents the secret name, if there is only one key in the secret it will be read, otherwise you can set a key name separated with a "/".
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSKeySecret string `json:"tlsKeySecret,omitempty" property:"tls-key-secret"`
	// The TLS CA certificate contents.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSCACertificate string `json:"tlsCACertificate,omitempty" property:"tls-ca-certificate"`
	// The secret name and key reference to the TLS CA certificate. The format is "secret-name[/key-name]", the value represents the secret name, if there is only one key in the secret it will be read, otherwise you can set a key name separated with a "/".
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSCACertificateSecret string `json:"tlsCACertificateSecret,omitempty" property:"tls-ca-certificate-secret"`
	// The destination CA certificate provides the contents of the ca certificate of the final destination.  When using reencrypt
	// termination this file should be provided in order to have routers use it for health checks on the secure connection.
	// If this field is not specified, the router may provide its own destination CA and perform hostname validation using
	// the short service name (service.namespace.svc), which allows infrastructure generated certificates to automatically
	// verify.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSDestinationCACertificate string `json:"tlsDestinationCACertificate,omitempty" property:"tls-destination-ca-certificate"`
	// The secret name and key reference to the destination CA certificate. The format is "secret-name[/key-name]", the value represents the secret name, if there is only one key in the secret it will be read, otherwise you can set a key name separated with a "/".
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSDestinationCACertificateSecret string `json:"tlsDestinationCACertificateSecret,omitempty" property:"tls-destination-ca-certificate-secret"`
	// To configure how to deal with insecure traffic, e.g. `Allow`, `Disable` or `Redirect` traffic.
	//
	// Refer to the OpenShift route documentation for additional information.
	// +kubebuilder:validation:Enum=None;Allow;Redirect
	TLSInsecureEdgeTerminationPolicy string `json:"tlsInsecureEdgeTerminationPolicy,omitempty" property:"tls-insecure-edge-termination-policy"`
}

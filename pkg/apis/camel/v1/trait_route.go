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

// The Route trait can be used to configure the creation of OpenShift routes for the integration.
//
// The certificate and key contents may be sourced either from the local filesystem or in a Openshift `secret` object.
// The user may use the parameters ending in `-secret` (example: `tls-certificate-secret`) to reference a certificate stored in a `secret`.
// Parameters ending in `-secret` have higher priorities and in case the same route parameter is set, for example: `tls-key-secret` and `tls-key`,
// then `tls-key-secret` is used.
// The recommended approach to set the key and certificates is to use `secrets` to store their contents and use the
// following parameters to reference them: `tls-certificate-secret`, `tls-key-secret`, `tls-ca-certificate-secret`, `tls-destination-ca-certificate-secret`
// See the examples section at the end of this page to see the setup options.
//
// +camel-k:trait=route
// nolint: tagliatelle
type RouteTrait struct {
	Trait `property:",squash" json:",inline"`
	// To configure the host exposed by the route.
	Host string `property:"host" json:"host,omitempty"`
	// The TLS termination type, like `edge`, `passthrough` or `reencrypt`.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSTermination string `property:"tls-termination" json:"tlsTermination,omitempty"`
	// The TLS certificate contents.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSCertificate string `property:"tls-certificate" json:"tlsCertificate,omitempty"`
	// The secret name and key reference to the TLS certificate. The format is "secret-name[/key-name]", the value represents the secret name, if there is only one key in the secret it will be read, otherwise you can set a key name separated with a "/".
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSCertificateSecret string `property:"tls-certificate-secret" json:"tlsCertificateSecret,omitempty"`
	// The TLS certificate key contents.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSKey string `property:"tls-key" json:"tlsKey,omitempty"`
	// The secret name and key reference to the TLS certificate key. The format is "secret-name[/key-name]", the value represents the secret name, if there is only one key in the secret it will be read, otherwise you can set a key name separated with a "/".
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSKeySecret string `property:"tls-key-secret" json:"tlsKeySecret,omitempty"`
	// The TLS CA certificate contents.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSCACertificate string `property:"tls-ca-certificate" json:"tlsCACertificate,omitempty"`
	// The secret name and key reference to the TLS CA certificate. The format is "secret-name[/key-name]", the value represents the secret name, if there is only one key in the secret it will be read, otherwise you can set a key name separated with a "/".
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSCACertificateSecret string `property:"tls-ca-certificate-secret" json:"tlsCACertificateSecret,omitempty"`
	// The destination CA certificate provides the contents of the ca certificate of the final destination.  When using reencrypt
	// termination this file should be provided in order to have routers use it for health checks on the secure connection.
	// If this field is not specified, the router may provide its own destination CA and perform hostname validation using
	// the short service name (service.namespace.svc), which allows infrastructure generated certificates to automatically
	// verify.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSDestinationCACertificate string `property:"tls-destination-ca-certificate" json:"tlsDestinationCACertificate,omitempty"`
	// The secret name and key reference to the destination CA certificate. The format is "secret-name[/key-name]", the value represents the secret name, if there is only one key in the secret it will be read, otherwise you can set a key name separated with a "/".
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSDestinationCACertificateSecret string `property:"tls-destination-ca-certificate-secret" json:"tlsDestinationCACertificateSecret,omitempty"`
	// To configure how to deal with insecure traffic, e.g. `Allow`, `Disable` or `Redirect` traffic.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSInsecureEdgeTerminationPolicy string `property:"tls-insecure-edge-termination-policy" json:"tlsInsecureEdgeTerminationPolicy,omitempty"`
}

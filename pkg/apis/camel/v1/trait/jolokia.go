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

// The Jolokia trait activates and configures the Jolokia Java agent.
// This trait is useful to enable JMX access to Camel application.
// Make sure you have the right privileges to perform such an action on the cluster.
//
// See https://jolokia.org/reference/html/manual/agents.html
//
// WARNING: The Jolokia trait is **deprecated** and will removed in future release versions:
// use `jvm.agents` configuration instead.
//
// +camel-k:trait=jolokia.
// +camel-k:deprecated=2.8.0.
//
//nolint:godoclint
type JolokiaTrait struct {
	Trait `json:",inline" property:",squash"`

	// The PEM encoded CA certification file path, used to verify client certificates,
	// applicable when `protocol` is `https` and `use-ssl-client-authentication` is `true`
	// (default `/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt` for OpenShift).
	CaCert *string `json:"CACert,omitempty" property:"ca-cert"`
	// The principal(s) which must be given in a client certificate to allow access to the Jolokia endpoint,
	// applicable when `protocol` is `https` and `use-ssl-client-authentication` is `true`
	// (default `clientPrincipal=cn=system:master-proxy`, `cn=hawtio-online.hawtio.svc` and `cn=fuse-console.fuse.svc` for OpenShift).
	ClientPrincipal []string `json:"clientPrincipal,omitempty" property:"client-principal"`
	// Listen for multicast requests (default `false`)
	DiscoveryEnabled *bool `json:"discoveryEnabled,omitempty" property:"discovery-enabled"`
	// Mandate the client certificate contains a client flag in the extended key usage section,
	// applicable when `protocol` is `https` and `use-ssl-client-authentication` is `true`
	// (default `true` for OpenShift).
	ExtendedClientCheck *bool `json:"extendedClientCheck,omitempty" property:"extended-client-check"`
	// The Host address to which the Jolokia agent should bind to. If `"\*"` or `"0.0.0.0"` is given,
	// the servers binds to every network interface (default `"*"`).
	Host *string `json:"host,omitempty" property:"host"`
	// The password used for authentication, applicable when the `user` option is set.
	Password *string `json:"password,omitempty" property:"password"`
	// The Jolokia endpoint port (default `8778`).
	Port int32 `json:"port,omitempty" property:"port"`
	// The protocol to use, either `http` or `https` (default `https` for OpenShift)
	Protocol *string `json:"protocol,omitempty" property:"protocol"`
	// The user to be used for authentication
	User *string `json:"user,omitempty" property:"user"`
	// Whether client certificates should be used for authentication (default `true` for OpenShift).
	UseSslClientAuthentication *bool `json:"useSSLClientAuthentication,omitempty" property:"use-ssl-client-authentication"`
	// A list of additional Jolokia options as defined
	// in https://jolokia.org/reference/html/agents.html#agent-jvm-config[JVM agent configuration options]
	Options []string `json:"options,omitempty" property:"options"`
}

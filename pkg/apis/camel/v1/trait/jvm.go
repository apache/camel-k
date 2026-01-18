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

// CACertConfig specifies a CA certificate to import into the truststore.
type CACertConfig struct {
	// Path to the PEM-encoded CA certificate file to import.
	CertPath string `json:"certPath,omitempty" property:"cert-path"`
}

// BaseTruststore represents an existing truststore to use as the base for adding certificates.
type BaseTruststore struct {
	// Path to the base truststore file.
	TruststorePath string `json:"truststorePath,omitempty" property:"truststore-path"`
	// Path to a file containing the password for the base truststore.
	PasswordPath string `json:"passwordPath,omitempty" property:"password-path"`
}

// The JVM trait is used to configure the JVM that runs the Integration. This trait is configured only for Integration and related IntegrationKits
// (bound to a container image) built by Camel K operator. If the system detects the usage of a different container image (ie, built externally), then, the
// trait is disabled by the platform.
//
// +camel-k:trait=jvm.
type JVMTrait struct {
	Trait `json:",inline" property:",squash"`

	// Activates remote debugging, so that a debugger can be attached to the JVM, e.g., using port-forwarding
	Debug *bool `json:"debug,omitempty" property:"debug"`
	// Suspends the target JVM immediately before the main class is loaded
	DebugSuspend *bool `json:"debugSuspend,omitempty" property:"debug-suspend"`
	// Prints the command used the start the JVM in the container logs (default `true`)
	// Deprecated: no longer in use.
	PrintCommand *bool `json:"printCommand,omitempty" property:"print-command"`
	// Transport address at which to listen for the newly launched JVM (default `*:5005`)
	DebugAddress string `json:"debugAddress,omitempty" property:"debug-address"`
	// A list of JVM options
	Options []string `json:"options,omitempty" property:"options"`
	// Additional JVM classpath (use `Linux` classpath separator)
	Classpath string `json:"classpath,omitempty" property:"classpath"`
	// The Jar dependency which will run the application. Leave it empty for managed Integrations.
	Jar string `json:"jar,omitempty" property:"jar"`
	// A list of JVM agents to download and execute with format `<agent-name>;<agent-url>[;<jvm-agent-options>]`.
	Agents []string `json:"agents,omitempty" property:"agents"`
	// A list of CA certificates to import into the truststore. Certificates must be mounted via the mount trait.
	CACertificates []CACertConfig `json:"caCertificates,omitempty" property:"ca-certificates"`
	// Optional base truststore to use as the starting point for adding certificates.
	BaseTruststore *BaseTruststore `json:"baseTruststore,omitempty" property:"base-truststore"`
	// Path to a file containing the password for the generated truststore. Required when using ca-certificates without base-truststore.
	TruststorePasswordPath string `json:"truststorePasswordPath,omitempty" property:"truststore-password-path"`
	// The path where the generated truststore will be mounted (default `/etc/camel/conf.d/_truststore`).
	CACertMountPath string `json:"caCertMountPath,omitempty" property:"ca-cert-mount-path"`
	// Deprecated: Use CACertificates instead. Path to a PEM-encoded CA certificate file.
	CACert string `json:"caCert,omitempty" property:"ca-cert"`
	// Deprecated: Use CACertificates instead. Path to a file containing the truststore password.
	CACertPassword string `json:"caCertPassword,omitempty" property:"ca-cert-password"`
}

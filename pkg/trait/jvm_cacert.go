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

import (
	"k8s.io/utils/ptr"
)

const (
	defaultCACertMountPath    = "/etc/camel/conf.d/_truststore"
	caCertVolumeName          = "jvm-truststore"
	trustStoreName            = "truststore.jks"
	truststorePasswordEnvVar  = "TRUSTSTORE_PASSWORD"
	jdkCacertsPath            = "$JAVA_HOME/lib/security/cacerts"
	jdkCacertsDefaultPassword = "changeit"
)

// hasCACerts returns true if any CA certificates are configured (either via CACerts array or CACert).
func (t *jvmTrait) hasCACerts() bool {
	return len(t.CACerts) > 0 || t.CACert != ""
}

func (t *jvmTrait) getCACertMountPath() string {
	if t.CACertMountPath != "" {
		return t.CACertMountPath
	}

	return defaultCACertMountPath
}

func (t *jvmTrait) getTrustStorePath() string {
	return t.getCACertMountPath() + "/" + trustStoreName
}

// hasCustomPassword returns true if a custom password file path is provided.
func (t *jvmTrait) hasCustomPassword() bool {
	return t.CACertPassword != ""
}

// getEffectiveTruststorePassword returns the password for the truststore.
func (t *jvmTrait) getEffectiveTruststorePassword() string {
	if t.hasCustomPassword() {
		return "$(cat " + t.CACertPassword + ")"
	}

	return jdkCacertsDefaultPassword
}

// getAllCACertPaths returns all configured CA certificate paths.
// It merges the CACert (if set) with the CACerts array.
func (t *jvmTrait) getAllCACertPaths() []string {
	var paths []string

	paths = append(paths, t.CACerts...)

	if t.CACert != "" {
		found := false
		for _, p := range paths {
			if p == t.CACert {
				found = true

				break
			}
		}
		if !found {
			paths = append(paths, t.CACert)
		}
	}

	return paths
}

// getCACertPasswordPath returns the user-provided password file path.
func (t *jvmTrait) getCACertPasswordPath() string {
	return t.CACertPassword
}

// useSystemTruststore returns true if JDK's default cacerts should be used as base.
func (t *jvmTrait) useSystemTruststore() bool {
	return ptr.Deref(t.CACertUseSystemTruststore, false)
}

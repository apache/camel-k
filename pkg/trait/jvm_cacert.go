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

import "errors"

const (
	defaultCACertMountPath   = "/etc/camel/conf.d/_truststore"
	caCertVolumeName         = "jvm-truststore"
	trustStoreName           = "truststore.jks"
	truststorePasswordEnvVar = "TRUSTSTORE_PASSWORD"
)

func (t *jvmTrait) hasCACert() bool {
	return t.CACert != ""
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

// validateCACertConfig validates that the required file paths are provided.
func (t *jvmTrait) validateCACertConfig() error {
	if t.CACert == "" {
		return nil
	}
	if t.CACertPassword == "" {
		return errors.New("ca-cert-password is required when ca-cert is set")
	}

	return nil
}

// getCACertPath returns the user-provided CA certificate file path.
func (t *jvmTrait) getCACertPath() string {
	return t.CACert
}

// getCACertPasswordPath returns the user-provided password file path.
func (t *jvmTrait) getCACertPasswordPath() string {
	return t.CACertPassword
}

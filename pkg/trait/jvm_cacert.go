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
	"errors"
	"fmt"

	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
)

const (
	defaultCACertMountPath = "/etc/camel/conf.d/_truststore"
	caCertVolumeName       = "jvm-truststore"
	trustStoreName         = "truststore.jks"
)

type CACertEntry struct {
	CertPath string
}

// hasCACerts returns true if any CA certificates are configured.
func (t *jvmTrait) hasCACerts() bool {
	//nolint:staticcheck
	return len(t.CACertificates) > 0 || t.CACert != ""
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

// hasBaseTruststore returns true if a base truststore is configured.
func (t *jvmTrait) hasBaseTruststore() bool {
	return t.BaseTruststore != nil && t.BaseTruststore.TruststorePath != "" && t.BaseTruststore.PasswordPath != ""
}

func (t *jvmTrait) getTruststorePasswordPath() string {
	if t.TruststorePasswordPath != "" {
		return t.TruststorePasswordPath
	}
	if t.hasBaseTruststore() {
		return t.BaseTruststore.PasswordPath
	}
	//nolint:staticcheck
	if t.CACert != "" && t.CACertPassword != "" {
		return t.CACertPassword
	}

	return ""
}

// getBaseTruststore returns the base truststore configuration if set.
func (t *jvmTrait) getBaseTruststore() *traitv1.BaseTruststore {
	return t.BaseTruststore
}

func (t *jvmTrait) getAllCACertEntries() []CACertEntry {
	var entries []CACertEntry

	for _, cert := range t.CACertificates {
		if cert.CertPath != "" {
			entries = append(entries, CACertEntry{CertPath: cert.CertPath})
		}
	}

	//nolint:staticcheck
	if t.CACert != "" && t.CACertPassword != "" {
		found := false
		for _, e := range entries {
			//nolint:staticcheck
			if e.CertPath == t.CACert {
				found = true

				break
			}
		}
		if !found {
			//nolint:staticcheck
			entries = append(entries, CACertEntry{CertPath: t.CACert})
		}
	}

	return entries
}

func (t *jvmTrait) validateCACertConfig() error {
	for i, cert := range t.CACertificates {
		if cert.CertPath == "" {
			return fmt.Errorf("CACertificates[%d]: cert-path is required", i)
		}
	}

	//nolint:staticcheck
	if t.CACert != "" && t.CACertPassword == "" {
		return errors.New("ca-cert-password is required when ca-cert is specified")
	}

	if t.BaseTruststore != nil {
		hasTruststorePath := t.BaseTruststore.TruststorePath != ""
		hasPasswordPath := t.BaseTruststore.PasswordPath != ""
		if hasTruststorePath != hasPasswordPath {
			return errors.New("base-truststore: both truststore-path and password-path are required")
		}
	}

	// For new CACertificates field, require explicit truststore password or base truststore
	if len(t.CACertificates) > 0 && t.TruststorePasswordPath == "" && !t.hasBaseTruststore() {
		return errors.New("truststore-password-path is required when using ca-certificates without base-truststore")
	}

	return nil
}

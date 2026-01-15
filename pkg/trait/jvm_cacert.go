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

// CACertEntry represents a resolved CA certificate configuration.
type CACertEntry struct {
	CertPath     string
	PasswordPath string
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

// getBaseTruststore returns the base truststore configuration if set.
func (t *jvmTrait) getBaseTruststore() *traitv1.BaseTruststore {
	return t.BaseTruststore
}

// getAllCACertEntries returns all configured CA certificate entries.
func (t *jvmTrait) getAllCACertEntries() []CACertEntry {
	var entries []CACertEntry

	for _, cert := range t.CACertificates {
		if cert.CertPath != "" && cert.PasswordPath != "" {
			entries = append(entries, CACertEntry{
				CertPath:     cert.CertPath,
				PasswordPath: cert.PasswordPath,
			})
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
			entries = append(entries, CACertEntry{
				//nolint:staticcheck
				CertPath: t.CACert,
				//nolint:staticcheck
				PasswordPath: t.CACertPassword,
			})
		}
	}

	return entries
}

// validateCACertConfig validates the CA certificate configuration.
func (t *jvmTrait) validateCACertConfig() error {
	for i, cert := range t.CACertificates {
		if cert.CertPath != "" && cert.PasswordPath == "" {
			return fmt.Errorf("CACertificates[%d]: password path is required when certificate path is specified", i)
		}
		if cert.CertPath == "" && cert.PasswordPath != "" {
			return fmt.Errorf("CACertificates[%d]: certificate path is required when password path is specified", i)
		}
	}

	//nolint:staticcheck
	if t.CACert != "" && t.CACertPassword == "" {
		return errors.New("ca-cert-password is required when ca-cert is specified")
	}

	if t.BaseTruststore != nil {
		if t.BaseTruststore.TruststorePath != "" && t.BaseTruststore.PasswordPath == "" {
			return errors.New("base-truststore password path is required when truststore path is specified")
		}
		if t.BaseTruststore.TruststorePath == "" && t.BaseTruststore.PasswordPath != "" {
			return errors.New("base-truststore truststore path is required when password path is specified")
		}
	}

	return nil
}

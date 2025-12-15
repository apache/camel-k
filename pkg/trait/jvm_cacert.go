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
	"strings"
)

const (
	defaultCACertMountPath = "/etc/camel/conf.d/_truststore"
	caCertVolumeName       = "jvm-truststore"
	caCertSecretVolumeName = "ca-cert-secret" //nolint:gosec // G101: not a credential, just a volume name
	trustStoreName         = "truststore.jks"
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

// parseSecretRef parses a secret reference in the format "secret:name" or "secret:name/key".
func parseSecretRef(ref string) (string, string, error) {
	if !strings.HasPrefix(ref, "secret:") {
		return "", "", fmt.Errorf("invalid CA cert reference %q: must start with 'secret:'", ref)
	}

	ref = strings.TrimPrefix(ref, "secret:")
	parts := strings.SplitN(ref, "/", 2)
	secretName, secretKey := parts[0], ""

	if len(parts) > 1 {
		secretKey = parts[1]
	}
	if secretName == "" {
		return "", "", errors.New("invalid CA cert reference: secret name is empty")
	}

	return secretName, secretKey, nil
}

func (t *jvmTrait) getTrustStorePath() string {
	return t.getCACertMountPath() + "/" + trustStoreName
}

func getTrustStorePassword(integrationName string) string {
	return "camelk-" + integrationName
}

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

package kubernetes

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type dockerConfigJSON struct {
	Auths map[string]dockerConfigEntry `json:"auths"`
}

type dockerConfigEntry struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth"`
}

func buildDockerConfigJSON(server, username, password string) ([]byte, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	cfg := dockerConfigJSON{
		Auths: map[string]dockerConfigEntry{
			server: {
				Username: username,
				Password: password,
				Auth:     auth,
			},
		},
	}
	return json.Marshal(cfg)
}

// DockerRegistrySecret returns a kubernetes.io/dockerconfigjson secret.
func DockerRegistrySecret(ctx context.Context, namespace, name, server, username, password string) (*corev1.Secret, error) {
	dockerCfgJSON, err := buildDockerConfigJSON(server, username, password)
	if err != nil {
		return nil, fmt.Errorf("building docker config json: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerCfgJSON,
		},
	}

	return secret, nil
}

// TLSSecretFromFiles mirrors `kubectl create secret tls --cert=<path> --key=<path>`,
// reading the cert/key from disk and validating them the same way kubectl does
// (tls.X509KeyPair) before submitting to the API.
func TLSSecretFromFiles(ctx context.Context, namespace, name string, certPath, keyPath string) (*corev1.Secret, error) {
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("reading cert file %q: %w", certPath, err)
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading key file %q: %w", keyPath, err)
	}

	// Same validation kubectl performs: parses PEM blocks and confirms the
	// private key matches the certificate's public key.
	if _, err := tls.X509KeyPair(certData, keyData); err != nil {
		return nil, fmt.Errorf("failed to load key pair: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       certData,
			corev1.TLSPrivateKeyKey: keyData,
		},
	}

	return secret, nil
}

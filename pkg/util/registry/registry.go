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

package registry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apache/camel-k/v2/pkg/util/io"

	"github.com/apache/camel-k/v2/pkg/client"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var knownServersByRegistry = map[string]string{
	"docker.io": "https://index.docker.io/v1/,docker.io",
}

// Auth contains basic information for authenticating against a container registry.
type Auth struct {
	Server   string
	Username string
	Password string

	// additional information
	Registry string
}

type DockerConfigList struct {
	Auths map[string]DockerConfig `json:"auths,omitempty"`
}

type DockerConfig struct {
	Auth     string `json:"auth,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// IsSet returns if information has been set on the object.
func (a Auth) IsSet() bool {
	return a.Server != "" ||
		a.Username != "" ||
		a.Password != ""
}

// validate checks if all fields are populated correctly.
func (a Auth) validate() error {
	actualSevers := a.getActualServers()
	if len(actualSevers) < 1 || a.Username == "" {
		return errors.New("not enough information to generate a registry authentication file")
	}

	return nil
}

// GenerateDockerConfig generates a Docker compatible config.json file.
func (a Auth) GenerateDockerConfig() ([]byte, error) {
	if err := a.validate(); err != nil {
		return nil, err
	}
	content := a.generateDockerConfigObject()

	return json.Marshal(content)
}

func (a Auth) generateDockerConfigObject() DockerConfigList {
	dockerConfigs := make(map[string]DockerConfig)
	for _, server := range a.getActualServers() {
		dockerConfigs[server] = DockerConfig{Auth: a.encodedCredentials()}
	}
	return DockerConfigList{Auths: dockerConfigs}
}

func (a Auth) getActualServers() []string {
	if a.Server != "" {
		return []string{a.Server}
	}
	if p, ok := knownServersByRegistry[a.Registry]; ok {
		return strings.Split(p, ",")
	}

	if a.Registry != "" {
		return []string{a.Registry}
	}

	return nil
}

func (a Auth) encodedCredentials() string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", a.Username, a.Password)))
}

// MountSecretRegistryConfig write a file containing the secret registry config in a temporary folder.
func MountSecretRegistryConfig(ctx context.Context, c client.Client, namespace, prefix, name string) (string, error) {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", err
	}

	secret, err := c.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if removeErr := os.RemoveAll(dir); removeErr != nil {
			err = multierr.Append(err, removeErr)
		}
		return "", err
	}

	for file, content := range secret.Data {
		if err := os.WriteFile(filepath.Join(dir, remap(file)), content, io.FilePerm600); err != nil {
			if removeErr := os.RemoveAll(dir); removeErr != nil {
				err = multierr.Append(err, removeErr)
			}
			return "", err
		}
	}
	return dir, nil
}

func remap(name string) string {
	if name == ".dockerconfigjson" {
		return "config.json"
	}
	return name
}

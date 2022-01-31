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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	base64 "encoding/base64"

	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/registry"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"
)

// The Registry trait sets up Maven to use the Image registry
// as a Maven repository
//
// +camel-k:trait=registry
type registryTrait struct {
	BaseTrait `property:",squash"`
}

func newRegistryTrait() Trait {
	return &registryTrait{
		BaseTrait: NewBaseTrait("registry", 1650),
	}
}

// InfluencesKit overrides base class method
func (t *registryTrait) InfluencesKit() bool {
	return true
}

func (t *registryTrait) Configure(e *Environment) (bool, error) {
	// disabled by default
	if IsNilOrFalse(t.Enabled) {
		return false, nil
	}

	return e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted), nil
}

func (t *registryTrait) Apply(e *Environment) error {
	build := getBuilderTask(e.BuildTasks)
	ext := v1.MavenArtifact{
		GroupID:    "com.github.johnpoth",
		ArtifactID: "wagon-docker-registry",
		Version:    "0.2.0-SNAPSHOT",
	}
	policy := v1.RepositoryPolicy{
		Enabled:        true,
		ChecksumPolicy: "fail",
	}

	if e.Platform.Spec.Build.Registry.CA != "" {
		secret, err := decodeSecretKeySelector(e.Platform.Spec.Build.Registry.CA)
		if err != nil {
			return err
		}
		contains := false
		for _, ca := range build.Maven.CASecret {
			if ca.Name == secret.Name && ca.Key == secret.Key {
				contains = true
			}
		}
		if !contains {
			build.Maven.CASecret = append(build.Maven.CASecret, *secret)
		}
	}

	if e.Platform.Spec.Build.Registry.Secret != "" {
		secret, err := decodeSecretKeySelector(e.Platform.Spec.Build.Registry.Secret)
		if err != nil {
			return err
		}
		secretData, err := kubernetes.GetSecretRefData(e.Ctx, e.Client, e.Platform.Namespace, secret)
		if err != nil {
			return err
		}
		dockerAuth := registry.DockerConfigList{}
		err = json.Unmarshal(secretData, &dockerAuth)
		if err != nil {
			return err
		}
		config, ok := dockerAuth.Auths[e.Platform.Spec.Build.Registry.Address]
		if !ok {
			return nil
		}
		username := config.Username
		password := config.Password
		if username == "" && config.Auth != "" {
			decoded := base64.StdEncoding.EncodeToString([]byte(config.Auth))
			parts := strings.Split(decoded, ":")
			if len(parts) == 2 {
				username = strings.Split(decoded, ":")[0]
			}
		}
		if password == "" && config.Auth != "" {
			decoded := base64.StdEncoding.EncodeToString([]byte(config.Auth))
			parts := strings.Split(decoded, ":")
			if len(parts) == 2 {
				password = strings.Split(decoded, ":")[1]
			}
		}
		server := v1.Server{
			ID:       "image-registry",
			Username: username,
			Password: password,
		}
		build.Maven.Servers = append(build.Maven.Servers, server)
	}
	repo := v1.Repository{
		ID:        "image-registry",
		URL:       "docker://" + e.Platform.Spec.Build.Registry.Address,
		Snapshots: policy,
		Releases:  policy,
	}
	// configure Maven to lookup dependencies in the Image registry
	build.Maven.Repositories = append(build.Maven.Repositories, repo)
	build.Maven.Extension = append(build.Maven.Extension, ext)
	return nil
}

func decodeSecretKeySelector(secretKey string) (*corev1.SecretKeySelector, error) {
	r := regexp.MustCompile(`^([a-zA-Z0-9-]*)/([a-zA-Z0-9].*)$`)

	if !r.MatchString(secretKey) {
		return nil, fmt.Errorf("illegal Maven CA certificates secret key selector, syntax: secret-name/secret-key")
	}

	match := r.FindStringSubmatch(secretKey)

	return &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: match[1],
		},
		Key: match[2],
	}, nil
}

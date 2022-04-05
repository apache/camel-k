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
	"errors"
	"fmt"
	"regexp"
	"strings"

	base64 "encoding/base64"

	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/registry"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"
)

// The Registry trait sets up Maven to use the Image registry
// as a Maven repository.
//
// +camel-k:trait=registry.
type registryTrait struct {
	BaseTrait `property:",squash"`
}

func newRegistryTrait() Trait {
	return &registryTrait{
		BaseTrait: NewBaseTrait("registry", 1650),
	}
}

// InfluencesKit overrides base class method.
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
	registryAddress := e.Platform.Spec.Build.Registry.Address
	if registryAddress == "" && e.Platform.Status.Cluster == v1.IntegrationPlatformClusterOpenShift {
		registryAddress = "image-registry.openshift-image-registry.svc:5000"
	}
	if registryAddress == "" {
		return errors.New("could not figure out Image Registry URL, please set it manually")
	}
	build := getBuilderTask(e.BuildTasks)
	registryCa := e.Platform.Spec.Build.Registry.CA
	registrySecret := e.Platform.Spec.Build.Registry.Secret
	if e.Platform.Status.Cluster == v1.IntegrationPlatformClusterOpenShift {
		if registryCa == "" {
			ca, err := getOpenShiftImageRegistryCA(e)
			if err != nil {
				return err
			}
			registryCa = ca
		}
		if registrySecret == "" {
			secret, err := getOpenShiftRegistrySecret(e)
			if err != nil {
				return err
			}
			registrySecret = secret
		}
	}
	if registryCa != "" {
		err := addImageRegistryCaToMavenBuild(registryCa, build)
		if err != nil {
			return err
		}
	}
	if registrySecret != "" {
		server, err := extractMavenServerCredentialsFromSecret(registrySecret, e, registryAddress)
		if err != nil {
			return err
		}
		build.Maven.Servers = append(build.Maven.Servers, server)
	}
	addRegistryAndExtensionToMaven(registryAddress, build, e.Platform.Status.Cluster, e.Platform.Namespace)
	return nil
}

func addRegistryAndExtensionToMaven(registryAddress string, build *v1.BuilderTask, clusterType v1.IntegrationPlatformCluster, ns string) {
	if clusterType == v1.IntegrationPlatformClusterOpenShift {
		registryAddress = fmt.Sprintf("%s/%s", registryAddress, ns)
	}
	ext := v1.MavenArtifact{
		GroupID:    "com.github.johnpoth",
		ArtifactID: "wagon-docker-registry",
		Version:    "0.2.0",
	}
	policy := v1.RepositoryPolicy{
		Enabled:        true,
		ChecksumPolicy: "fail",
	}
	repo := v1.Repository{
		ID:        "image-registry",
		URL:       "docker://" + registryAddress,
		Snapshots: policy,
		Releases:  policy,
	}
	build.Maven.Repositories = append(build.Maven.Repositories, repo)
	build.Maven.Extension = append(build.Maven.Extension, ext)
}

func getOpenShiftRegistrySecret(e *Environment) (string, error) {
	// does not create it if it already exists
	err := platform.CreateBuilderServiceAccount(e.Ctx, e.Client, e.Platform)
	if err != nil {
		return "", err
	}
	sa := corev1.ServiceAccount{}
	key := ctrl.ObjectKey{
		Name:      platform.BuilderServiceAccount,
		Namespace: e.Platform.Namespace,
	}
	err = e.Client.Get(e.Ctx, key, &sa)
	if err != nil {
		return "", err
	}

	for _, secret := range sa.Secrets {
		if strings.Contains(secret.Name, "camel-k-builder-dockercfg") {
			return secret.Name, nil
		}
	}
	return "", nil
}

func extractMavenServerCredentialsFromSecret(registrySecret string, e *Environment, registryAddress string) (v1.Server, error) {
	secret, err := kubernetes.GetSecret(e.Ctx, e.Client, registrySecret, e.Platform.Namespace)
	if err != nil {
		return v1.Server{}, err
	}
	if secret.Type == corev1.SecretTypeBasicAuth {
		return v1.Server{
			ID:       "image-registry",
			Username: string(secret.Data[corev1.BasicAuthUsernameKey]),
			Password: string(secret.Data[corev1.BasicAuthPasswordKey]),
		}, nil
	}

	secretData, err := getDockerConfig(secret)
	if err != nil {
		return v1.Server{}, err
	}
	dockerAuth := registry.DockerConfigList{}
	err = json.Unmarshal(secretData, &dockerAuth)
	if err != nil {
		return v1.Server{}, err
	}
	config, ok := dockerAuth.Auths[registryAddress]
	if !ok {
		return v1.Server{}, fmt.Errorf("cannot resolve registry address %s in secret %s", registryAddress, registrySecret)
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
	return v1.Server{
		ID:       "image-registry",
		Username: username,
		Password: password,
	}, nil
}

func getDockerConfig(secret *corev1.Secret) ([]byte, error) {
	if secret.Type == corev1.SecretTypeDockerConfigJson {
		secretData, ok := secret.Data[corev1.DockerConfigJsonKey]
		if !ok {
			return nil, fmt.Errorf("cannot convert secret into Docker Config")
		}
		return secretData, nil
	}
	if secret.Type == corev1.SecretTypeDockercfg {
		secretData, ok := secret.Data[corev1.DockerConfigKey]
		if !ok {
			return nil, fmt.Errorf("cannot convert secret into Docker config")
		}
		secretData = []byte(fmt.Sprintf("{ \"auths\": %s}", secretData))
		return secretData, nil
	}
	return nil, fmt.Errorf("unsupported Secret type %s", secret.Type)
}

func getOpenShiftImageRegistryCA(e *Environment) (string, error) {
	// does not create it if it already exists
	err := platform.CreateBuilderServiceAccount(e.Ctx, e.Client, e.Platform)
	if err != nil {
		return "", err
	}
	sa := corev1.ServiceAccount{}
	key := ctrl.ObjectKey{
		Name:      platform.BuilderServiceAccount,
		Namespace: e.Platform.Namespace,
	}
	err = e.Client.Get(e.Ctx, key, &sa)
	if err != nil {
		return "", err
	}
	for _, secret := range sa.Secrets {
		if strings.Contains(secret.Name, "camel-k-builder-token") {
			return secret.Name + "/service-ca.crt", nil
		}
	}
	return "", errors.New("could not find OpenShift Image Registry CA")
}

func addImageRegistryCaToMavenBuild(registryCa string, build *v1.BuilderTask) error {
	secret, err := decodeSecretKeySelector(registryCa)
	if err != nil {
		return err
	}
	contains := false
	for _, ca := range build.Maven.CASecrets {
		if ca.Name == secret.Name && ca.Key == secret.Key {
			contains = true
		}
	}
	if !contains {
		build.Maven.CASecrets = append(build.Maven.CASecrets, *secret)
	}
	return nil
}

func decodeSecretKeySelector(secretKey string) (*corev1.SecretKeySelector, error) {
	r := regexp.MustCompile(`^([a-zA-Z0-9-]*)/([a-zA-Z0-9.].*)$`)

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

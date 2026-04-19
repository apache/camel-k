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

package platform

import (
	"strings"
	"testing"
	"time"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetEnvPlatform_Defaults(t *testing.T) {
	// No environment variables set
	pl := getEnvPlatform() // reinitialize to get the value from env vars

	assert.NotNil(t, pl)
	assert.Equal(t, DefaultBuildStrategy, pl.BuildConfiguration.Strategy)
	assert.Equal(t, DefaultBuildOrderStrategy, pl.BuildConfiguration.OrderStrategy)
	assert.Equal(t, DefaultBuildTimeout, pl.BuildTimeout)
	assert.Empty(t, pl.Registry.Address)
	assert.Equal(t, strings.Split(DefaultMavenCLIOptions, ","), pl.Maven.CLIOptions)
}

func TestGetEnvPlatform_WithEnv(t *testing.T) {
	t.Setenv("BUILD_RUNTIME_VERSION", "1.2.3")
	t.Setenv("BUILD_TIMEOUT_SECONDS", "10")
	t.Setenv("BUILD_STRATEGY", "pod")
	t.Setenv("BUILD_ORDER_STRATEGY", "fifo")
	t.Setenv("REGISTRY_ADDRESS", "registry.io")
	t.Setenv("REGISTRY_SECRET", "secret")
	t.Setenv("REGISTRY_INSECURE", "true")
	t.Setenv("BUILD_IMAGE_PLATFORMS", "linux/amd64,linux/arm64")
	t.Setenv("MAVEN_CLI_OPTIONS", "opt1,opt2")
	t.Setenv("MAVEN_CA_SECRETS", "secret1@key1,secret2@key2")
	t.Setenv("MAVEN_SETTINGS", "configmap:my-settings@settings")
	t.Setenv("MAVEN_SETTINGS_SECURITY", "secret:my-settings-sec@sec")

	p := getEnvPlatform() // reinitialize to get the value from env vars

	assert.Equal(t, "1.2.3", p.BuildRuntimeVersion)
	assert.Equal(t, time.Duration(10)*time.Second, p.BuildTimeout)
	assert.Equal(t, v1.BuildStrategy("pod"), p.BuildConfiguration.Strategy)
	assert.Equal(t, v1.BuildOrderStrategy("fifo"), p.BuildConfiguration.OrderStrategy)
	assert.Equal(t, "registry.io", p.Registry.Address)
	assert.Equal(t, "secret", p.Registry.Secret)
	assert.True(t, p.Registry.Insecure)
	assert.Equal(t, []string{"linux/amd64", "linux/arm64"}, p.BuildConfiguration.ImagePlatforms)
	assert.Equal(t, []string{"opt1", "opt2"}, p.Maven.CLIOptions)

	// Check CA secrets
	assert.Len(t, p.Maven.CASecrets, 2)
	assert.Equal(t, "secret1", p.Maven.CASecrets[0].Name)
	assert.Equal(t, "key1", p.Maven.CASecrets[0].Key)

	// Check value sources
	assert.NotNil(t, p.Maven.Settings.ConfigMapKeyRef)
	assert.Equal(t, "my-settings", p.Maven.Settings.ConfigMapKeyRef.Name)
	assert.Equal(t, "settings", p.Maven.Settings.ConfigMapKeyRef.Key)

	assert.NotNil(t, p.Maven.SettingsSecurity.SecretKeyRef)
	assert.Equal(t, "my-settings-sec", p.Maven.SettingsSecurity.SecretKeyRef.Name)
	assert.Equal(t, "sec", p.Maven.SettingsSecurity.SecretKeyRef.Key)
}

func TestValueSource_InvalidFormat(t *testing.T) {
	t.Setenv("MAVEN_SETTINGS", "invalidformat")

	vs, err := valueSource("MAVEN_SETTINGS")
	assert.Error(t, err)
	assert.Empty(t, vs.ConfigMapKeyRef)
	assert.Empty(t, vs.SecretKeyRef)
}

func TestCaSecrets_InvalidAndValid(t *testing.T) {
	t.Setenv("MAVEN_CA_SECRETS", "good@key,badformat,another@key2")

	secrets := caSecrets()
	assert.Len(t, secrets, 2)
	assert.Equal(t, "good", secrets[0].Name)
	assert.Equal(t, "key", secrets[0].Key)
	assert.Equal(t, "another", secrets[1].Name)
	assert.Equal(t, "key2", secrets[1].Key)
}

func TestRegistryParsing(t *testing.T) {
	t.Setenv("REGISTRY_ADDRESS", "reg.io")
	t.Setenv("REGISTRY_INSECURE", "true")

	r := registry()
	assert.Equal(t, "reg.io", r.Address)
	assert.True(t, r.Insecure)
}

func TestImagePlatforms_FromEnv(t *testing.T) {
	t.Setenv("BUILD_IMAGE_PLATFORMS", "linux/amd64,linux/arm64")

	ips := imagePlatforms()
	assert.Equal(t, []string{"linux/amd64", "linux/arm64"}, ips)
}

func TestImagePlatforms_Arm64Fallback(t *testing.T) {
	// Make sure the env var is empty
	t.Setenv("BUILD_IMAGE_PLATFORMS", "")

	// Override operatorArch to simulate arm64
	oldArch := operatorArch
	operatorArch = "arm64"
	defer func() { operatorArch = oldArch }()

	ips := imagePlatforms()
	assert.Equal(t, []string{"linux/arm64"}, ips)
}

func TestImagePlatforms_OtherArch(t *testing.T) {
	// Make sure the env var is empty
	t.Setenv("BUILD_IMAGE_PLATFORMS", "")

	// Simulate a non-arm64 arch
	oldArch := operatorArch
	operatorArch = "amd64"
	defer func() { operatorArch = oldArch }()

	ips := imagePlatforms()
	assert.Nil(t, ips)
}

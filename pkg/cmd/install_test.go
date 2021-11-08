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

package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/test"
)

const cmdInstall = "install"

// nolint: unparam
func initializeInstallCmdOptions(t *testing.T) (*installCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	options, rootCmd := kamelTestPreAddCommandInit()
	installCmdOptions := addTestInstallCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	return installCmdOptions, rootCmd, *options
}

func addTestInstallCmd(options RootCmdOptions, rootCmd *cobra.Command) *installCmdOptions {
	// Add a testing version of install Command
	installCmd, installOptions := newCmdInstall(&options)
	installCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	installCmd.PostRunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	installCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(installCmd)
	return installOptions
}

func TestInstallNoFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall)
	assert.Nil(t, err)
	// Check default expected values
	assert.Equal(t, false, installCmdOptions.Wait)
	assert.Equal(t, false, installCmdOptions.ClusterSetupOnly)
	assert.Equal(t, false, installCmdOptions.SkipOperatorSetup)
	assert.Equal(t, false, installCmdOptions.SkipClusterSetup)
	assert.Equal(t, false, installCmdOptions.ExampleSetup)
	assert.Equal(t, false, installCmdOptions.Global)
	assert.Equal(t, false, installCmdOptions.KanikoBuildCache)
	assert.Equal(t, false, installCmdOptions.Save)
	assert.Equal(t, false, installCmdOptions.Force)
	assert.Equal(t, true, installCmdOptions.Olm)
	assert.Equal(t, "", installCmdOptions.olmOptions.OperatorName)
	assert.Equal(t, "", installCmdOptions.olmOptions.Package)
	assert.Equal(t, "", installCmdOptions.olmOptions.Channel)
	assert.Equal(t, "", installCmdOptions.olmOptions.Source)
	assert.Equal(t, "", installCmdOptions.olmOptions.SourceNamespace)
	assert.Equal(t, "", installCmdOptions.olmOptions.GlobalNamespace)
	assert.Equal(t, int32(8081), installCmdOptions.HealthPort)
	assert.Equal(t, false, installCmdOptions.Monitoring)
	assert.Equal(t, int32(8080), installCmdOptions.MonitoringPort)
}

func TestInstallNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--nonExistingFlag")
	assert.NotNil(t, err)
}

func TestInstallBaseImageFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--base-image", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", installCmdOptions.BaseImage)
}

func TestInstallBuildPublishStrategyFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--build-publish-strategy", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", installCmdOptions.BuildPublishStrategy)
}

func TestInstallBuildStrategyFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--build-strategy", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", installCmdOptions.BuildStrategy)
}

func TestInstallBuildTimeoutFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--build-timeout", "10")
	assert.Nil(t, err)
	assert.Equal(t, "10", installCmdOptions.BuildTimeout)
}

func TestInstallClusterSetupFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--cluster-setup")
	assert.Nil(t, err)
	assert.Equal(t, true, installCmdOptions.ClusterSetupOnly)
}

func TestInstallClusterTypeFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--cluster-type", "Kubernetes")
	assert.Nil(t, err)
	assert.Equal(t, "Kubernetes", installCmdOptions.ClusterType)
}

func TestInstallExampleFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--example")
	assert.Nil(t, err)
	assert.Equal(t, true, installCmdOptions.ExampleSetup)
}

func TestInstallForceFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--force")
	assert.Nil(t, err)
	assert.Equal(t, true, installCmdOptions.Force)
}

func TestInstallGlobalFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--global")
	assert.Nil(t, err)
	assert.Equal(t, true, installCmdOptions.Global)
}

func TestInstallHealthFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--health-port", "7777")
	assert.Nil(t, err)
	assert.Equal(t, int32(7777), installCmdOptions.HealthPort)
}

func TestInstallHttpProxySecretFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--http-proxy-secret", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", installCmdOptions.HTTPProxySecret)
}

func TestInstallKanikoBuildCacheFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--kaniko-build-cache")
	assert.Nil(t, err)
	assert.Equal(t, true, installCmdOptions.KanikoBuildCache)
}

func TestInstallLocalRepositoryFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--maven-local-repository", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", installCmdOptions.MavenLocalRepository)
}

func TestInstallMavenRepositoryFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall,
		"--maven-repository", "someString1",
		"--maven-repository", "someString2")
	assert.Nil(t, err)
	assert.Equal(t, "someString1", installCmdOptions.MavenRepositories[0])
	assert.Equal(t, "someString2", installCmdOptions.MavenRepositories[1])
}

func TestInstallMavenSettingsFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--maven-settings", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", installCmdOptions.MavenSettings)
}

func TestInstallMonitoringFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall,
		"--monitoring",
		"--monitoring-port", "7777")
	assert.Nil(t, err)
	assert.Equal(t, true, installCmdOptions.Monitoring)
	assert.Equal(t, int32(7777), installCmdOptions.MonitoringPort)
}

func TestInstallOlmFalseFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--olm=false")
	assert.Nil(t, err)
	assert.Equal(t, false, installCmdOptions.Olm)
}

func TestInstallOlmTrueNonDefaultFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--olm",
		"--olm-channel", "olmChannel",
		"--olm-global-namespace", "olmGlobalNamespace",
		"--olm-operator-name", "olmOperatorName",
		"--olm-package", "olmPackage",
		"--olm-source", "olmSource",
		"--olm-source-namespace", "olmSourceNamespace",
		"--olm-starting-csv", "olmStartingCSV")
	assert.Nil(t, err)
	assert.Equal(t, true, installCmdOptions.Olm)
	assert.Equal(t, "olmChannel", installCmdOptions.olmOptions.Channel)
	assert.Equal(t, "olmGlobalNamespace", installCmdOptions.olmOptions.GlobalNamespace)
	assert.Equal(t, "olmOperatorName", installCmdOptions.olmOptions.OperatorName)
	assert.Equal(t, "olmPackage", installCmdOptions.olmOptions.Package)
	assert.Equal(t, "olmSource", installCmdOptions.olmOptions.Source)
	assert.Equal(t, "olmSourceNamespace", installCmdOptions.olmOptions.SourceNamespace)
	assert.Equal(t, "olmStartingCSV", installCmdOptions.olmOptions.StartingCSV)
}

func TestInstallOperatorImageFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--operator-image", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", installCmdOptions.OperatorImage)
}

func TestInstallOperatorImagePullPolicyFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--operator-image-pull-policy", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", installCmdOptions.OperatorImagePullPolicy)
}

func TestInstallOutputFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--output", "yaml")
	assert.Nil(t, err)
	assert.Equal(t, "yaml", installCmdOptions.OutputFormat)
}

func TestInstallPropertyFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall,
		"--maven-property", "someString1",
		"--maven-property", "someString2")
	assert.Nil(t, err)
	assert.Equal(t, "someString1", installCmdOptions.MavenProperties[0])
	assert.Equal(t, "someString2", installCmdOptions.MavenProperties[1])
}

func TestInstallRegistryFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall,
		"--registry", "registry",
		"--organization", "organization",
		"--registry-insecure",
		"--registry-secret", "secret")
	assert.Nil(t, err)
	assert.Equal(t, "registry", installCmdOptions.registry.Address)
	assert.Equal(t, "organization", installCmdOptions.registry.Organization)
	assert.Equal(t, true, installCmdOptions.registry.Insecure)
	assert.Equal(t, "secret", installCmdOptions.registry.Secret)
}

func TestInstallRegistryWithAuthFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall,
		"--registry-auth-password", "authPassword",
		"--registry-auth-server", "authServer",
		"--registry-auth-username", "authUsername")
	assert.Nil(t, err)
	assert.Equal(t, "authPassword", installCmdOptions.registryAuth.Password)
	assert.Equal(t, "authServer", installCmdOptions.registryAuth.Server)
	assert.Equal(t, "authUsername", installCmdOptions.registryAuth.Username)
}

func TestInstallRuntimeVersionFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--runtime-version", "1.3.0")
	assert.Nil(t, err)
	assert.Equal(t, "1.3.0", installCmdOptions.RuntimeVersion)
}

func TestInstallSaveFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--save")
	assert.Nil(t, err)
	assert.Equal(t, true, installCmdOptions.Save)
}

func TestInstallSkipClusterSetupFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--skip-cluster-setup")
	assert.Nil(t, err)
	assert.Equal(t, true, installCmdOptions.SkipClusterSetup)
}

func TestInstallSkipOperatorSetupFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--skip-operator-setup")
	assert.Nil(t, err)
	assert.Equal(t, true, installCmdOptions.SkipOperatorSetup)
}

func TestInstallSkipRegistrySetupFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--skip-registry-setup")
	assert.Nil(t, err)
	assert.Equal(t, true, installCmdOptions.SkipRegistrySetup)
}

func TestInstallTraitProfileFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--trait-profile", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", installCmdOptions.TraitProfile)
}

func TestInstallWaitFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--wait")
	assert.Nil(t, err)
	assert.Equal(t, true, installCmdOptions.Wait)
}

func TestDecodeMavenSettings(t *testing.T) {
	var err error
	var val v1.ValueSource

	// ConfigMap
	val, err = decodeMavenSettings("configmap:maven-settings/s.xml")
	assert.Nil(t, err)
	assert.Nil(t, val.SecretKeyRef)
	assert.Equal(t, "maven-settings", val.ConfigMapKeyRef.Name)
	assert.Equal(t, "s.xml", val.ConfigMapKeyRef.Key)

	val, err = decodeMavenSettings("configmap:maven-settings")
	assert.Nil(t, err)
	assert.Nil(t, val.SecretKeyRef)
	assert.Equal(t, "maven-settings", val.ConfigMapKeyRef.Name)
	assert.Empty(t, val.ConfigMapKeyRef.Key)

	// Secrets
	val, err = decodeMavenSettings("secret:maven-settings-secret/s.xml")
	assert.Nil(t, err)
	assert.Nil(t, val.ConfigMapKeyRef)
	assert.Equal(t, "maven-settings-secret", val.SecretKeyRef.Name)
	assert.Equal(t, "s.xml", val.SecretKeyRef.Key)

	val, err = decodeMavenSettings("secret:maven-settings-secret")
	assert.Nil(t, err)
	assert.Nil(t, val.ConfigMapKeyRef)
	assert.Equal(t, "maven-settings-secret", val.SecretKeyRef.Name)
	assert.Empty(t, val.SecretKeyRef.Key)

	// Errors
	_, err = decodeMavenSettings("something:maven-settings-secret/s.xml")
	assert.NotNil(t, err)
	_, err = decodeMavenSettings("secret")
	assert.NotNil(t, err)
}

func TestInstallTolerationFlag(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall,
		"--toleration", "key1=value1:NoSchedule",
		"--toleration", "key2=value2:NoExecute")
	assert.Nil(t, err)
	assert.Equal(t, "key1=value1:NoSchedule", installCmdOptions.Tolerations[0])
	assert.Equal(t, "key2=value2:NoExecute", installCmdOptions.Tolerations[1])
}

func TestInstallMavenExtension(t *testing.T) {
	installCmdOptions, rootCmd, _ := initializeInstallCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall,
		"--maven-extension", "fi.yle.tools:aws-maven:1.4.2")
	assert.Nil(t, err)
	assert.Equal(t, "fi.yle.tools:aws-maven:1.4.2", installCmdOptions.MavenExtensions[0])
}

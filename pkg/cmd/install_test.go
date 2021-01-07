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

	"github.com/apache/camel-k/pkg/util/olm"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/spf13/cobra"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"

	"github.com/stretchr/testify/assert"
)

const cmdInstall = "install"

func initializeRunCmdOptions(t *testing.T) (*installCmdOptions, *cobra.Command, RootCmdOptions) {
	options, rootCmd := kamelTestPreAddCommandInit()
	runCmdOptions := addTestInstallCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	return runCmdOptions, rootCmd, *options
}

func addTestInstallCmd(options RootCmdOptions, rootCmd *cobra.Command) *installCmdOptions {
	//add a testing version of install Command
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
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall)
	assert.Nil(t, err)
	//Check default expected values
	assert.Equal(t, false, runCmdOptions.Wait)
	assert.Equal(t, false, runCmdOptions.ClusterSetupOnly)
	assert.Equal(t, false, runCmdOptions.SkipOperatorSetup)
	assert.Equal(t, false, runCmdOptions.SkipClusterSetup)
	assert.Equal(t, false, runCmdOptions.ExampleSetup)
	assert.Equal(t, false, runCmdOptions.Global)
	assert.Equal(t, false, runCmdOptions.KanikoBuildCache)
	assert.Equal(t, false, runCmdOptions.Save)
	assert.Equal(t, false, runCmdOptions.Force)
	assert.Equal(t, true, runCmdOptions.Olm)
	assert.Equal(t, olm.DefaultOperatorName, runCmdOptions.olmOptions.OperatorName)
	assert.Equal(t, olm.DefaultPackage, runCmdOptions.olmOptions.Package)
	assert.Equal(t, olm.DefaultChannel, runCmdOptions.olmOptions.Channel)
	assert.Equal(t, olm.DefaultSource, runCmdOptions.olmOptions.Source)
	assert.Equal(t, olm.DefaultSourceNamespace, runCmdOptions.olmOptions.SourceNamespace)
	assert.Equal(t, olm.DefaultGlobalNamespace, runCmdOptions.olmOptions.GlobalNamespace)
	assert.Equal(t, int32(8081), runCmdOptions.HealthPort)
	assert.Equal(t, false, runCmdOptions.Monitoring)
	assert.Equal(t, int32(8080), runCmdOptions.MonitoringPort)
}

func TestInstallNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--nonExistingFlag")
	assert.NotNil(t, err)
}

func TestInstallBaseImageFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--base-image", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", runCmdOptions.BaseImage)
}

func TestInstallBuildPublishStrategyFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--build-publish-strategy", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", runCmdOptions.BuildPublishStrategy)
}

func TestInstallBuildStrategyFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--build-strategy", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", runCmdOptions.BuildStrategy)
}

func TestInstallBuildTimeoutFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--build-timeout", "10")
	assert.Nil(t, err)
	assert.Equal(t, "10", runCmdOptions.BuildTimeout)
}

func TestInstallClusterSetupFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--cluster-setup")
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.ClusterSetupOnly)
}

func TestInstallClusterTypeFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--cluster-type", "Kubernetes")
	assert.Nil(t, err)
	assert.Equal(t, "Kubernetes", runCmdOptions.ClusterType)
}

func TestInstallExampleFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--example")
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.ExampleSetup)
}

func TestInstallForceFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--force")
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.Force)
}

func TestInstallGlobalFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--global")
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.Global)
}

func TestInstallHealthFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--health-port", "7777")
	assert.Nil(t, err)
	assert.Equal(t, int32(7777), runCmdOptions.HealthPort)
}
func TestInstallHttpProxySecretFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--http-proxy-secret", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", runCmdOptions.HTTPProxySecret)
}

func TestInstallKanikoBuildCacheFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--kaniko-build-cache")
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.KanikoBuildCache)
}

func TestInstallLocalRepositoryFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--local-repository", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", runCmdOptions.LocalRepository)
}

func TestInstallMavenRepositoryFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall,
		"--maven-repository", "someString1",
		"--maven-repository", "someString2")
	assert.Nil(t, err)
	assert.Equal(t, "someString1", runCmdOptions.MavenRepositories[0])
	assert.Equal(t, "someString2", runCmdOptions.MavenRepositories[1])
}

func TestInstallMavenSettingsFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--maven-settings", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", runCmdOptions.MavenSettings)
}

func TestInstallMonitoringFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall,
		"--monitoring",
		"--monitoring-port", "7777")
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.Monitoring)
	assert.Equal(t, int32(7777), runCmdOptions.MonitoringPort)
}

func TestInstallOlmFalseFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--olm=false")
	assert.Nil(t, err)
	assert.Equal(t, false, runCmdOptions.Olm)
}

func TestInstallOlmTrueNonDefaultFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--olm",
		"--olm-channel", "olmChannel",
		"--olm-global-namespace", "olmGlobalNamespace",
		"--olm-operator-name", "olmOperatorName",
		"--olm-package", "olmPackage",
		"--olm-source", "olmSource",
		"--olm-source-namespace", "olmSourceNamespace",
		"--olm-starting-csv", "olmStartingCSV")
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.Olm)
	assert.Equal(t, "olmChannel", runCmdOptions.olmOptions.Channel)
	assert.Equal(t, "olmGlobalNamespace", runCmdOptions.olmOptions.GlobalNamespace)
	assert.Equal(t, "olmOperatorName", runCmdOptions.olmOptions.OperatorName)
	assert.Equal(t, "olmPackage", runCmdOptions.olmOptions.Package)
	assert.Equal(t, "olmSource", runCmdOptions.olmOptions.Source)
	assert.Equal(t, "olmSourceNamespace", runCmdOptions.olmOptions.SourceNamespace)
	assert.Equal(t, "olmStartingCSV", runCmdOptions.olmOptions.StartingCSV)
}

func TestInstallOperatorImageFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--operator-image", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", runCmdOptions.OperatorImage)
}

func TestInstallOperatorImagePullPolicyFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--operator-image-pull-policy", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", runCmdOptions.OperatorImagePullPolicy)
}

func TestInstallOutputFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--output", "yaml")
	assert.Nil(t, err)
	assert.Equal(t, "yaml", runCmdOptions.OutputFormat)
}

func TestInstallPropertyFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall,
		"--property", "someString1",
		"--property", "someString2")
	assert.Nil(t, err)
	assert.Equal(t, "someString1", runCmdOptions.Properties[0])
	assert.Equal(t, "someString2", runCmdOptions.Properties[1])
}

func TestInstallRegistryFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall,
		"--registry", "registry",
		"--organization", "organization",
		"--registry-insecure",
		"--registry-secret", "secret")
	assert.Nil(t, err)
	assert.Equal(t, "registry", runCmdOptions.registry.Address)
	assert.Equal(t, "organization", runCmdOptions.registry.Organization)
	assert.Equal(t, true, runCmdOptions.registry.Insecure)
	assert.Equal(t, "secret", runCmdOptions.registry.Secret)
}

func TestInstallRegistryWithAuthFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall,
		"--registry-auth-password", "authPassword",
		"--registry-auth-server", "authServer",
		"--registry-auth-username", "authUsername")
	assert.Nil(t, err)
	assert.Equal(t, "authPassword", runCmdOptions.registryAuth.Password)
	assert.Equal(t, "authServer", runCmdOptions.registryAuth.Server)
	assert.Equal(t, "authUsername", runCmdOptions.registryAuth.Username)
}

func TestInstallRuntimeVersionFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--runtime-version", "1.3.0")
	assert.Nil(t, err)
	assert.Equal(t, "1.3.0", runCmdOptions.RuntimeVersion)
}

func TestInstallSaveFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--save")
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.Save)
}

func TestInstallSkipClusterSetupFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--skip-cluster-setup")
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.SkipClusterSetup)
}

func TestInstallSkipOperatorSetupFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--skip-operator-setup")
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.SkipOperatorSetup)
}

func TestInstallTraitProfileFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--trait-profile", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", runCmdOptions.TraitProfile)
}

func TestInstallWaitFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdInstall, "--wait")
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.Wait)
}

func TestDecodeMavenSettings(t *testing.T) {
	var err error
	var val v1.ValueSource

	//
	// ConfigMap
	//
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

	//
	// Secrets
	//
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

	//
	// Errors
	//
	_, err = decodeMavenSettings("something:maven-settings-secret/s.xml")
	assert.NotNil(t, err)
	_, err = decodeMavenSettings("secret")
	assert.NotNil(t, err)
}

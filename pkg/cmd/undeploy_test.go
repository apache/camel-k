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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

const cmdUndeploy = "undeploy"

func initializeUndeployCmdOptions(t *testing.T, initObjs ...runtime.Object) (*cobra.Command, *undeployCmdOptions) {
	t.Helper()
	fakeClient, err := internal.NewFakeClient(initObjs...)
	require.NoError(t, err)
	options, rootCmd := kamelTestPreAddCommandInitWithClient(fakeClient)
	options.Namespace = "default"
	undeployCmdOptions := addTestUndeployCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd, options)

	return rootCmd, undeployCmdOptions
}

func addTestUndeployCmd(options RootCmdOptions, rootCmd *cobra.Command) *undeployCmdOptions {
	undeployCmd, undeployOptions := newCmdUndeploy(&options)
	undeployCmd.Args = ArbitraryArgs
	rootCmd.AddCommand(undeployCmd)
	return undeployOptions
}

func TestUndeployNonExistingFlag(t *testing.T) {
	cmd, _ := initializeUndeployCmdOptions(t)
	_, err := ExecuteCommand(cmd, cmdUndeploy, "--nonExistingFlag")
	require.Error(t, err)
	assert.Equal(t, "unknown flag: --nonExistingFlag", err.Error())
}

func TestUndeployNoArgs(t *testing.T) {
	cmd, _ := initializeUndeployCmdOptions(t)
	_, err := ExecuteCommand(cmd, cmdUndeploy)
	require.Error(t, err)
	assert.Equal(t, "undeploy requires an Integration name argument", err.Error())
}

func TestUndeployMissingIntegrations(t *testing.T) {
	cmd, _ := initializeUndeployCmdOptions(t)
	_, err := ExecuteCommand(cmd, cmdUndeploy, "missing")
	require.Error(t, err)
	assert.Equal(t,
		"could not find integration missing in namespace default: integrations.camel.apache.org \"missing\" not found",
		err.Error())
}

func TestUndeployNotRunningIntegrations(t *testing.T) {
	it := v1.NewIntegration("default", "my-it")
	it.Status.Phase = v1.IntegrationPhaseBuildRunning
	cmd, _ := initializeUndeployCmdOptions(t, &it)
	output, err := ExecuteCommand(cmd, cmdUndeploy, "my-it")
	require.NoError(t, err)
	assert.Contains(t, output, "could not undeploy integration my-it, it is not in status Running")
}

func TestUndeployMissingDontRunAnnotationIntegrations(t *testing.T) {
	it := v1.NewIntegration("default", "my-it")
	it.Status.Phase = v1.IntegrationPhaseRunning
	cmd, _ := initializeUndeployCmdOptions(t, &it)
	output, err := ExecuteCommand(cmd, cmdUndeploy, "my-it")
	require.NoError(t, err)
	assert.Contains(t, output, "could not undeploy integration my-it, it is not annotated with camel.apache.org/dont-run-after-build=true")
}

func TestUndeployIntegrations(t *testing.T) {
	it := v1.NewIntegration("default", "my-it")
	it.Status.Phase = v1.IntegrationPhaseRunning
	it.Annotations = map[string]string{
		v1.IntegrationDontRunAfterBuildAnnotation: "true",
	}
	cmd, _ := initializeUndeployCmdOptions(t, &it)
	output, err := ExecuteCommand(cmd, cmdUndeploy, "my-it")
	require.NoError(t, err)
	assert.Contains(t, output, "1 integrations have been undeployed")
}

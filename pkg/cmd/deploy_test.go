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

const cmdDeploy = "deploy"

func initializeDeployCmdOptions(t *testing.T, initObjs ...runtime.Object) (*cobra.Command, *deployCmdOptions) {
	t.Helper()
	fakeClient, err := internal.NewFakeClient(initObjs...)
	require.NoError(t, err)
	options, rootCmd := kamelTestPreAddCommandInitWithClient(fakeClient)
	options.Namespace = "default"
	deployCmdOptions := addTestDeployCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd, options)

	return rootCmd, deployCmdOptions
}

func addTestDeployCmd(options RootCmdOptions, rootCmd *cobra.Command) *deployCmdOptions {
	deployCmd, deployOptions := newCmdDeploy(&options)
	deployCmd.Args = ArbitraryArgs
	rootCmd.AddCommand(deployCmd)
	return deployOptions
}

func TestDeployNonExistingFlag(t *testing.T) {
	cmd, _ := initializeDeployCmdOptions(t)
	_, err := ExecuteCommand(cmd, cmdDeploy, "--nonExistingFlag")
	require.Error(t, err)
	assert.Equal(t, "unknown flag: --nonExistingFlag", err.Error())
}

func TestDeployMissingInput(t *testing.T) {
	cmd, _ := initializeDeployCmdOptions(t)
	_, err := ExecuteCommand(cmd, cmdDeploy)
	require.Error(t, err)
	assert.Equal(t, "deploy requires an Integration name argument", err.Error())
}

func TestDeployMissingIntegration(t *testing.T) {
	cmd, _ := initializeDeployCmdOptions(t)
	_, err := ExecuteCommand(cmd, cmdDeploy, "missing-it")
	require.Error(t, err)
	assert.Equal(t, "could not get Integration missing-it: integrations.camel.apache.org \"missing-it\" not found", err.Error())
}

func TestDeployCantDeployRunningIntegration(t *testing.T) {
	it := v1.NewIntegration("default", "my-it")
	it.Status.Phase = v1.IntegrationPhaseRunning
	cmd, _ := initializeDeployCmdOptions(t, &it)
	_, err := ExecuteCommand(cmd, cmdDeploy, "my-it")
	require.Error(t, err)
	assert.Equal(t, "could not run an Integration in Running status", err.Error())
}

func TestDeployIntegration(t *testing.T) {
	it := v1.NewIntegration("default", "my-it")
	it.Status.Phase = v1.IntegrationPhaseBuildComplete
	cmd, _ := initializeDeployCmdOptions(t, &it)
	output, err := ExecuteCommand(cmd, cmdDeploy, "my-it")
	require.NoError(t, err)
	assert.Contains(t, output, "Integration \"my-it\" deployed")
}

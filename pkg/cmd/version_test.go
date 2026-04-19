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
	"fmt"
	"testing"

	"github.com/apache/camel-k/v2/pkg/internal"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cmdVersion = "version"

func initializeVersionCmdOptions(t *testing.T, initObjs ...runtime.Object) (*versionCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	fakeClient, err := internal.NewFakeClient(initObjs...)
	require.NoError(t, err)
	options, rootCmd := kamelTestPreAddCommandInitWithClient(fakeClient)
	versionCmdOptions := addTestVersionCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd, options)

	return versionCmdOptions, rootCmd, *options
}

func addTestVersionCmd(options RootCmdOptions, rootCmd *cobra.Command) *versionCmdOptions {
	// add a testing version of version Command
	versionCmd, versionOptions := newCmdVersion(&options)
	versionCmd.Args = ArbitraryArgs
	rootCmd.AddCommand(versionCmd)
	return versionOptions
}

func TestVersionNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeVersionCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdVersion, "--nonExistingFlag")
	require.Error(t, err)
}

func TestVersionClient(t *testing.T) {
	_, rootCmd, _ := initializeVersionCmdOptions(t)
	output, err := ExecuteCommand(rootCmd, cmdVersion)
	require.NoError(t, err)
	assert.Contains(t, output, fmt.Sprintf("Camel K Client %s\n", defaults.Version))
}

func TestVersionOperatorFlag(t *testing.T) {
	versionCmdOptions, rootCmd, _ := initializeVersionCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdVersion, "--operator")
	require.NoError(t, err)
	assert.True(t, versionCmdOptions.Operator)
}

func TestVersionClientVerbose(t *testing.T) {
	versionCmdOptions, rootCmd, _ := initializeVersionCmdOptions(t)
	output, err := ExecuteCommand(rootCmd, cmdVersion, "-v")
	require.NoError(t, err)
	assert.True(t, versionCmdOptions.Verbose)
	assert.Contains(t, output, fmt.Sprintf("Camel K Client %s\nGit Commit: %s\n", defaults.Version, defaults.GitCommit))
}

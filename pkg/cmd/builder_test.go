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
	"github.com/stretchr/testify/require"
)

const cmdBuilder = "builder"

// nolint: unparam
func initializeBuilderCmdOptions(t *testing.T) (*builderCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	options, rootCmd := kamelTestPreAddCommandInit()
	builderCmdOptions := addTestBuilderCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd, options)

	return builderCmdOptions, rootCmd, *options
}

func addTestBuilderCmd(options RootCmdOptions, rootCmd *cobra.Command) *builderCmdOptions {
	// add a testing version of builder Command
	builderCmd, builderOptions := newCmdBuilder(&options)
	builderCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	builderCmd.PostRunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	builderCmd.Args = ArbitraryArgs
	rootCmd.AddCommand(builderCmd)
	return builderOptions
}

func TestBuilderNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeBuilderCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdBuilder, "--nonExistingFlag")
	require.Error(t, err)
}

func TestBuilderBuildNameFlag(t *testing.T) {
	builderCmdOptions, rootCmd, _ := initializeBuilderCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdBuilder, "--build-name", "someBuild")
	require.NoError(t, err)
	assert.Equal(t, "someBuild", builderCmdOptions.BuildName)
}

func TestBuilderTaskNameFlag(t *testing.T) {
	builderCmdOptions, rootCmd, _ := initializeBuilderCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdBuilder, "--task-name", "someTask")
	require.NoError(t, err)
	assert.Equal(t, "someTask", builderCmdOptions.TaskName)
}

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

	"github.com/apache/camel-k/pkg/util/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const cmdRebuild = "rebuild"

// nolint: unparam
func initializeRebuildCmdOptions(t *testing.T) (*rebuildCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	options, rootCmd := kamelTestPreAddCommandInit()
	rebuildCmdOptions := addTestRebuildCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	return rebuildCmdOptions, rootCmd, *options
}

func addTestRebuildCmd(options RootCmdOptions, rootCmd *cobra.Command) *rebuildCmdOptions {
	// add a testing version of rebuild Command
	rebuildCmd, rebuildOptions := newCmdRebuild(&options)
	rebuildCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	rebuildCmd.PostRunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	rebuildCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(rebuildCmd)
	return rebuildOptions
}

func TestRebuildNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeRebuildCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRebuild, "--nonExistingFlag")
	assert.NotNil(t, err)
}

func TestRebuildAllFlag(t *testing.T) {
	rebuildCmdOptions, rootCmd, _ := initializeRebuildCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRebuild, "--all")
	assert.Nil(t, err)
	assert.Equal(t, true, rebuildCmdOptions.RebuildAll)
}

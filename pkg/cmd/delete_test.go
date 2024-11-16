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

const cmdDelete = "delete"

// nolint: unparam
func initializeDeleteCmdOptions(t *testing.T) (*deleteCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	options, rootCmd := kamelTestPreAddCommandInit()
	deleteCmdOptions := addTestDeleteCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd, options)

	return deleteCmdOptions, rootCmd, *options
}

func addTestDeleteCmd(options RootCmdOptions, rootCmd *cobra.Command) *deleteCmdOptions {
	// add a testing version of delete Command
	deleteCmd, deleteOptions := newCmdDelete(&options)
	deleteCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	deleteCmd.PostRunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	deleteCmd.Args = ArbitraryArgs
	rootCmd.AddCommand(deleteCmd)
	return deleteOptions
}

func TestDeleteNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeDeleteCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdDelete, "--nonExistingFlag")
	require.Error(t, err)
}

func TestDeleteAllFlag(t *testing.T) {
	deleteCmdOptions, rootCmd, _ := initializeDeleteCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdDelete, "--all")
	require.NoError(t, err)
	assert.True(t, deleteCmdOptions.DeleteAll)
}

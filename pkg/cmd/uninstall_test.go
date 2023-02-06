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

	"github.com/stretchr/testify/assert"

	"github.com/apache/camel-k/pkg/util/test"
	"github.com/spf13/cobra"
)

func addTestUninstallCmd(options *RootCmdOptions, rootCmd *cobra.Command) *uninstallCmdOptions {
	// add a testing version of install Command
	uninstallCmd, installOptions := newCmdUninstall(options)
	uninstallCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	uninstallCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(uninstallCmd)
	return installOptions
}

func TestUninstallOlmFlags(t *testing.T) {
	options, cmd := kamelTestPreAddCommandInit()

	uninstallCmdOptions := addTestUninstallCmd(options, cmd)

	kamelTestPostAddCommandInit(t, cmd)

	_, err := test.ExecuteCommand(cmd, "uninstall", "--olm=false", "--olm-operator-name", "my-operator")
	assert.Nil(t, err)
	assert.False(t, uninstallCmdOptions.OlmEnabled)
	assert.Equal(t, "my-operator", uninstallCmdOptions.OlmOptions.OperatorName)
}

func TestUninstallSkipFlags(t *testing.T) {
	options, cmd := kamelTestPreAddCommandInit()

	uninstallCmdOptions := addTestUninstallCmd(options, cmd)

	kamelTestPostAddCommandInit(t, cmd)

	_, err := test.ExecuteCommand(cmd, "uninstall", "--skip-crd", "--skip-cluster-roles", "--skip-integration-platform")
	assert.Nil(t, err)
	assert.True(t, uninstallCmdOptions.SkipCrd)
	assert.True(t, uninstallCmdOptions.SkipClusterRoles)
	assert.True(t, uninstallCmdOptions.SkipIntegrationPlatform)
}

func TestUninstallAllFlag(t *testing.T) {
	options, cmd := kamelTestPreAddCommandInit()

	uninstallCmdOptions := addTestUninstallCmd(options, cmd)

	kamelTestPostAddCommandInit(t, cmd)

	_, err := test.ExecuteCommand(cmd, "uninstall", "--all")
	assert.Nil(t, err)
	assert.True(t, uninstallCmdOptions.SkipCrd)
	assert.True(t, uninstallCmdOptions.SkipClusterRoles)
	assert.False(t, uninstallCmdOptions.SkipIntegrationPlatform)
}

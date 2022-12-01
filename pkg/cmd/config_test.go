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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apache/camel-k/pkg/util/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const cmdConfig = "config"

// nolint: unparam
func initializeConfigCmdOptions(t *testing.T, mock bool) (*configCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	options, rootCmd := kamelTestPreAddCommandInit()
	configCmdOptions := addTestConfigCmd(*options, rootCmd, mock)
	kamelTestPostAddCommandInit(t, rootCmd)

	return configCmdOptions, rootCmd, *options
}

func addTestConfigCmd(options RootCmdOptions, rootCmd *cobra.Command, mock bool) *configCmdOptions {
	// add a testing version of config Command
	configCmd, configOptions := newCmdConfig(&options)
	if mock {
		configCmd.RunE = func(c *cobra.Command, args []string) error {
			return nil
		}
	}
	configCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(configCmd)
	return configOptions
}

func TestConfigNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeConfigCmdOptions(t, true)
	_, err := test.ExecuteCommand(rootCmd, cmdConfig, "--nonExistingFlag")
	assert.NotNil(t, err)
}

func TestConfigDefaultNamespaceFlag(t *testing.T) {
	configCmdOptions, rootCmd, _ := initializeConfigCmdOptions(t, true)
	_, err := test.ExecuteCommand(rootCmd, cmdConfig, "--default-namespace", "foo")
	assert.Nil(t, err)
	assert.Equal(t, "foo", configCmdOptions.DefaultNamespace)
}

func TestConfigListFlag(t *testing.T) {
	_, rootCmd, _ := initializeConfigCmdOptions(t, false)
	output, err := test.ExecuteCommand(rootCmd, cmdConfig, "--list")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(output, "No settings"), "The output is unexpected: "+output)
}

func TestConfigFolderFlagToUsed(t *testing.T) {
	_, rootCmd, _ := initializeConfigCmdOptions(t, false)
	output, err := test.ExecuteCommand(rootCmd, cmdConfig, "--list", "--folder", "used")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(output, fmt.Sprintf(" %s", DefaultConfigLocation)), "The output is unexpected: "+output)
}

func TestConfigFolderFlagToSub(t *testing.T) {
	_, rootCmd, _ := initializeConfigCmdOptions(t, false)
	output, err := test.ExecuteCommand(rootCmd, cmdConfig, "--list", "--folder", "sub")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(output, filepath.FromSlash(fmt.Sprintf(".kamel/%s", DefaultConfigLocation))), "The output is unexpected: "+output)
}

func TestConfigFolderFlagToHome(t *testing.T) {
	_, rootCmd, _ := initializeConfigCmdOptions(t, false)
	output, err := test.ExecuteCommand(rootCmd, cmdConfig, "--list", "--folder", "home")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(output, filepath.FromSlash(fmt.Sprintf(".kamel/%s", DefaultConfigLocation))), "The output is unexpected: "+output)
}

func TestConfigFolderFlagToEnv(t *testing.T) {
	os.Setenv("KAMEL_CONFIG_PATH", "/foo/bar")
	t.Cleanup(func() { os.Unsetenv("KAMEL_CONFIG_PATH") })
	_, rootCmd, _ := initializeConfigCmdOptions(t, false)
	output, err := test.ExecuteCommand(rootCmd, cmdConfig, "--list", "--folder", "env")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(output, filepath.FromSlash(fmt.Sprintf("foo/bar/%s", DefaultConfigLocation))), "The output is unexpected: "+output)
}

func TestConfigFolderFlagToEnvWithConfigName(t *testing.T) {
	os.Setenv("KAMEL_CONFIG_NAME", "config")
	os.Setenv("KAMEL_CONFIG_PATH", "/foo/bar")
	t.Cleanup(func() {
		os.Unsetenv("KAMEL_CONFIG_NAME")
		os.Unsetenv("KAMEL_CONFIG_PATH")
	})
	_, rootCmd, _ := initializeConfigCmdOptions(t, false)
	output, err := test.ExecuteCommand(rootCmd, cmdConfig, "--list", "--folder", "env")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(output, filepath.FromSlash("/foo/bar/config.yaml")), "The output is unexpected: "+output)
}

func TestConfigDefaultNamespace(t *testing.T) {
	_, err := os.Stat(DefaultConfigLocation)
	assert.True(t, os.IsNotExist(err), "No file at "+DefaultConfigLocation+" was expected")
	_, rootCmd, _ := initializeConfigCmdOptions(t, false)
	t.Cleanup(func() { os.Remove(DefaultConfigLocation) })
	_, err = test.ExecuteCommand(rootCmd, cmdConfig, "--default-namespace", "foo")
	assert.Nil(t, err)
	_, err = os.Stat(DefaultConfigLocation)
	assert.Nil(t, err, "A file at "+DefaultConfigLocation+" was expected")
	output, err := test.ExecuteCommand(rootCmd, cmdConfig, "--list")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(output, "foo"), "The output is unexpected: "+output)
	_, rootCmd, _ = initializeInstallCmdOptions(t)
	_, err = test.ExecuteCommand(rootCmd, cmdInstall)
	assert.Nil(t, err)
	// Check default namespace is set
	assert.Equal(t, "foo", rootCmd.Flag("namespace").Value.String())
}

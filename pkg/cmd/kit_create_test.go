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

const subCmdKit = "create"

// nolint: unparam
func initializeKitCreateCmdOptions(t *testing.T) (*kitCreateCommandOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	options, rootCmd := kamelTestPreAddCommandInit()
	kitCreateCmdOptions := addTestKitCreateCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	return kitCreateCmdOptions, rootCmd, *options
}

func addTestKitCreateCmd(options RootCmdOptions, rootCmd *cobra.Command) *kitCreateCommandOptions {
	// add a testing version of kit create Command
	kitCreateCmd, kitCreateOptions := newKitCreateCmd(&options)
	kitCreateCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	kitCreateCmd.PostRunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	kitCreateCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(kitCreateCmd)
	return kitCreateOptions
}

func TestKitCreateNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeKitCreateCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, subCmdKit, "--nonExistingFlag")
	assert.NotNil(t, err)
}

func TestKitCreateConfigMapFlag(t *testing.T) {
	kitCreateCmdOptions, rootCmd, _ := initializeKitCreateCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, subCmdKit,
		"--configmap", "someString1",
		"--configmap", "someString2")
	assert.Nil(t, err)
	assert.Len(t, kitCreateCmdOptions.Configmaps, 2)
	assert.Equal(t, "someString1", kitCreateCmdOptions.Configmaps[0])
	assert.Equal(t, "someString2", kitCreateCmdOptions.Configmaps[1])
}

func TestKitCreateDependencyFlag(t *testing.T) {
	kitCreateCmdOptions, rootCmd, _ := initializeKitCreateCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, subCmdKit,
		"--dependency", "someString1",
		"--dependency", "someString2")
	assert.Nil(t, err)
	assert.Len(t, kitCreateCmdOptions.Dependencies, 2)
	assert.Equal(t, "someString1", kitCreateCmdOptions.Dependencies[0])
	assert.Equal(t, "someString2", kitCreateCmdOptions.Dependencies[1])
}

func TestKitCreateImageFlag(t *testing.T) {
	kitCreateCmdOptions, rootCmd, _ := initializeKitCreateCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, subCmdKit, "--image", "someString")
	assert.Nil(t, err)
	assert.Equal(t, "someString", kitCreateCmdOptions.Image)
}

func TestKitCreatePropertyFlag(t *testing.T) {
	kitCreateCmdOptions, rootCmd, _ := initializeKitCreateCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, subCmdKit,
		"--property", "someString1",
		"--property", "someString2")
	assert.Nil(t, err)
	assert.Len(t, kitCreateCmdOptions.Properties, 2)
	assert.Equal(t, "someString1", kitCreateCmdOptions.Properties[0])
	assert.Equal(t, "someString2", kitCreateCmdOptions.Properties[1])
}

func TestKitCreateRepositoryFlag(t *testing.T) {
	kitCreateCmdOptions, rootCmd, _ := initializeKitCreateCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, subCmdKit,
		"--repository", "someString1",
		"--repository", "someString2")
	assert.Nil(t, err)
	assert.Len(t, kitCreateCmdOptions.Repositories, 2)
	assert.Equal(t, "someString1", kitCreateCmdOptions.Repositories[0])
	assert.Equal(t, "someString2", kitCreateCmdOptions.Repositories[1])
}

func TestKitCreateSecretFlag(t *testing.T) {
	kitCreateCmdOptions, rootCmd, _ := initializeKitCreateCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, subCmdKit,
		"--secret", "someString1",
		"--secret", "someString2")
	assert.Nil(t, err)
	assert.Len(t, kitCreateCmdOptions.Secrets, 2)
	assert.Equal(t, "someString1", kitCreateCmdOptions.Secrets[0])
	assert.Equal(t, "someString2", kitCreateCmdOptions.Secrets[1])
}

func TestKitCreateTraitFlag(t *testing.T) {
	kitCreateCmdOptions, rootCmd, _ := initializeKitCreateCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, subCmdKit,
		"--trait", "someString1",
		"--trait", "someString2")
	assert.Nil(t, err)
	assert.Len(t, kitCreateCmdOptions.Traits, 2)
	assert.Equal(t, "someString1", kitCreateCmdOptions.Traits[0])
	assert.Equal(t, "someString2", kitCreateCmdOptions.Traits[1])
}

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

const cmdVersion = "version"

func initializeVersionCmdOptions(t *testing.T) (*versionCmdOptions, *cobra.Command, RootCmdOptions) {
	options, rootCmd := kamelTestPreAddCommandInit()
	versionCmdOptions := addTestVersionCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	return versionCmdOptions, rootCmd, *options
}

func addTestVersionCmd(options RootCmdOptions, rootCmd *cobra.Command) *versionCmdOptions {
	//add a testing version of version Command
	versionCmd, versionOptions := newCmdVersion(&options)
	versionCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	versionCmd.PostRunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	versionCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(versionCmd)
	return versionOptions
}

func TestVersionNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeVersionCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdVersion, "--nonExistingFlag")
	assert.NotNil(t, err)
}

func TestVersionOperatorFlag(t *testing.T) {
	versionCmdOptions, rootCmd, _ := initializeVersionCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdVersion, "--operator")
	assert.Nil(t, err)
	assert.Equal(t, true, versionCmdOptions.Operator)
}

func TestCompatibleVersions(t *testing.T) {
	assert.Equal(t, true, compatibleVersions("1.3.0", "1.3.0"))
	assert.Equal(t, true, compatibleVersions("1.3.0", "1.3.1"))
	assert.Equal(t, true, compatibleVersions("1.3.0", "1.3.0-SNAPSHOT"))
	assert.Equal(t, false, compatibleVersions("1.3.0", "1.2.0"))
	assert.Equal(t, false, compatibleVersions("1.3.0", "2.3.0"))
	assert.Equal(t, false, compatibleVersions("1.3.0", "dsadsa"))
	assert.Equal(t, false, compatibleVersions("dsadsa", "1.3.4"))
}

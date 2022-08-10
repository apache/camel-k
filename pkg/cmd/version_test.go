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

	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const cmdVersion = "version"

//nolint: unparam
func initializeVersionCmdOptions(t *testing.T) (*versionCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	options, rootCmd := kamelTestPreAddCommandInit()
	versionCmdOptions := addTestVersionCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	return versionCmdOptions, rootCmd, *options
}

func addTestVersionCmd(options RootCmdOptions, rootCmd *cobra.Command) *versionCmdOptions {
	// add a testing version of version Command
	versionCmd, versionOptions := newCmdVersion(&options)
	versionCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(versionCmd)
	return versionOptions
}

func TestVersionNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeVersionCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdVersion, "--nonExistingFlag")
	assert.NotNil(t, err)
}

func TestVersionClient(t *testing.T) {
	_, rootCmd, _ := initializeVersionCmdOptions(t)
	output, err := test.ExecuteCommand(rootCmd, cmdVersion)
	assert.Nil(t, err)
	assert.Equal(t, fmt.Sprintf("Camel K Client %s\n", defaults.Version), output)
}

func TestVersionOperatorFlag(t *testing.T) {
	versionCmdOptions, rootCmd, _ := initializeVersionCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdVersion, "--operator")
	assert.Nil(t, err)
	assert.Equal(t, true, versionCmdOptions.Operator)
}

func TestVersionClientVerbose(t *testing.T) {
	versionCmdOptions, rootCmd, _ := initializeVersionCmdOptions(t)
	output, err := test.ExecuteCommand(rootCmd, cmdVersion, "-v")
	assert.Nil(t, err)
	assert.Equal(t, true, versionCmdOptions.Verbose)
	assert.Equal(t, fmt.Sprintf("Camel K Client %s\nGit Commit: %s\n", defaults.Version, defaults.GitCommit), output)
}

func TestCompatibleVersions(t *testing.T) {
	_, rootCmd, _ := initializeVersionCmdOptions(t)
	assert.Equal(t, true, compatibleVersions("1.3.0", "1.3.0", rootCmd))
	assert.Equal(t, true, compatibleVersions("1.3.0", "1.3.1", rootCmd))
	assert.Equal(t, true, compatibleVersions("1.3.0", "1.3.0-SNAPSHOT", rootCmd))
	assert.Equal(t, false, compatibleVersions("1.3.0", "1.2.0", rootCmd))
	assert.Equal(t, false, compatibleVersions("1.3.0", "2.3.0", rootCmd))
	assert.Equal(t, false, compatibleVersions("1.3.0", "dsadsa", rootCmd))
	assert.Equal(t, false, compatibleVersions("dsadsa", "1.3.4", rootCmd))
}

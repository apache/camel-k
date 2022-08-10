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

const cmdOperator = "operator"

//nolint: unparam
func initializeOperatorCmdOptions(t *testing.T) (*operatorCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	options, rootCmd := kamelTestPreAddCommandInit()
	operatorCmdOptions := addTestOperatorCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	return operatorCmdOptions, rootCmd, *options
}

//nolint: unparam
func addTestOperatorCmd(options RootCmdOptions, rootCmd *cobra.Command) *operatorCmdOptions {
	// add a testing version of operator Command
	operatorCmd, operatorOptions := newCmdOperator()
	operatorCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	operatorCmd.PostRunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	operatorCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(operatorCmd)
	return operatorOptions
}

func TestOperatorNoFlag(t *testing.T) {
	operatorCmdOptions, rootCmd, _ := initializeOperatorCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdOperator)
	assert.Nil(t, err)
	// Check default expected values
	assert.Equal(t, int32(8081), operatorCmdOptions.HealthPort)
	assert.Equal(t, int32(8080), operatorCmdOptions.MonitoringPort)
}

func TestOperatorNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeOperatorCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdOperator, "--nonExistingFlag")
	assert.NotNil(t, err)
}

func TestOperatorHealthPortFlag(t *testing.T) {
	operatorCmdOptions, rootCmd, _ := initializeOperatorCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdOperator, "--health-port", "7171")
	assert.Nil(t, err)
	assert.Equal(t, int32(7171), operatorCmdOptions.HealthPort)
}

func TestOperatorMonitoringPortFlag(t *testing.T) {
	operatorCmdOptions, rootCmd, _ := initializeOperatorCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdOperator, "--monitoring-port", "7172")
	assert.Nil(t, err)
	assert.Equal(t, int32(7172), operatorCmdOptions.MonitoringPort)
}

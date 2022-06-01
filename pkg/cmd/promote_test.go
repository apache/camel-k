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
)

const cmdPromote = "promote"

// nolint: unparam
func initializePromoteCmdOptions(t *testing.T) (*promoteCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	options, rootCmd := kamelTestPreAddCommandInit()
	promoteCmdOptions := addTestPromoteCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	return promoteCmdOptions, rootCmd, *options
}

// nolint: unparam
func addTestPromoteCmd(options RootCmdOptions, rootCmd *cobra.Command) *promoteCmdOptions {
	// add a testing version of operator Command
	operatorCmd, promoteOptions := newCmdPromote(&options)
	operatorCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	operatorCmd.PostRunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	operatorCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(operatorCmd)
	return promoteOptions
}

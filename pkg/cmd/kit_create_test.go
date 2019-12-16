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
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/spf13/cobra"
)

func addTestKitCreateCmd(options RootCmdOptions, rootCmd *cobra.Command) *kitCreateCommandOptions {
	//add a testing version of kitCreate Command
	kitCreateCmd, kitCreateCmdOptions := newKitCreateCmd(&options)
	kitCreateCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	kitCreateCmd.Args = test.ArbitraryArgs
	kitCmd := newTestCmdKit(&options)
	kitCmd.AddCommand(kitCreateCmd)
	rootCmd.AddCommand(kitCmd)
	return kitCreateCmdOptions
}

//TODO: add a proper test, take inspiration by run_test.go

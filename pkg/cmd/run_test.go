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
	"testing"
)

func addTestRunCmd(options RootCmdOptions, rootCmd *cobra.Command) *runCmdOptions {
	//add a testing version of run Command
	runCmd, runCmdOptions := newCmdRun(&options)
	runCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	runCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(runCmd)
	return runCmdOptions
}

func TestRunPropertyFlag(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()

	runCmdOptions := addTestRunCmd(options, rootCmd)

	rootCmd = kamelTestPostAddCommandInit(rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "run", "route.java", "--property", "key1=value,othervalue", "--property", "key2=value2")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(runCmdOptions.Properties) != 2 {
		t.Fatalf("Properties expected to contain: \n %v elements\nGot:\n %v elemtns\n", 2, len(runCmdOptions.Properties))
	}
	if runCmdOptions.Properties[0] != "key1=value,othervalue" || runCmdOptions.Properties[1] != "key2=value2" {
		t.Fatalf("Properties expected to be: \n %v\nGot:\n %v\n", "[key1=value,othervalue key2=value2]", runCmdOptions.Properties)
	}
}

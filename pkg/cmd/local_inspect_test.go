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

func addTestLocalInspectCmd(rootCmdOptions *RootCmdOptions, rootCmd *cobra.Command) *localInspectCmdOptions {
	// add a testing version of inspect Command
	localInspectCmd, localInspectCmdOptions := newCmdLocalInspect(rootCmdOptions)
	localInspectCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	localInspectCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(localInspectCmd)
	return localInspectCmdOptions
}

func TestLocalInspectAcceptsTraits(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()

	addTestLocalInspectCmd(options, rootCmd)

	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "inspect", "route.java", "-t", "jolokia.enabled=true", "--trait", "prometheus.enabled=true")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

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
	"github.com/stretchr/testify/require"
)

func addTestLocalInspectCmd(rootCmdOptions *RootCmdOptions, rootCmd *cobra.Command) *localInspectCmdOptions {
	localCmd, localCmdOptions := newCmdLocal(rootCmdOptions)
	// remove predefined sub commands
	localCmd.RemoveCommand(localCmd.Commands()...)
	// add a testing version of inspect Command
	localInspectCmd, localInspectCmdOptions := newCmdLocalInspect(localCmdOptions)
	localInspectCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	localInspectCmd.Args = test.ArbitraryArgs
	localCmd.AddCommand(localInspectCmd)
	rootCmd.AddCommand(localCmd)
	return localInspectCmdOptions
}

func TestLocalInspectAcceptsTraits(t *testing.T) {
	rootOptions, rootCmd := kamelTestPreAddCommandInit()
	addTestLocalInspectCmd(rootOptions, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "local", "inspect", "route.java",
		"-t", "jolokia.enabled=true",
		"--trait", "prometheus.enabled=true")

	require.NoError(t, err)
}

func TestLocalInspectWithDependencies(t *testing.T) {
	rootOptions, rootCmd := kamelTestPreAddCommandInit()
	options := addTestLocalInspectCmd(rootOptions, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "local", "inspect", "route.java",
		"-d", "camel-amqp",
		"-d", "camel:bean",
		"-d", "camel-quarkus-controlbus",
		"-d", "camel-quarkus:directvm",
		"--dependency", "mvn:test:component:1.0.0")

	require.NoError(t, err)
	assert.Len(t, options.Dependencies, 5)
	assert.ElementsMatch(t, options.Dependencies, []string{
		"camel:amqp", "camel:bean", "camel:controlbus", "camel:directvm", "mvn:test:component:1.0.0",
	})
}

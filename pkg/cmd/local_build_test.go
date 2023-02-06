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

func addTestLocalBuildCmd(rootCmdOptions *RootCmdOptions, rootCmd *cobra.Command) *localBuildCmdOptions {
	localCmd, localCmdOptions := newCmdLocal(rootCmdOptions)
	// remove predefined sub commands
	localCmd.RemoveCommand(localCmd.Commands()...)
	// add a testing version of build Command
	localBuildCmd, localBuildCmdOptions := newCmdLocalBuild(localCmdOptions)
	localBuildCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	localBuildCmd.Args = test.ArbitraryArgs
	localCmd.AddCommand(localBuildCmd)
	rootCmd.AddCommand(localCmd)
	return localBuildCmdOptions
}

func TestLocalBuildAcceptsTraits(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()
	addTestLocalBuildCmd(options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "local", "build", "route.java",
		"-t", "jolokia.enabled=true",
		"--trait", "prometheus.enabled=true")

	require.NoError(t, err)
}

func TestLocalBuildWithDependencies(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()
	localBuildCmdOptions := addTestLocalBuildCmd(options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "local", "build", "route.java",
		"-d", "camel-amqp",
		"-d", "camel:bean",
		"-d", "camel-quarkus-controlbus",
		"-d", "camel-quarkus:directvm",
		"--dependency", "mvn:test:component:1.0.0")

	require.NoError(t, err)
	assert.Len(t, localBuildCmdOptions.Dependencies, 5)
	assert.ElementsMatch(t, localBuildCmdOptions.Dependencies, []string{
		"camel:amqp", "camel:bean", "camel:controlbus", "camel:directvm", "mvn:test:component:1.0.0",
	})
}

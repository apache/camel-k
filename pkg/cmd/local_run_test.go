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

func addTestLocalRunCmd(rootCmdOptions *RootCmdOptions, rootCmd *cobra.Command) *localRunCmdOptions {
	localCmd, localCmdOptions := newCmdLocal(rootCmdOptions)
	// remove predefined sub commands
	localCmd.RemoveCommand(localCmd.Commands()...)
	// add a testing version of run Command
	localRunCmd, localRunCmdOptions := newCmdLocalRun(localCmdOptions)
	localRunCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	localRunCmd.Args = test.ArbitraryArgs
	localCmd.AddCommand(localRunCmd)
	rootCmd.AddCommand(localCmd)
	return localRunCmdOptions
}

func TestLocalRunPropertyFileFlag(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()
	localRunCmdOptions := addTestLocalRunCmd(options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "local", "run", "route.java",
		"--property-file", "file1.properties",
		"--property-file", "file2.properties")

	require.NoError(t, err)
	if len(localRunCmdOptions.PropertyFiles) != 2 {
		t.Fatalf(
			"Property files expected to contain: \n %v elements\nGot:\n %v elements\n",
			2, len(localRunCmdOptions.PropertyFiles))
	}
	if localRunCmdOptions.PropertyFiles[0] != "file1.properties" ||
		localRunCmdOptions.PropertyFiles[1] != "file2.properties" {
		t.Fatalf(
			"Property files expected to be: \n %v\nGot:\n %v\n", "[file1.properties, file2.properties]",
			localRunCmdOptions.PropertyFiles)
	}
}

func TestLocalRunPropertiesFlag(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()
	localRunCmdOptions := addTestLocalRunCmd(options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "local", "run", "route.java",
		"-p", "prop1=value1",
		"-p", "prop2=value2")

	require.NoError(t, err)
	if len(localRunCmdOptions.Properties) != 2 {
		t.Fatalf(
			"Additional dependencies expected to contain: \n %v elements\nGot:\n %v elements\n",
			2, len(localRunCmdOptions.Properties))
	}
	if localRunCmdOptions.Properties[0] != "prop1=value1" || localRunCmdOptions.Properties[1] != "prop2=value2" {
		t.Fatalf(
			"Additional dependencies expected to be: \n %v\nGot:\n %v\n", "[prop1=value1, prop2=value2]",
			localRunCmdOptions.Properties)
	}
}

func TestLocalRunAdditionalDependenciesFlag(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()
	localRunCmdOptions := addTestLocalRunCmd(options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "local", "run", "route.java",
		"-d", "camel-amqp",
		"-d", "camel:bean",
		"-d", "camel-quarkus-controlbus",
		"-d", "camel-quarkus:directvm",
		"--dependency", "mvn:test:component:1.0.0")

	require.NoError(t, err)
	assert.Len(t, localRunCmdOptions.Dependencies, 5)
	assert.ElementsMatch(t, localRunCmdOptions.Dependencies, []string{
		"camel:amqp", "camel:bean", "camel:controlbus", "camel:directvm", "mvn:test:component:1.0.0",
	})
}

func TestLocalRunAcceptsTraits(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()
	addTestLocalRunCmd(options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "local", "run", "route.java",
		"-t", "jolokia.enabled=true",
		"--trait", "prometheus.enabled=true")

	require.NoError(t, err)
}

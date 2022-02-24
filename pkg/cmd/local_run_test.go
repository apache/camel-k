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

func addTestLocalRunCmd(rootCmdOptions *RootCmdOptions, rootCmd *cobra.Command) *localRunCmdOptions {
	// add a testing version of run Command
	localRunCmd, localRunCmdOptions := newCmdLocalRun(rootCmdOptions)
	localRunCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	localRunCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(localRunCmd)
	return localRunCmdOptions
}

func TestLocalRunPropertyFileFlag(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()

	localRunCmdOptions := addTestLocalRunCmd(options, rootCmd)

	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "run", "route.java", "--property-file", "file1.properties", "--property-file", "file2.properties")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(localRunCmdOptions.PropertyFiles) != 2 {
		t.Fatalf("Property files expected to contain: \n %v elements\nGot:\n %v elements\n", 2, len(localRunCmdOptions.PropertyFiles))
	}
	if localRunCmdOptions.PropertyFiles[0] != "file1.properties" || localRunCmdOptions.PropertyFiles[1] != "file2.properties" {
		t.Fatalf("Property files expected to be: \n %v\nGot:\n %v\n", "[file1.properties, file2.properties]", localRunCmdOptions.PropertyFiles)
	}
}

func TestLocalRunPropertiesFlag(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()

	localRunCmdOptions := addTestLocalRunCmd(options, rootCmd)

	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "run", "route.java", "-p", "prop1=value1", "-p", "prop2=value2")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(localRunCmdOptions.Properties) != 2 {
		t.Fatalf("Additional dependencies expected to contain: \n %v elements\nGot:\n %v elements\n", 2, len(localRunCmdOptions.Properties))
	}
	if localRunCmdOptions.Properties[0] != "prop1=value1" || localRunCmdOptions.Properties[1] != "prop2=value2" {
		t.Fatalf("Additional dependencies expected to be: \n %v\nGot:\n %v\n", "[prop1=value1, prop2=value2]", localRunCmdOptions.Properties)
	}
}

func TestLocalRunAdditionalDependenciesFlag(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()

	localRunCmdOptions := addTestLocalRunCmd(options, rootCmd)

	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "run", "route.java", "-d", "mvn:camel-component-1", "-d", "mvn:camel-component-2")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(localRunCmdOptions.AdditionalDependencies) != 2 {
		t.Fatalf("Additional dependencies expected to contain: \n %v elements\nGot:\n %v elements\n", 2, len(localRunCmdOptions.AdditionalDependencies))
	}
	if localRunCmdOptions.AdditionalDependencies[0] != "mvn:camel-component-1" || localRunCmdOptions.AdditionalDependencies[1] != "mvn:camel-component-2" {
		t.Fatalf("Additional dependencies expected to be: \n %v\nGot:\n %v\n", "[mvn:camel-component-1, mvn:camel-component-2]", localRunCmdOptions.AdditionalDependencies)
	}
}

func TestLocalRunAcceptsTraits(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()

	addTestLocalRunCmd(options, rootCmd)

	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "run", "route.java", "-t", "jolokia.enabled=true", "--trait", "prometheus.enabled=true")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

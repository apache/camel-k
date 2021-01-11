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
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/apache/camel-k/pkg/util/test"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func kamelTestPostAddCommandInit(t *testing.T, rootCmd *cobra.Command) {
	err := kamelPostAddCommandInit(rootCmd)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func kamelTestPreAddCommandInit() (*RootCmdOptions, *cobra.Command) {
	fakeClient, _ := test.NewFakeClient()
	options := RootCmdOptions{
		Context: context.Background(),
		_client: fakeClient,
	}
	rootCmd := kamelPreAddCommandInit(&options)
	rootCmd.Run = test.EmptyRun
	return &options, rootCmd
}

func TestLoadFromEnvVar(t *testing.T) {
	//shows how to include a "," character inside an env value see VAR1 value
	os.Setenv("KAMEL_RUN_ENVS", "\"VAR1=value,\"\"othervalue\"\"\",VAR2=value2")

	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)

	_, err := test.ExecuteCommand(rootCmd, "run", "route.java")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(runCmdOptions.EnvVars) != 2 {
		t.Fatalf("Properties expected to contain: \n %v elements\nGot:\n %v elemtns\n", 2, len(runCmdOptions.EnvVars))
	}
	if runCmdOptions.EnvVars[0] != "VAR1=value,\"othervalue\"" || runCmdOptions.EnvVars[1] != "VAR2=value2" {
		t.Fatalf("EnvVars expected to be: \n %v\nGot:\n %v\n", "[VAR1=value,\"othervalue\" VAR=value2]", runCmdOptions.EnvVars)
	}
}

func TestLoadFromFile(t *testing.T) {
	//shows how to include a "," character inside a property value see VAR1 value
	var propertiesFile = []byte(`kamel.run.envs: "VAR1=value,""othervalue""",VAR2=value2`)
	viper.SetConfigType("properties")
	readViperConfigFromBytes(propertiesFile, t)

	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)

	_, err := test.ExecuteCommand(rootCmd, "run", "route.java")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(runCmdOptions.EnvVars) != 2 {
		t.Fatalf("Properties expected to contain: \n %v elements\nGot:\n %v elemtns\n", 2, len(runCmdOptions.EnvVars))
	}
	if runCmdOptions.EnvVars[0] != "VAR1=value,\"othervalue\"" || runCmdOptions.EnvVars[1] != "VAR2=value2" {
		t.Fatalf("EnvVars expected to be: \n %v\nGot:\n %v\n", "[VAR1=value,\"othervalue\" VAR=value2]", runCmdOptions.EnvVars)
	}
}

func TestPrecedenceEnvVarOverFile(t *testing.T) {
	os.Setenv("KAMEL_RUN_ENVS", "VAR1=envVar")
	var propertiesFile = []byte(`kamel.run.envs: VAR2=file`)
	viper.SetConfigType("properties")
	readViperConfigFromBytes(propertiesFile, t)

	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)

	_, err := test.ExecuteCommand(rootCmd, "run", "route.java")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(runCmdOptions.EnvVars) != 1 {
		t.Fatalf("Properties expected to contain: \n %v elements\nGot:\n %v elements\n", 1, len(runCmdOptions.EnvVars))
	}
	if runCmdOptions.EnvVars[0] != "VAR1=envVar" {
		t.Fatalf("EnvVars expected to be: \n %v\nGot:\n %v\n", "VAR1=envVar", runCmdOptions.EnvVars)
	}
}

func TestPrecedenceCommandLineOverEverythingElse(t *testing.T) {
	os.Setenv("KAMEL_RUN_ENVS", "VAR1=envVar")
	var propertiesFile = []byte(`kamel.run.envs: VAR2=file`)
	viper.SetConfigType("properties")
	readViperConfigFromBytes(propertiesFile, t)

	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)

	_, err := test.ExecuteCommand(rootCmd, "run", "route.java", "--env", "VAR3=commandLine")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(runCmdOptions.EnvVars) != 1 {
		t.Fatalf("Properties expected to contain: \n %v elements\nGot:\n %v elements\n", 1, len(runCmdOptions.EnvVars))
	}
	if runCmdOptions.EnvVars[0] != "VAR3=commandLine" {
		t.Fatalf("EnvVars expected to be: \n %v\nGot:\n %v\n", "VAR3=commandLine", runCmdOptions.EnvVars)
	}
}

func readViperConfigFromBytes(propertiesFile []byte, t *testing.T) {
	unexpectedErr := viper.ReadConfig(bytes.NewReader(propertiesFile))
	if unexpectedErr != nil {
		t.Fatalf("Unexpected error: %v", unexpectedErr)
	}
}

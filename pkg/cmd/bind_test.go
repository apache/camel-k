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
	"os"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cmdBind = "bind"

// nolint: unparam
func initializeBindCmdOptions(t *testing.T) (*bindCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	defaultIntegrationPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	fakeClient, _ := internal.NewFakeClient(&defaultIntegrationPlatform)

	options, rootCmd := kamelTestPreAddCommandInitWithClient(fakeClient)
	bindCmdOptions := addTestBindCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd, options)

	return bindCmdOptions, rootCmd, *options
}

func addTestBindCmd(options RootCmdOptions, rootCmd *cobra.Command) *bindCmdOptions {
	// add a testing version of bind Command
	bindCmd, bindOptions := newCmdBind(&options)
	bindCmd.Args = ArbitraryArgs
	rootCmd.AddCommand(bindCmd)
	return bindOptions
}

func TestBindOutputJSON(t *testing.T) {
	buildCmdOptions, bindCmd, _ := initializeBindCmdOptions(t)
	output, err := ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "json")
	assert.Equal(t, "json", buildCmdOptions.OutputFormat)

	require.NoError(t, err)
	assert.Equal(t, `{"kind":"Pipe","apiVersion":"camel.apache.org/v1","metadata":{"name":"my-to-my","creationTimestamp":null,"annotations":{"camel.apache.org/operator.id":"camel-k"}},"spec":{"source":{"uri":"my:src"},"sink":{"uri":"my:dst"}},"status":{}}`, output)
}

func TestBindOutputYAML(t *testing.T) {
	buildCmdOptions, bindCmd, _ := initializeBindCmdOptions(t)
	output, err := ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "yaml")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	require.NoError(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  annotations:
    camel.apache.org/operator.id: camel-k
  creationTimestamp: null
  name: my-to-my
spec:
  sink:
    uri: my:dst
  source:
    uri: my:src
status: {}
`, output)
}

func TestBindOutputUnknownFormat(t *testing.T) {
	buildCmdOptions, bindCmd, _ := initializeBindCmdOptions(t)
	output, _ := ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "fail")
	assert.Equal(t, "fail", buildCmdOptions.OutputFormat)

	assert.Equal(t, "invalid output format option 'fail', should be one of: yaml|json\n", output)
}

func TestBindErrorHandlerDLCKamelet(t *testing.T) {
	buildCmdOptions, bindCmd, _ := initializeBindCmdOptions(t)
	output, err := ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "yaml",
		"--error-handler", "sink:my-kamelet", "-p", "error-handler.my-prop=value")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	require.NoError(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  annotations:
    camel.apache.org/operator.id: camel-k
  creationTimestamp: null
  name: my-to-my
spec:
  errorHandler:
    sink:
      endpoint:
        properties:
          my-prop: value
        ref:
          apiVersion: camel.apache.org/v1
          kind: Kamelet
          name: my-kamelet
  sink:
    uri: my:dst
  source:
    uri: my:src
status: {}
`, output)
}

func TestBindErrorHandlerNone(t *testing.T) {
	buildCmdOptions, bindCmd, _ := initializeBindCmdOptions(t)
	output, err := ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "yaml",
		"--error-handler", "none")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	require.NoError(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  annotations:
    camel.apache.org/operator.id: camel-k
  creationTimestamp: null
  name: my-to-my
spec:
  errorHandler:
    none: null
  sink:
    uri: my:dst
  source:
    uri: my:src
status: {}
`, output)
}

func TestBindErrorHandlerLog(t *testing.T) {
	buildCmdOptions, bindCmd, _ := initializeBindCmdOptions(t)
	output, err := ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "yaml",
		"--error-handler", "log")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	require.NoError(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  annotations:
    camel.apache.org/operator.id: camel-k
  creationTimestamp: null
  name: my-to-my
spec:
  errorHandler:
    log: null
  sink:
    uri: my:dst
  source:
    uri: my:src
status: {}
`, output)
}

func TestBindTraits(t *testing.T) {
	buildCmdOptions, bindCmd, _ := initializeBindCmdOptions(t)
	output, err := ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "yaml",
		"-t", "mount.configs=configmap:my-cm")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	require.NoError(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  annotations:
    camel.apache.org/operator.id: camel-k
    trait.camel.apache.org/mount.configs: configmap:my-cm
  creationTimestamp: null
  name: my-to-my
spec:
  sink:
    uri: my:dst
  source:
    uri: my:src
status: {}
`, output)
}

func TestBindTraitsArray(t *testing.T) {
	buildCmdOptions, bindCmd, _ := initializeBindCmdOptions(t)
	output, err := ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "yaml",
		"-t", "camel.properties=a=1", "-t", "camel.properties=b=2")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	require.NoError(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  annotations:
    camel.apache.org/operator.id: camel-k
    trait.camel.apache.org/camel.properties: '[a=1,b=2]'
  creationTimestamp: null
  name: my-to-my
spec:
  sink:
    uri: my:dst
  source:
    uri: my:src
status: {}
`, output)
}

func TestBindSteps(t *testing.T) {
	buildCmdOptions, bindCmd, _ := initializeBindCmdOptions(t)
	output, err := ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "yaml",
		"--step", "dst:step1", "--step", "src:step2",
		"-p", "step-1.var1=my-step1-var1", "-p", "step-1.var2=my-step1-var2",
		"-p", "step-2.var1=my-step2-var1", "-p", "step-2.var2=my-step2-var2")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	require.NoError(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  annotations:
    camel.apache.org/operator.id: camel-k
  creationTimestamp: null
  name: my-to-my
spec:
  sink:
    uri: my:dst
  source:
    uri: my:src
  steps:
  - properties:
      var1: my-step1-var1
      var2: my-step1-var2
    uri: dst:step1
  - properties:
      var1: my-step2-var1
      var2: my-step2-var2
    uri: src:step2
status: {}
`, output)
}

func TestBindServiceAccountName(t *testing.T) {
	_, bindCmd, _ := initializeBindCmdOptions(t)
	output, err := ExecuteCommand(bindCmd, cmdBind, "timer:foo", "log:bar",
		"-o", "yaml",
		"--service-account", "my-service-account")

	require.NoError(t, err)
	assert.Contains(t, output, "serviceAccountName: my-service-account")
}

func TestBindOutputWithoutKubernetesCluster(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "camel-k-kubeconfig-*")
	require.NoError(t, err)

	bindCmdOptions, bindCmd, _ := initializeBindCmdOptions(t)
	bindCmdOptions._client = nil // remove the default fake client which can bypass this test
	bindCmdOptions.KubeConfig = tmpFile.Name()
	_, err = ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "yaml")
	require.NoError(t, err)
}

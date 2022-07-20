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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const cmdBind = "bind"

// nolint: unparam
func initializeBindCmdOptions(t *testing.T) (*bindCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	defaultIntegrationPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	fakeClient, _ := test.NewFakeClient(&defaultIntegrationPlatform)

	options, rootCmd := kamelTestPreAddCommandInitWithClient(fakeClient)
	bindCmdOptions := addTestBindCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	return bindCmdOptions, rootCmd, *options
}

func addTestBindCmd(options RootCmdOptions, rootCmd *cobra.Command) *bindCmdOptions {
	// add a testing version of bind Command
	bindCmd, bindOptions := newCmdBind(&options)
	bindCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(bindCmd)
	return bindOptions
}

func TestBindOutputJSON(t *testing.T) {
	buildCmdOptions, bindCmd, _ := initializeBindCmdOptions(t)
	output, err := test.ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "json")
	assert.Equal(t, "json", buildCmdOptions.OutputFormat)

	assert.Nil(t, err)
	// nolint: lll
	assert.Equal(t,
		`{"kind":"KameletBinding","apiVersion":"camel.apache.org/v1alpha1","metadata":{"name":"my-to-my","creationTimestamp":null,"annotations":{"camel.apache.org/operator.id":"camel-k"}},"spec":{"source":{"uri":"my:src"},"sink":{"uri":"my:dst"}},"status":{}}`,
		output)
}

func TestBindOutputYAML(t *testing.T) {
	buildCmdOptions, bindCmd, _ := initializeBindCmdOptions(t)
	output, err := test.ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "yaml")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	assert.Nil(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1alpha1
kind: KameletBinding
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
	output, _ := test.ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "fail")
	assert.Equal(t, "fail", buildCmdOptions.OutputFormat)

	assert.Equal(t, "invalid output format option 'fail', should be one of: yaml|json\n", output)
}

func TestBindErrorHandlerDLCKamelet(t *testing.T) {
	buildCmdOptions, bindCmd, _ := initializeBindCmdOptions(t)
	output, err := test.ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "yaml",
		"--error-handler", "sink:my-kamelet", "-p", "error-handler.my-prop=value")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	assert.Nil(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1alpha1
kind: KameletBinding
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
          apiVersion: camel.apache.org/v1alpha1
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
	output, err := test.ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "yaml",
		"--error-handler", "none")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	assert.Nil(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1alpha1
kind: KameletBinding
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
	output, err := test.ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst", "-o", "yaml",
		"--error-handler", "log")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	assert.Nil(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1alpha1
kind: KameletBinding
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
	output, err := test.ExecuteCommand(bindCmd, cmdBind, "my:src", "my:dst",
		"-o", "yaml", "-t", "mount.configs=configmap:my-cm", "-c", "my-service-binding")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	assert.Nil(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1alpha1
kind: KameletBinding
metadata:
  annotations:
    camel.apache.org/operator.id: camel-k
  creationTimestamp: null
  name: my-to-my
spec:
  integration:
    traits:
      mount:
        configs:
        - configmap:my-cm
      service-binding:
        services:
        - my-service-binding
  sink:
    uri: my:dst
  source:
    uri: my:src
status: {}
`, output)
}

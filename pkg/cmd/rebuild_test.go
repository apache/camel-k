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

	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

const cmdRebuild = "rebuild"

// nolint: unparam
func initializeRebuildOptions(t *testing.T, initObjs ...runtime.Object) (*rebuildCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()
	fakeClient, err := test.NewFakeClient(initObjs...)
	assert.Nil(t, err)
	options, rootCmd := kamelTestPreAddCommandInitWithClient(fakeClient)
	options.Namespace = "default"
	rebuildCmdOptions := addTestRebuildCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	return rebuildCmdOptions, rootCmd, *options
}

func addTestRebuildCmd(options RootCmdOptions, rootCmd *cobra.Command) *rebuildCmdOptions {
	rebuildCmd, rebuildOptions := newCmdRebuild(&options)
	rebuildCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(rebuildCmd)
	return rebuildOptions
}

func TestRebuildNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeRebuildOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRebuild, "--nonExistingFlag")
	assert.NotNil(t, err)
}

func TestRebuildAllFlag(t *testing.T) {
	rebuildCmdOptions, rootCmd, _ := initializeRebuildOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRebuild, "--all")
	assert.Nil(t, err)
	assert.Equal(t, true, rebuildCmdOptions.RebuildAll)
}

func TestRebuildAllKameletBindingsAndIntegrations(t *testing.T) {
	defaultIntegration := nominalIntegration("my-it-test")
	defaultKB := nominalKameletBinding("my-kb-test")
	itGeneratedByKlb := nominalIntegration("my-kb-test")
	itGeneratedByKlb.Labels = map[string]string{
		kubernetes.CamelCreatorLabelKind: "KameletBinding",
	}

	_, rebuildCmd, _ := initializeRebuildOptions(t, &defaultIntegration, &defaultKB, &itGeneratedByKlb)
	output, err := test.ExecuteCommand(rebuildCmd, cmdRebuild, "--all")
	assert.Nil(t, err)
	assert.Contains(t, output, "1 kamelet bindings have been rebuilt")
	assert.Contains(t, output, "1 integrations have been rebuilt")
}

func TestRebuildNone(t *testing.T) {
	defaultIntegration := nominalIntegration("my-it-test")
	defaultKB := nominalKameletBinding("my-kb-test")
	itGeneratedByKlb := nominalIntegration("my-kb-test")
	itGeneratedByKlb.Labels = map[string]string{
		kubernetes.CamelCreatorLabelKind: "KameletBinding",
	}

	_, rebuildCmd, _ := initializeRebuildOptions(t, &defaultIntegration, &defaultKB, &itGeneratedByKlb)
	output, err := test.ExecuteCommand(rebuildCmd, cmdRebuild, "my-missing")
	assert.NotNil(t, err)
	assert.NotContains(t, output, "have been rebuilt")
	assert.Contains(t, output, "could not find kamelet binding my-missing in namespace default")
	assert.Contains(t, output, "could not find integration my-missing in namespace default")
}

func TestRebuildKameletBindingOnly(t *testing.T) {
	defaultIntegration := nominalIntegration("my-it-test")
	defaultKB := nominalKameletBinding("my-kb-test")
	itGeneratedByKlb := nominalIntegration("my-kb-test")
	itGeneratedByKlb.Labels = map[string]string{
		kubernetes.CamelCreatorLabelKind: "KameletBinding",
	}

	_, rebuildCmd, _ := initializeRebuildOptions(t, &defaultIntegration, &defaultKB, &itGeneratedByKlb)
	output, err := test.ExecuteCommand(rebuildCmd, cmdRebuild, "my-kb-test")
	assert.Nil(t, err)
	assert.Contains(t, output, "1 kamelet bindings have been rebuilt")
	assert.NotContains(t, output, "1 integrations have been rebuilt")
}

func TestRebuildIntegrationOnly(t *testing.T) {
	defaultIntegration := nominalIntegration("my-it-test")
	defaultKB := nominalKameletBinding("my-kb-test")
	itGeneratedByKlb := nominalIntegration("my-kb-test")
	itGeneratedByKlb.Labels = map[string]string{
		kubernetes.CamelCreatorLabelKind: "KameletBinding",
	}

	_, rebuildCmd, _ := initializeRebuildOptions(t, &defaultIntegration, &defaultKB, &itGeneratedByKlb)
	output, err := test.ExecuteCommand(rebuildCmd, cmdRebuild, "my-it-test")
	assert.Nil(t, err)
	assert.NotContains(t, output, "1 kamelet bindings have been rebuilt")
	assert.Contains(t, output, "1 integrations have been rebuilt")
}

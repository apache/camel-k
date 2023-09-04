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
	"fmt"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

const cmdPromote = "promote"

// nolint: unparam
func initializePromoteCmdOptions(t *testing.T, initObjs ...runtime.Object) (*promoteCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()
	fakeClient, err := test.NewFakeClient(initObjs...)
	assert.Nil(t, err)
	options, rootCmd := kamelTestPreAddCommandInitWithClient(fakeClient)
	options.Namespace = "default"
	promoteCmdOptions := addTestPromoteCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	return promoteCmdOptions, rootCmd, *options
}

func addTestPromoteCmd(options RootCmdOptions, rootCmd *cobra.Command) *promoteCmdOptions {
	promoteCmd, promoteOptions := newCmdPromote(&options)
	promoteCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(promoteCmd)
	return promoteOptions
}

func TestIntegrationNotCompatible(t *testing.T) {
	srcPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	srcPlatform.Status.Version = defaults.Version
	srcPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	srcPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	dstPlatform := v1.NewIntegrationPlatform("prod-namespace", platform.DefaultPlatformName)
	dstPlatform.Status.Version = "0.0.1"
	dstPlatform.Status.Build.RuntimeVersion = "0.0.1"
	dstPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	defaultIntegration := nominalIntegration("my-it-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	_, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &srcCatalog, &dstCatalog)
	_, err := test.ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "--to", "prod-namespace", "-n", "default")
	assert.NotNil(t, err)
	assert.Equal(t,
		fmt.Sprintf("could not verify operators compatibility: source (%s) and destination (0.0.1) Camel K operator versions are not compatible", defaults.Version),
		err.Error(),
	)
}

func TestIntegrationDryRun(t *testing.T) {
	srcPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	srcPlatform.Status.Version = defaults.Version
	srcPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	srcPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	dstPlatform := v1.NewIntegrationPlatform("prod-namespace", platform.DefaultPlatformName)
	dstPlatform.Status.Version = defaults.Version
	dstPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	dstPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	defaultIntegration := nominalIntegration("my-it-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	promoteCmdOptions, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &srcCatalog, &dstCatalog)
	output, err := test.ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "--to", "prod-namespace", "-o", "yaml", "-n", "default")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	assert.Nil(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Integration
metadata:
  creationTimestamp: null
  name: my-it-test
  namespace: prod-namespace
spec:
  traits:
    container:
      image: my-special-image
status: {}
`, output)
}

func nominalIntegration(name string) v1.Integration {
	it := v1.NewIntegration("default", name)
	it.Status.Phase = v1.IntegrationPhaseRunning
	it.Status.Image = "my-special-image"
	return it
}

func TestPipeDryRun(t *testing.T) {
	srcPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	srcPlatform.Status.Version = defaults.Version
	srcPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	srcPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	dstPlatform := v1.NewIntegrationPlatform("prod-namespace", platform.DefaultPlatformName)
	dstPlatform.Status.Version = defaults.Version
	dstPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	dstPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	defaultKB := nominalPipe("my-kb-test")
	defaultIntegration := nominalIntegration("my-kb-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	promoteCmdOptions, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultKB, &defaultIntegration, &srcCatalog, &dstCatalog)
	output, err := test.ExecuteCommand(promoteCmd, cmdPromote, "my-kb-test", "--to", "prod-namespace", "-o", "yaml", "-n", "default")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	assert.Nil(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  creationTimestamp: null
  name: my-kb-test
  namespace: prod-namespace
spec:
  integration:
    traits:
      container:
        image: my-special-image
  sink: {}
  source: {}
status: {}
`, output)
}

func nominalPipe(name string) v1.Pipe {
	kb := v1.NewPipe("default", name)
	kb.Status.Phase = v1.PipePhaseReady
	return kb
}

func createTestCamelCatalog(platform v1.IntegrationPlatform) v1.CamelCatalog {
	c := v1.NewCamelCatalog(platform.Namespace, defaults.DefaultRuntimeVersion)
	c.Spec = v1.CamelCatalogSpec{Runtime: v1.RuntimeSpec{Provider: platform.Status.Build.RuntimeProvider, Version: platform.Status.Build.RuntimeVersion}}
	return c
}

func TestIntegrationWithMetadataDryRun(t *testing.T) {
	srcPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	srcPlatform.Status.Version = defaults.Version
	srcPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	srcPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	dstPlatform := v1.NewIntegrationPlatform("prod-namespace", platform.DefaultPlatformName)
	dstPlatform.Status.Version = defaults.Version
	dstPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	dstPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	defaultIntegration := nominalIntegration("my-it-test")
	defaultIntegration.Annotations = map[string]string{
		"camel.apache.org/operator.id": "camel-k",
		"my-annotation":                "my-value",
	}
	defaultIntegration.Labels = map[string]string{
		"my-label": "my-value",
	}
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	promoteCmdOptions, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &srcCatalog, &dstCatalog)
	output, err := test.ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "--to", "prod-namespace", "-o", "yaml", "-n", "default")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	assert.Nil(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Integration
metadata:
  annotations:
    my-annotation: my-value
  creationTimestamp: null
  labels:
    my-label: my-value
  name: my-it-test
  namespace: prod-namespace
spec:
  traits:
    container:
      image: my-special-image
status: {}
`, output)
}

func TestPipeWithMetadataDryRun(t *testing.T) {
	srcPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	srcPlatform.Status.Version = defaults.Version
	srcPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	srcPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	dstPlatform := v1.NewIntegrationPlatform("prod-namespace", platform.DefaultPlatformName)
	dstPlatform.Status.Version = defaults.Version
	dstPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	dstPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	defaultKB := nominalPipe("my-kb-test")
	defaultKB.Annotations = map[string]string{
		"camel.apache.org/operator.id": "camel-k",
		"my-annotation":                "my-value",
	}
	defaultKB.Labels = map[string]string{
		"my-label": "my-value",
	}
	defaultIntegration := nominalIntegration("my-kb-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	promoteCmdOptions, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultKB, &defaultIntegration, &srcCatalog, &dstCatalog)
	output, err := test.ExecuteCommand(promoteCmd, cmdPromote, "my-kb-test", "--to", "prod-namespace", "-o", "yaml", "-n", "default")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	assert.Nil(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  annotations:
    my-annotation: my-value
  creationTimestamp: null
  labels:
    my-label: my-value
  name: my-kb-test
  namespace: prod-namespace
spec:
  integration:
    traits:
      container:
        image: my-special-image
  sink: {}
  source: {}
status: {}
`, output)
}

func TestItImageOnly(t *testing.T) {
	srcPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	srcPlatform.Status.Version = defaults.Version
	srcPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	srcPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	dstPlatform := v1.NewIntegrationPlatform("prod-namespace", platform.DefaultPlatformName)
	dstPlatform.Status.Version = defaults.Version
	dstPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	dstPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	defaultIntegration := nominalIntegration("my-it-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	_, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &srcCatalog, &dstCatalog)
	output, err := test.ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "--to", "prod-namespace", "-i", "-n", "default")
	assert.Nil(t, err)
	assert.Equal(t, "my-special-image\n", output)
}

func TestPipeImageOnly(t *testing.T) {
	srcPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	srcPlatform.Status.Version = defaults.Version
	srcPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	srcPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	dstPlatform := v1.NewIntegrationPlatform("prod-namespace", platform.DefaultPlatformName)
	dstPlatform.Status.Version = defaults.Version
	dstPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	dstPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	defaultKB := nominalPipe("my-kb-test")
	defaultIntegration := nominalIntegration("my-kb-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	_, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultKB, &defaultIntegration, &srcCatalog, &dstCatalog)
	output, err := test.ExecuteCommand(promoteCmd, cmdPromote, "my-kb-test", "--to", "prod-namespace", "-i", "-n", "default")
	assert.Nil(t, err)
	assert.Equal(t, "my-special-image\n", output)
}

func TestIntegrationToOperatorId(t *testing.T) {
	srcPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	srcPlatform.Status.Version = defaults.Version
	srcPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	srcPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	dstPlatform := v1.NewIntegrationPlatform("prod-namespace", platform.DefaultPlatformName)
	dstPlatform.Status.Version = defaults.Version
	dstPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	dstPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	defaultIntegration := nominalIntegration("my-it-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	// Verify default (missing) operator Id
	promoteCmdOptions, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &srcCatalog, &dstCatalog)
	output, err := test.ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "-x", "my-prod-operator", "-o", "yaml", "--to", "prod")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	assert.Nil(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Integration
metadata:
  annotations:
    camel.apache.org/operator.id: my-prod-operator
  creationTimestamp: null
  name: my-it-test
  namespace: prod
spec:
  traits:
    container:
      image: my-special-image
status: {}
`, output)
	// Verify also when the operator Id is set in the integration
	defaultIntegration.Annotations = map[string]string{
		"camel.apache.org/operator.id": "camel-k",
	}
	promoteCmdOptions, promoteCmd, _ = initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &srcCatalog, &dstCatalog)
	output, err = test.ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "-x", "my-prod-operator", "-o", "yaml", "--to", "prod")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	assert.Nil(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Integration
metadata:
  annotations:
    camel.apache.org/operator.id: my-prod-operator
  creationTimestamp: null
  name: my-it-test
  namespace: prod
spec:
  traits:
    container:
      image: my-special-image
status: {}
`, output)
}

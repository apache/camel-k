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
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/io"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

const cmdPromote = "promote"

// nolint: unparam
func initializePromoteCmdOptions(t *testing.T, initObjs ...runtime.Object) (*promoteCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()
	fakeClient, err := internal.NewFakeClient(initObjs...)
	require.NoError(t, err)
	options, rootCmd := kamelTestPreAddCommandInitWithClient(fakeClient)
	options.Namespace = "default"
	promoteCmdOptions := addTestPromoteCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd, options)

	return promoteCmdOptions, rootCmd, *options
}

func addTestPromoteCmd(options RootCmdOptions, rootCmd *cobra.Command) *promoteCmdOptions {
	promoteCmd, promoteOptions := newCmdPromote(&options)
	promoteCmd.Args = ArbitraryArgs
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
	defaultIntegration, defaultKit := nominalIntegration("my-it-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	_, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &defaultKit, &srcCatalog, &dstCatalog)
	_, err := ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "--to", "prod-namespace", "-n", "default")
	require.Error(t, err)
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
	defaultIntegration, defaultKit := nominalIntegration("my-it-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	promoteCmdOptions, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &defaultKit, &srcCatalog, &dstCatalog)
	output, err := ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "--to", "prod-namespace", "-o", "yaml", "-n", "default")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	require.NoError(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Integration
metadata:
  creationTimestamp: null
  name: my-it-test
  namespace: prod-namespace
spec:
  traits:
    camel:
      runtimeVersion: 1.2.3
    container:
      image: my-special-image
    jvm:
      classpath: /path/to/artifact-1/*:/path/to/artifact-2/*
status: {}
`, output)
}

func nominalIntegration(name string) (v1.Integration, v1.IntegrationKit) {
	it := v1.NewIntegration("default", name)
	it.Status.Phase = v1.IntegrationPhaseRunning
	it.Status.Image = "my-special-image"
	ik := v1.NewIntegrationKit("default", name+"-kit")
	ik.Status = v1.IntegrationKitStatus{
		Artifacts: []v1.Artifact{
			{Target: "/path/to/artifact-1/a-1.jar"},
			{Target: "/path/to/artifact-2/a-2.jar"},
		},
		RuntimeVersion: "1.2.3",
	}
	it.Status.IntegrationKit = &corev1.ObjectReference{
		Namespace: ik.Namespace,
		Name:      ik.Name,
		Kind:      ik.Kind,
	}
	return it, *ik
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
	defaultKB := nominalPipe("my-pipe-test")
	defaultIntegration, defaultKit := nominalIntegration("my-pipe-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	promoteCmdOptions, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultKB, &defaultIntegration, &defaultKit, &srcCatalog, &dstCatalog)
	output, err := ExecuteCommand(promoteCmd, cmdPromote, "my-pipe-test", "--to", "prod-namespace", "-o", "yaml", "-n", "default")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	require.NoError(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  annotations:
    trait.camel.apache.org/camel.runtime-version: 1.2.3
    trait.camel.apache.org/container.image: my-special-image
    trait.camel.apache.org/jvm.classpath: /path/to/artifact-1/*:/path/to/artifact-2/*
  creationTimestamp: null
  name: my-pipe-test
  namespace: prod-namespace
spec:
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
	defaultIntegration, defaultKit := nominalIntegration("my-it-test")
	defaultIntegration.Annotations = map[string]string{
		"camel.apache.org/operator.id": "camel-k",
		"my-annotation":                "my-value",
	}
	defaultIntegration.Labels = map[string]string{
		"my-label": "my-value",
	}
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	promoteCmdOptions, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &defaultKit, &srcCatalog, &dstCatalog)
	output, err := ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "--to", "prod-namespace", "-o", "yaml", "-n", "default")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	require.NoError(t, err)
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
    camel:
      runtimeVersion: 1.2.3
    container:
      image: my-special-image
    jvm:
      classpath: /path/to/artifact-1/*:/path/to/artifact-2/*
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
	defaultKB := nominalPipe("my-pipe-test")
	defaultKB.Annotations = map[string]string{
		"camel.apache.org/operator.id": "camel-k",
		"my-annotation":                "my-value",
	}
	defaultKB.Labels = map[string]string{
		"my-label": "my-value",
	}
	defaultIntegration, defaultKit := nominalIntegration("my-pipe-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	promoteCmdOptions, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultKB, &defaultIntegration, &defaultKit, &srcCatalog, &dstCatalog)
	output, err := ExecuteCommand(promoteCmd, cmdPromote, "my-pipe-test", "--to", "prod-namespace", "-o", "yaml", "-n", "default")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	require.NoError(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  annotations:
    my-annotation: my-value
    trait.camel.apache.org/camel.runtime-version: 1.2.3
    trait.camel.apache.org/container.image: my-special-image
    trait.camel.apache.org/jvm.classpath: /path/to/artifact-1/*:/path/to/artifact-2/*
  creationTimestamp: null
  labels:
    my-label: my-value
  name: my-pipe-test
  namespace: prod-namespace
spec:
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
	defaultIntegration, defaultKit := nominalIntegration("my-it-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	_, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &defaultKit, &srcCatalog, &dstCatalog)
	output, err := ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "--to", "prod-namespace", "-i", "-n", "default")
	require.NoError(t, err)
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
	defaultKB := nominalPipe("my-pipe-test")
	defaultIntegration, defaultKit := nominalIntegration("my-pipe-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	_, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultKB, &defaultIntegration, &defaultKit, &srcCatalog, &dstCatalog)
	output, err := ExecuteCommand(promoteCmd, cmdPromote, "my-pipe-test", "--to", "prod-namespace", "-i", "-n", "default")
	require.NoError(t, err)
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
	defaultIntegration, defaultKit := nominalIntegration("my-it-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	// Verify default (missing) operator Id
	promoteCmdOptions, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &defaultKit, &srcCatalog, &dstCatalog)
	output, err := ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "-x", "my-prod-operator", "-o", "yaml", "--to", "prod")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	require.NoError(t, err)
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
    camel:
      runtimeVersion: 1.2.3
    container:
      image: my-special-image
    jvm:
      classpath: /path/to/artifact-1/*:/path/to/artifact-2/*
status: {}
`, output)
	// Verify also when the operator Id is set in the integration
	defaultIntegration.Annotations = map[string]string{
		"camel.apache.org/operator.id": "camel-k",
	}
	promoteCmdOptions, promoteCmd, _ = initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &defaultKit, &srcCatalog, &dstCatalog)
	output, err = ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "-x", "my-prod-operator", "-o", "yaml", "--to", "prod")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	require.NoError(t, err)
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
    camel:
      runtimeVersion: 1.2.3
    container:
      image: my-special-image
    jvm:
      classpath: /path/to/artifact-1/*:/path/to/artifact-2/*
status: {}
`, output)
}

func TestIntegrationWithSavedTraitsDryRun(t *testing.T) {
	srcPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	srcPlatform.Status.Version = defaults.Version
	srcPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	srcPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	dstPlatform := v1.NewIntegrationPlatform("prod-namespace", platform.DefaultPlatformName)
	dstPlatform.Status.Version = defaults.Version
	dstPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	dstPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	defaultIntegration, defaultKit := nominalIntegration("my-it-test")
	defaultIntegration.Status.Traits = &v1.Traits{
		Service: &trait.ServiceTrait{
			Trait: trait.Trait{
				Enabled: ptr.To(true),
			},
		},
	}
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	promoteCmdOptions, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &defaultKit, &srcCatalog, &dstCatalog)
	output, err := ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "--to", "prod-namespace", "-o", "yaml", "-n", "default")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	require.NoError(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Integration
metadata:
  creationTimestamp: null
  name: my-it-test
  namespace: prod-namespace
spec:
  traits:
    camel:
      runtimeVersion: 1.2.3
    container:
      image: my-special-image
    jvm:
      classpath: /path/to/artifact-1/*:/path/to/artifact-2/*
    service:
      enabled: true
status: {}
`, output)
}

func TestPipeWithSavedTraitsDryRun(t *testing.T) {
	srcPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	srcPlatform.Status.Version = defaults.Version
	srcPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	srcPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	dstPlatform := v1.NewIntegrationPlatform("prod-namespace", platform.DefaultPlatformName)
	dstPlatform.Status.Version = defaults.Version
	dstPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	dstPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	defaultKB := nominalPipe("my-pipe-test")
	defaultKB.Annotations = map[string]string{
		"camel.apache.org/operator.id": "camel-k",
		"my-annotation":                "my-value",
	}
	defaultKB.Labels = map[string]string{
		"my-label": "my-value",
	}
	defaultIntegration, defaultKit := nominalIntegration("my-pipe-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	promoteCmdOptions, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultKB, &defaultIntegration, &defaultKit, &srcCatalog, &dstCatalog)
	output, err := ExecuteCommand(promoteCmd, cmdPromote, "my-pipe-test", "--to", "prod-namespace", "-o", "yaml", "-n", "default")
	assert.Equal(t, "yaml", promoteCmdOptions.OutputFormat)
	require.NoError(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  annotations:
    my-annotation: my-value
    trait.camel.apache.org/camel.runtime-version: 1.2.3
    trait.camel.apache.org/container.image: my-special-image
    trait.camel.apache.org/jvm.classpath: /path/to/artifact-1/*:/path/to/artifact-2/*
  creationTimestamp: null
  labels:
    my-label: my-value
  name: my-pipe-test
  namespace: prod-namespace
spec:
  sink: {}
  source: {}
status: {}
`, output)
}

const expectedGitOpsIt = `apiVersion: camel.apache.org/v1
kind: Integration
metadata:
  creationTimestamp: null
  name: my-it-test
spec:
  traits:
    affinity:
      nodeAffinityLabels:
      - my-node
    camel:
      properties:
      - my.property=val
      runtimeVersion: 1.2.3
    container:
      image: my-special-image
      imagePullPolicy: Always
      limitCPU: "1"
      limitMemory: 1024Mi
      port: 2000
      requestCPU: "0.5"
      requestMemory: 512Mi
    environment:
      vars:
      - MY_VAR=val
    jvm:
      classpath: /path/to/artifact-1/*:/path/to/artifact-2/*
      jar: my.jar
      options:
      - -XMX 123
    mount:
      configs:
      - configmap:my-cm
      - secret:my-sec
    service:
      annotations:
        my-annotation: "123"
      auto: false
      enabled: true
    toleration:
      taints:
      - taint1:true
status: {}
`

const expectedGitOpsItPatch = `apiVersion: camel.apache.org/v1
kind: Integration
metadata:
  creationTimestamp: null
  name: my-it-test
spec:
  traits:
    affinity:
      nodeAffinityLabels:
      - my-node
    camel:
      properties:
      - my.property=val
    container:
      limitCPU: "1"
      limitMemory: 1024Mi
      requestCPU: "0.5"
      requestMemory: 512Mi
    environment:
      vars:
      - MY_VAR=val
    jvm:
      options:
      - -XMX 123
    mount:
      configs:
      - configmap:my-cm
      - secret:my-sec
    toleration:
      taints:
      - taint1:true
status: {}
`

func TestIntegrationGitOps(t *testing.T) {
	promoteCmd := prepareMyIntegrationTestPromoteCmd(t)
	tmpDir := t.TempDir()

	output, err := ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "--to", "prod-namespace", "--export-gitops-dir", tmpDir, "-n", "default")
	require.NoError(t, err)
	assert.Contains(t, output, `Exported a Kustomize based Gitops directory`)

	baseIt, err := os.ReadFile(filepath.Join(tmpDir, "my-it-test", "base", "integration.yaml"))
	require.NoError(t, err)
	assert.Equal(t, expectedGitOpsIt, string(baseIt))

	patchIt, err := os.ReadFile(filepath.Join(tmpDir, "my-it-test", "overlays", "prod-namespace", "patch-integration.yaml"))
	require.NoError(t, err)
	assert.Equal(t, expectedGitOpsItPatch, string(patchIt))
}

const expectedGitOpsPipe = `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  annotations:
    my-annotation: my-value
    trait.camel.apache.org/affinity.node-affinity-labels: '[node1,node2]'
    trait.camel.apache.org/camel.properties: '[a=1]'
    trait.camel.apache.org/camel.runtime-version: 1.2.3
    trait.camel.apache.org/container.image: my-special-image
    trait.camel.apache.org/container.image-pull-policy: Always
    trait.camel.apache.org/container.limit-cpu: "2"
    trait.camel.apache.org/container.limit-memory: 1024Mi
    trait.camel.apache.org/container.request-cpu: "1"
    trait.camel.apache.org/container.request-memory: 2048Mi
    trait.camel.apache.org/environment.vars: '[MYVAR=1]'
    trait.camel.apache.org/jvm.classpath: /path/to/artifact-1/*:/path/to/artifact-2/*
    trait.camel.apache.org/jvm.jar: my.jar
    trait.camel.apache.org/jvm.options: '[-XMX 123]'
    trait.camel.apache.org/mount.resources: '[configmap:my-cm,secret:my-sec/my-key@/tmp/file.txt]'
    trait.camel.apache.org/service.auto: "false"
    trait.camel.apache.org/toleration.taints: '[mytaints:true]'
  creationTimestamp: null
  labels:
    my-label: my-value
  name: my-pipe-test
spec:
  sink: {}
  source: {}
status: {}
`

const expectedGitOpsPipePatch = `apiVersion: camel.apache.org/v1
kind: Pipe
metadata:
  annotations:
    my-annotation: my-value
    trait.camel.apache.org/affinity.node-affinity-labels: '[node1,node2]'
    trait.camel.apache.org/camel.properties: '[a=1]'
    trait.camel.apache.org/container.limit-cpu: "2"
    trait.camel.apache.org/container.limit-memory: 1024Mi
    trait.camel.apache.org/container.request-cpu: "1"
    trait.camel.apache.org/container.request-memory: 2048Mi
    trait.camel.apache.org/environment.vars: '[MYVAR=1]'
    trait.camel.apache.org/jvm.options: '[-XMX 123]'
    trait.camel.apache.org/mount.resources: '[configmap:my-cm,secret:my-sec/my-key@/tmp/file.txt]'
    trait.camel.apache.org/toleration.taints: '[mytaints:true]'
  creationTimestamp: null
  name: my-pipe-test
spec:
  sink: {}
  source: {}
status: {}
`

func TestPipeGitOps(t *testing.T) {
	promoteCmd := prepareMyPipeTestPromoteCmd(t)
	tmpDir := t.TempDir()

	output, err := ExecuteCommand(promoteCmd, cmdPromote, "my-pipe-test", "--to", "prod-namespace", "--export-gitops-dir", tmpDir, "-n", "default")
	require.NoError(t, err)
	assert.Contains(t, output, `Exported a Kustomize based Gitops directory`)

	baseIt, err := os.ReadFile(filepath.Join(tmpDir, "my-pipe-test", "base", "pipe.yaml"))
	require.NoError(t, err)
	assert.Equal(t, expectedGitOpsPipe, string(baseIt))

	patchPipe, err := os.ReadFile(filepath.Join(tmpDir, "my-pipe-test", "overlays", "prod-namespace", "patch-pipe.yaml"))
	require.NoError(t, err)
	assert.Equal(t, expectedGitOpsPipePatch, string(patchPipe))
}

func prepareMyPipeTestPromoteCmd(t *testing.T) *cobra.Command {
	srcPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	srcPlatform.Status.Version = defaults.Version
	srcPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	srcPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	dstPlatform := v1.NewIntegrationPlatform("prod-namespace", platform.DefaultPlatformName)
	dstPlatform.Status.Version = defaults.Version
	dstPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	dstPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	defaultKB := nominalPipe("my-pipe-test")
	defaultKB.Annotations = map[string]string{
		"camel.apache.org/operator.id": "camel-k",
		"my-annotation":                "my-value",
		v1.TraitAnnotationPrefix + "affinity.node-affinity-labels": "[node1,node2]",
		v1.TraitAnnotationPrefix + "camel.properties":              "[a=1]",
		v1.TraitAnnotationPrefix + "container.limit-cpu":           "2",
		v1.TraitAnnotationPrefix + "container.limit-memory":        "1024Mi",
		v1.TraitAnnotationPrefix + "container.request-cpu":         "1",
		v1.TraitAnnotationPrefix + "container.request-memory":      "2048Mi",
		v1.TraitAnnotationPrefix + "container.image-pull-policy":   "Always",
		v1.TraitAnnotationPrefix + "environment.vars":              "[MYVAR=1]",
		v1.TraitAnnotationPrefix + "jvm.options":                   "[-XMX 123]",
		v1.TraitAnnotationPrefix + "jvm.jar":                       "my.jar",
		v1.TraitAnnotationPrefix + "mount.resources":               "[configmap:my-cm,secret:my-sec/my-key@/tmp/file.txt]",
		v1.TraitAnnotationPrefix + "service.auto":                  "false",
		v1.TraitAnnotationPrefix + "toleration.taints":             "[mytaints:true]",
	}
	defaultKB.Labels = map[string]string{
		"my-label": "my-value",
	}
	defaultIntegration, defaultKit := nominalIntegration("my-pipe-test")
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	_, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultKB, &defaultIntegration, &defaultKit, &srcCatalog, &dstCatalog)

	return promoteCmd
}

func prepareMyIntegrationTestPromoteCmd(t *testing.T) *cobra.Command {
	srcPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	srcPlatform.Status.Version = defaults.Version
	srcPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	srcPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	dstPlatform := v1.NewIntegrationPlatform("prod-namespace", platform.DefaultPlatformName)
	dstPlatform.Status.Version = defaults.Version
	dstPlatform.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	dstPlatform.Status.Phase = v1.IntegrationPlatformPhaseReady
	defaultIntegration, defaultKit := nominalIntegration("my-it-test")
	defaultIntegration.Status.Traits = &v1.Traits{
		Affinity: &trait.AffinityTrait{
			NodeAffinityLabels: []string{"my-node"},
		},
		Camel: &trait.CamelTrait{
			Properties: []string{"my.property=val"},
		},
		Container: &trait.ContainerTrait{
			LimitCPU:        "1",
			LimitMemory:     "1024Mi",
			RequestCPU:      "0.5",
			RequestMemory:   "512Mi",
			Port:            2000,
			ImagePullPolicy: corev1.PullAlways,
		},
		Environment: &trait.EnvironmentTrait{
			Vars: []string{"MY_VAR=val"},
		},
		JVM: &trait.JVMTrait{
			Jar:     "my.jar",
			Options: []string{"-XMX 123"},
		},
		Mount: &trait.MountTrait{
			Configs: []string{"configmap:my-cm", "secret:my-sec"},
		},
		Service: &trait.ServiceTrait{
			Trait: trait.Trait{
				Enabled: ptr.To(true),
			},
			Auto: ptr.To(false),
			Annotations: map[string]string{
				"my-annotation": "123",
			},
		},
		Toleration: &trait.TolerationTrait{
			Taints: []string{"taint1:true"},
		},
	}
	srcCatalog := createTestCamelCatalog(srcPlatform)
	dstCatalog := createTestCamelCatalog(dstPlatform)

	_, promoteCmd, _ := initializePromoteCmdOptions(t, &srcPlatform, &dstPlatform, &defaultIntegration, &defaultKit, &srcCatalog, &dstCatalog)

	return promoteCmd
}

func TestGitHubUrlParsing(t *testing.T) {
	gitRepo, err := git.PlainInit(t.TempDir(), true)
	assert.NoError(t, err)
	remote, err := gitRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{
			"git@github.com:apache/camel-k.git",
		},
	})
	assert.NoError(t, err)
	owner, repo, err := parseGitHubURL(remote)
	assert.NoError(t, err)
	assert.Equal(t, "apache", owner)
	assert.Equal(t, "camel-k", repo)
	remote, err = gitRepo.CreateRemote(&config.RemoteConfig{
		Name: "fork",
		URLs: []string{
			"https://github.com/squakez/camel-k.git",
		},
	})
	assert.NoError(t, err)
	owner, repo, err = parseGitHubURL(remote)
	assert.NoError(t, err)
	assert.Equal(t, "squakez", owner)
	assert.Equal(t, "camel-k", repo)
	remote, err = gitRepo.CreateRemote(&config.RemoteConfig{
		Name: "invalid",
		URLs: []string{
			"https://gitlab.com/squakez/camel-k.git",
		},
	})
	assert.NoError(t, err)
	_, _, err = parseGitHubURL(remote)
	assert.Error(t, err)
}

func TestIntegrationGitOpsWithPush(t *testing.T) {
	promoteCmd := prepareMyIntegrationTestPromoteCmd(t)
	tmpDir := t.TempDir()

	repo, err := git.PlainInit(tmpDir, false)
	require.NoError(t, err)
	_, err = repo.Head()
	// here we test default git branch without initial commit (empty git repo)
	require.Error(t, err, "HEAD reference should not be found")

	runCmd := func() {
		output, err := ExecuteCommand(promoteCmd, cmdPromote, "my-it-test", "--to", "prod-namespace",
			"--export-gitops-dir", tmpDir, "-n", "default", "--push-gitops-dir")
		require.NoError(t, err)
		assert.Contains(t, output, `Exported a Kustomize based Gitops directory`)
	}

	assertGitOpsExportWithPush(t, "my-it-test", "integration.yaml", "patch-integration.yaml", tmpDir, repo, runCmd)
}

func TestPipeGitOpsWithPush(t *testing.T) {
	promoteCmd := prepareMyPipeTestPromoteCmd(t)
	tmpDir := t.TempDir()

	// create local and remote repository
	remoteRepoDir := t.TempDir()
	remoteRepo, err := git.PlainInit(remoteRepoDir, true)
	require.NoError(t, err)
	localRepo, err := git.PlainInit(tmpDir, false)
	require.NoError(t, err)
	_, err = localRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteRepoDir},
	})
	require.NoError(t, err)

	// here we test custom branch, so create and push custom initial commit
	workTree, err := localRepo.Worktree()
	require.NoError(t, err)
	_, err = workTree.Commit("initial commit", &git.CommitOptions{
		AllowEmptyCommits: true,
		Author: &object.Signature{
			Name: "Camel K",
		},
	})
	require.NoError(t, err)
	err = localRepo.Push(&git.PushOptions{})
	require.NoError(t, err)
	const customBranch = "my-custom-branch"
	err = workTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(customBranch),
		Create: true,
	})
	require.NoError(t, err)

	runCmd := func() {
		output, err := ExecuteCommand(promoteCmd, cmdPromote, "my-pipe-test", "--to", "prod-namespace",
			"--export-gitops-dir", tmpDir, "-n", "default", "--push-gitops-dir")
		require.NoError(t, err)
		assert.Contains(t, output, `Exported a Kustomize based Gitops directory`)
	}

	assertGitOpsExportWithPush(t, "my-pipe-test", "pipe.yaml", "patch-pipe.yaml", tmpDir, localRepo, runCmd)

	// verify that git commit was pushed to remote
	foundBranches := make([]plumbing.ReferenceName, 0)
	refs, err := remoteRepo.References()
	require.NoError(t, err)
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			branchName := ref.Name().Short()
			if strings.HasPrefix(branchName, "camel-k-gitops-export-") {
				foundBranches = append(foundBranches, ref.Target())
			}
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, len(foundBranches))
	commits, err := remoteRepo.CommitObjects()
	require.NoError(t, err)
	foundCommits := make([]string, 0)
	err = commits.ForEach(func(commit *object.Commit) error {
		if strings.Contains(commit.Message, "GitOps base overlay") {
			foundCommits = append(foundCommits, commit.Message)
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, len(foundCommits))
}

func assertGitOpsExportWithPush(t *testing.T, projectName, baseOverlayFileName, otherOverlayFileName,
	tmpDir string, repo *git.Repository, runCmd func()) {
	runCmd()

	// check expected files generated and the git checkout doesn't delete them
	_, err := os.Stat(filepath.Join(tmpDir, projectName, "base", baseOverlayFileName))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tmpDir, projectName, "base", "kustomization.yaml"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tmpDir, projectName, "overlays", "prod-namespace", otherOverlayFileName))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tmpDir, projectName, "overlays", "prod-namespace", "kustomization.yaml"))
	require.NoError(t, err)

	// check git commit
	headAfter, err := repo.Head()
	require.NoError(t, err)
	assert.NotNil(t, headAfter)
	assert.NotEmpty(t, headAfter.Hash())
	commit, err := repo.CommitObject(headAfter.Hash())
	require.NoError(t, err)
	commitStats, err := commit.Stats()
	require.NoError(t, err)
	requiredFiles := []string{filepath.Join(projectName, baseOverlayDirName, "kustomization.yaml"),
		filepath.Join(projectName, baseOverlayDirName, baseOverlayFileName)}
	for _, fileStat := range commitStats {
		assert.Contains(t, requiredFiles, fileStat.Name)
	}
	assert.Contains(t, commit.Message, "add GitOps base overlay")

	// check created branch
	currentHead, err := repo.Head()
	require.NoError(t, err)
	assert.Contains(t, currentHead.Name().Short(), "camel-k-gitops-export-")

	// check that running command that generates the same files as are already in the git repo won't result in an error
	runCmd()

	// make some changes so that we can test git repository update
	baseKustomizationPath := filepath.Join(tmpDir, projectName, baseOverlayDirName, "kustomization.yaml")
	content, err := os.ReadFile(baseKustomizationPath)
	require.NoError(t, err)
	newContent := strings.ReplaceAll(string(content), baseOverlayFileName, "whatever")
	err = os.WriteFile(baseKustomizationPath, []byte(newContent), io.FilePerm755)
	require.NoError(t, err)
	workTree, err := repo.Worktree()
	require.NoError(t, err)
	_, err = workTree.Add(filepath.Join(projectName, baseOverlayDirName, "kustomization.yaml"))
	require.NoError(t, err)
	_, err = workTree.Commit("Update kustomization.yaml", &git.CommitOptions{
		AllowEmptyCommits: false,
		Author: &object.Signature{
			Name: "Camel K",
		},
	})
	require.NoError(t, err)

	// test update
	runCmd()
	headAfter, err = repo.Head()
	require.NoError(t, err)
	assert.NotNil(t, headAfter)
	assert.NotEmpty(t, headAfter.Hash())
	commit, err = repo.CommitObject(headAfter.Hash())
	require.NoError(t, err)
	commitStats, err = commit.Stats()
	require.NoError(t, err)
	for _, fileStat := range commitStats {
		assert.Contains(t, filepath.Join(projectName, baseOverlayDirName, "kustomization.yaml"), fileStat.Name)
	}
	assert.Contains(t, commit.Message, "update GitOps base overlay")
}

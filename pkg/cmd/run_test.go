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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"k8s.io/utils/ptr"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	cmdRun            = "run"
	integrationSource = "example.js"
	yamlIntegration   = `# camel-k: language=yaml

- from:
    uri: "timer:yaml"
    parameters:
      period: "1000"
    steps:
      - setBody:
          constant: "Hello Camel K from yaml"
      - to: "log:info"
`
)

// nolint: unparam
func initializeRunCmdOptions(t *testing.T) (*runCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	options, rootCmd := kamelTestPreAddCommandInit()
	runCmdOptions := addTestRunCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd, options)

	return runCmdOptions, rootCmd, *options
}

// nolint: unparam
func initializeRunCmdOptionsWithOutput(t *testing.T) (*runCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()
	defaultIntegrationPlatform := v1.NewIntegrationPlatform("default", platform.DefaultPlatformName)
	c := v1.NewCamelCatalog(defaultIntegrationPlatform.Namespace, defaults.DefaultRuntimeVersion)
	c.Spec = v1.CamelCatalogSpec{Runtime: v1.RuntimeSpec{Provider: defaultIntegrationPlatform.Status.Build.RuntimeProvider, Version: defaultIntegrationPlatform.Status.Build.RuntimeVersion}}
	fakeClient, _ := internal.NewFakeClient(&defaultIntegrationPlatform, &c)

	options, rootCmd := kamelTestPreAddCommandInitWithClient(fakeClient)
	runCmdOptions := addTestRunCmdWithOutput(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd, options)

	return runCmdOptions, rootCmd, *options
}

func addTestRunCmd(options RootCmdOptions, rootCmd *cobra.Command) *runCmdOptions {
	// add a testing version of run Command
	runCmd, runOptions := newCmdRun(&options)
	runCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	runCmd.PostRunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	runCmd.Args = ArbitraryArgs
	rootCmd.AddCommand(runCmd)
	return runOptions
}

func addTestRunCmdWithOutput(options RootCmdOptions, rootCmd *cobra.Command) *runCmdOptions {
	// add a testing version of run Command with output
	runCmd, runOptions := newCmdRun(&options)
	runCmd.Args = ArbitraryArgs
	rootCmd.AddCommand(runCmd)
	return runOptions
}

func TestRunNoFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, integrationSource)
	require.NoError(t, err)
	// Check default expected values
	assert.False(t, runCmdOptions.Wait)
	assert.False(t, runCmdOptions.Logs)
	assert.False(t, runCmdOptions.Sync)
	assert.False(t, runCmdOptions.Dev)
	assert.True(t, runCmdOptions.UseFlows)
	assert.False(t, runCmdOptions.Compression)
	assert.False(t, runCmdOptions.Save)
}

func TestRunNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "--nonExistingFlag", integrationSource)
	require.Error(t, err)
}

func TestRunCompressionFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "--compression", integrationSource)
	require.NoError(t, err)
	assert.True(t, runCmdOptions.Compression)
}

func TestRunDependencyFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun,
		"--dependency", "dependency1",
		"--dependency", "dependency2",
		"--dependency", "dependency3",
		integrationSource)
	require.NoError(t, err)
	assert.Len(t, runCmdOptions.Dependencies, 3)
	assert.Equal(t, "dependency1", runCmdOptions.Dependencies[0])
	assert.Equal(t, "dependency2", runCmdOptions.Dependencies[1])
	assert.Equal(t, "dependency3", runCmdOptions.Dependencies[2])
}

func TestRunDevFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "--dev", integrationSource)
	require.NoError(t, err)
	assert.True(t, runCmdOptions.Dev)
}

func TestRunDevModeOutputFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "--dev", "-o", "yaml", integrationSource)
	assert.True(t, runCmdOptions.Dev)
	assert.Equal(t, "yaml", runCmdOptions.OutputFormat)
	require.Error(t, err)
	assert.Equal(t, "cannot use --dev with -o/--output option",
		err.Error())
}

func TestRunEnvFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun,
		"--env", "env1",
		"--env", "env2",
		integrationSource)
	require.NoError(t, err)
	assert.Len(t, runCmdOptions.EnvVars, 2)
	assert.Equal(t, "env1", runCmdOptions.EnvVars[0])
	assert.Equal(t, "env2", runCmdOptions.EnvVars[1])
}

func TestRunKitFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "--kit", "myKit", integrationSource)
	require.NoError(t, err)
	assert.Equal(t, "myKit", runCmdOptions.IntegrationKit)
}

func TestRunLabelFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun,
		"--label", "label1=l1",
		"--label", "label2=l2",
		"--label", "label3=l3",
		integrationSource)
	require.NoError(t, err)
	assert.Len(t, runCmdOptions.Labels, 3)
	assert.Equal(t, "label1=l1", runCmdOptions.Labels[0])
	assert.Equal(t, "label2=l2", runCmdOptions.Labels[1])
	assert.Equal(t, "label3=l3", runCmdOptions.Labels[2])
}

func TestRunLabelWrongFormatFlag(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "--label", "label1", integrationSource)
	require.Error(t, err)
}

func TestRunLogsFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "--logs", integrationSource)
	require.NoError(t, err)
	assert.True(t, runCmdOptions.Logs)
}

func TestRunMavenRepositoryFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun,
		"--maven-repository", "repo1",
		"--maven-repository", "repo2",
		"--maven-repository", "repo3",
		integrationSource)
	require.NoError(t, err)
	assert.Len(t, runCmdOptions.Repositories, 3)
	assert.Equal(t, "repo1", runCmdOptions.Repositories[0])
	assert.Equal(t, "repo2", runCmdOptions.Repositories[1])
	assert.Equal(t, "repo3", runCmdOptions.Repositories[2])
}

func TestRunNameFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "--name", "myIntegration", integrationSource)
	require.NoError(t, err)
	assert.Equal(t, "myIntegration", runCmdOptions.IntegrationName)
}

func TestRunOpenApiFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun,
		"--open-api", "configmap:oapi2",
		integrationSource)
	require.NoError(t, err)
	assert.Len(t, runCmdOptions.OpenAPIs, 1)
	assert.Equal(t, "configmap:oapi2", runCmdOptions.OpenAPIs[0])
}

func TestRunOpenApiInvalidFlag(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun,
		"--open-api", "secret:oapi1",
		"--open-api", "oapi2",
		integrationSource)
	require.Error(t, err)
}

func TestRunOutputFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "-o", "yaml", integrationSource)
	require.NoError(t, err)
	assert.Equal(t, "yaml", runCmdOptions.OutputFormat)
}

func TestRunProfileFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "--profile", "myProfile", integrationSource)
	require.NoError(t, err)
	assert.Equal(t, "myProfile", runCmdOptions.Profile)
}

func TestRunPropertyFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun,
		"--property", "property1",
		"--property", "property2",
		"--property", "property3",
		integrationSource)
	require.NoError(t, err)
	assert.Len(t, runCmdOptions.Properties, 3)
	assert.Equal(t, "property1", runCmdOptions.Properties[0])
	assert.Equal(t, "property2", runCmdOptions.Properties[1])
	assert.Equal(t, "property3", runCmdOptions.Properties[2])
}

const TestPropertyFileContent = `
a=b
# There's an issue in the properties lib: https://github.com/magiconair/properties/issues/59
# so the following cases have been commented. Btw, we don't use equal sign in keys
#c\=d=e
#d=c\=e
#ignore=me
f=g:h
i=j\nk
`

func TestExtractProperties_SingleKeyValue(t *testing.T) {
	correctValues := []string{"key=val", "key = val", "key= val", " key   =  val"}
	runCmdOptions, _, _ := initializeRunCmdOptionsWithOutput(t)
	for _, val := range correctValues {
		prop, err := runCmdOptions.extractProperties(nil, val)
		require.NoError(t, err)
		value, ok := prop.Get("key")
		assert.True(t, ok)
		assert.Equal(t, "val", value)
	}
}

func TestExtractProperties_FromFile(t *testing.T) {
	var tmpFile1 *os.File
	var err error
	if tmpFile1, err = os.CreateTemp("", "camel-k-*.properties"); err != nil {
		t.Error(err)
	}

	require.NoError(t, tmpFile1.Close())
	require.NoError(t, os.WriteFile(tmpFile1.Name(), []byte(`
	key=value
	#key2=value2
	my.key=value
	`), 0o400))
	runCmdOptions, _, _ := initializeRunCmdOptionsWithOutput(t)
	props, err := runCmdOptions.extractProperties(nil, "file:"+tmpFile1.Name())
	require.NoError(t, err)
	assert.Equal(t, 2, props.Len())
	for _, prop := range props.Keys() {
		value, ok := props.Get(prop)
		assert.True(t, ok)
		assert.Equal(t, "value", value)
	}
}

func TestExtractPropertiesFromFileAndSingleValue(t *testing.T) {
	var tmpFile1 *os.File
	var err error
	if tmpFile1, err = os.CreateTemp("", "camel-k-*.properties"); err != nil {
		t.Error(err)
	}

	require.NoError(t, tmpFile1.Close())
	require.NoError(t, os.WriteFile(tmpFile1.Name(), []byte(`
	key=value
	#key2=value2
	my.key=value
	`), 0o400))

	properties := []string{"key=override", "file:" + tmpFile1.Name(), "my.key = override"}
	runCmdOptions, _, _ := initializeRunCmdOptionsWithOutput(t)
	props, err := runCmdOptions.mergePropertiesWithPrecedence(nil, properties)
	require.NoError(t, err)
	assert.Equal(t, 2, props.Len())
	val, ok := props.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "override", val)
	val, ok = props.Get("my.key")
	assert.True(t, ok)
	assert.Equal(t, "override", val)
}

func TestAddPropertyFile(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = os.CreateTemp("", "camel-k-"); err != nil {
		t.Error(err)
	}

	require.NoError(t, tmpFile.Close())
	require.NoError(t, os.WriteFile(tmpFile.Name(), []byte(TestPropertyFileContent), 0o400))

	runCmdOptions, _, _ := initializeRunCmdOptionsWithOutput(t)
	properties, err := runCmdOptions.convertToTraitParameter(nil, "file:"+tmpFile.Name(), "trait.properties")
	require.NoError(t, err)
	assert.Equal(t, 3, len(properties))
	assert.Equal(t, `trait.properties=a = b`, properties[0])
	assert.Equal(t, `trait.properties=f = g:h`, properties[1])
	assert.Equal(t, `trait.properties=i = j\nk`, properties[2])
}

func TestRunProperty(t *testing.T) {
	runCmdOptions, _, _ := initializeRunCmdOptionsWithOutput(t)
	properties, err := runCmdOptions.convertToTraitParameter(nil, `key=value\nnewline`, "trait.properties")
	require.NoError(t, err)
	assert.Equal(t, 1, len(properties))
	assert.Equal(t, `trait.properties=key = value\nnewline`, properties[0])
}

func TestRunResourceFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun,
		"--resource", "res1",
		"--resource", "res2",
		integrationSource)
	require.NoError(t, err)
	assert.Len(t, runCmdOptions.Resources, 2)
	assert.Equal(t, "res1", runCmdOptions.Resources[0])
	assert.Equal(t, "res2", runCmdOptions.Resources[1])
}

func TestRunSaveFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "--save", integrationSource)
	require.NoError(t, err)
	assert.True(t, runCmdOptions.Save)
}

func TestRunSourceFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "--source", "source1", integrationSource)
	require.NoError(t, err)
	assert.Len(t, runCmdOptions.Sources, 1)
	assert.Equal(t, "source1", runCmdOptions.Sources[0])
}

func TestRunSyncFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "--sync", integrationSource)
	require.NoError(t, err)
	assert.True(t, runCmdOptions.Sync)
}

func TestRunExistingTraitFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun,
		"--trait", "jvm.enabled",
		"--trait", "logging.enabled",
		integrationSource)
	require.NoError(t, err)
	assert.Len(t, runCmdOptions.Traits, 2)
	assert.Equal(t, "jvm.enabled", runCmdOptions.Traits[0])
	assert.Equal(t, "logging.enabled", runCmdOptions.Traits[1])
}

func TestRunMissingTraitFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun,
		"--trait", "bogus.missing",
		integrationSource)
	require.Error(t, err)
	assert.Equal(t, "trait bogus does not exist in catalog", err.Error())
	assert.Len(t, runCmdOptions.Traits, 1)
	assert.Equal(t, "bogus.missing", runCmdOptions.Traits[0])
}

func TestConfigureTraits(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, "run",
		"--trait", "affinity.pod-affinity=false",
		"--trait", "environment.container-meta=false",
		"--trait", "prometheus.pod-monitor=false",
		"--trait", "telemetry.auto=true",
		"example.js")
	if err != nil {
		t.Error(err)
	}
	client, err := runCmdOptions.GetCmdClient()
	if err != nil {
		t.Error(err)
	}
	catalog := trait.NewCatalog(client)

	traits := v1.Traits{}
	err = trait.ConfigureTraits(runCmdOptions.Traits, &traits, catalog)

	require.NoError(t, err)
	traitMap, err := trait.ToTraitMap(traits)
	require.NoError(t, err)
	assert.Len(t, traits.Addons, 0)
	assert.Len(t, traitMap, 4)
	assertTraitConfiguration(t, traits.Affinity, &traitv1.AffinityTrait{PodAffinity: ptr.To(false)})
	assertTraitConfiguration(t, traits.Environment, &traitv1.EnvironmentTrait{ContainerMeta: ptr.To(false)})
	assertTraitConfiguration(t, traits.Prometheus, &traitv1.PrometheusTrait{PodMonitor: ptr.To(false)})
	assertTraitConfiguration(t, traits.Telemetry, &traitv1.TelemetryTrait{Auto: ptr.To(true)})
}

func assertTraitConfiguration(t *testing.T, trait interface{}, expected interface{}) {
	t.Helper()

	assert.NotNil(t, trait)
	assert.Equal(t, expected, trait)
}

func TestRunUseFlowsFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun, "--use-flows=false", integrationSource)
	require.NoError(t, err)
	assert.False(t, runCmdOptions.UseFlows)
}

func TestRunVolumeFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun,
		"-v", "pvcname:/container1/path",
		"-v", "pvcname:/container2/path",
		integrationSource)
	require.NoError(t, err)
	assert.Len(t, runCmdOptions.Volumes, 2)
	assert.Equal(t, "pvcname:/container1/path", runCmdOptions.Volumes[0])
	assert.Equal(t, "pvcname:/container2/path", runCmdOptions.Volumes[1])
}

func TestRunVolumeFlagWrongPVCFormat(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun,
		"-v", "pvcname",
		"-v", "pvcname/container2/path",
		integrationSource)
	require.Error(t, err)
}

func TestRunBuildPropertyFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := ExecuteCommand(rootCmd, cmdRun,
		"--build-property", "build-prop1=val1",
		"--build-property", "build-prop2=val2",
		integrationSource)
	require.NoError(t, err)
	assert.Len(t, runCmdOptions.BuildProperties, 2)
	assert.Equal(t, "build-prop1=val1", runCmdOptions.BuildProperties[0])
	assert.Equal(t, "build-prop2=val2", runCmdOptions.BuildProperties[1])
}

func TestRunValidateArgs(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)

	args := []string{"run_test.go"}
	err := runCmdOptions.validateArgs(rootCmd, args)
	require.NoError(t, err)

	args = []string{"missing_file"}
	err = runCmdOptions.validateArgs(rootCmd, args)
	require.Error(t, err)
	assert.Equal(t, "one of the provided sources is not reachable: missing file or unsupported scheme in missing_file", err.Error())
}

func TestResolvePodTemplate(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	templateText := `
containers:
  - name: integration
    env:
      - name: TEST
        value: TEST
    volumeMounts:
      - name: var-logs
        mountPath: /var/log
volumes:
  - name: var-logs
    emptyDir: { }
`

	integrationSpec := v1.IntegrationSpec{}
	err := resolvePodTemplate(context.TODO(), rootCmd, templateText, &integrationSpec)
	require.NoError(t, err)
	assert.NotNil(t, integrationSpec.PodTemplate)
	assert.Equal(t, 1, len(integrationSpec.PodTemplate.Spec.Containers))
	// assert.Equal(t, 1,len(integrationSpec.PodTemplate.Spec.Containers[0].VolumeMounts))
}

func TestResolveJsonPodTemplate(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	integrationSpec := v1.IntegrationSpec{}
	minifiedYamlTemplate := `{"containers": [{"name": "second"}, {"name": "integration", "env": [{"name": "CAMEL_K_DIGEST", "value": "new_value"}]}]}`

	err := resolvePodTemplate(context.TODO(), rootCmd, minifiedYamlTemplate, &integrationSpec)

	require.NoError(t, err)
	assert.NotNil(t, integrationSpec.PodTemplate)
	assert.Equal(t, 2, len(integrationSpec.PodTemplate.Spec.Containers))
}

func TestFilterBuildPropertyFiles(t *testing.T) {
	inputValues := []string{"file:/tmp/test", "key=val"}
	outputValues := filterBuildPropertyFiles(inputValues)

	assert.Len(t, outputValues, 1)
	assert.Equal(t, "/tmp/test", outputValues[0])
}

const TestSrcContent = `
import org.apache.camel.builder.RouteBuilder;

public class Sample extends RouteBuilder {
  @Override
  public void configure() throws Exception {
	  from("timer:tick")
        .log("Hello Camel K!");
  }
}
`

func TestOutputYaml(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = os.CreateTemp("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile.Close())
	require.NoError(t, os.WriteFile(tmpFile.Name(), []byte(TestSrcContent), 0o400))
	fileName := filepath.Base(tmpFile.Name())

	runCmdOptions, runCmd, _ := initializeRunCmdOptionsWithOutput(t)
	output, err := ExecuteCommand(runCmd, cmdRun, tmpFile.Name(), "-o", "yaml")
	assert.Equal(t, "yaml", runCmdOptions.OutputFormat)

	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(`apiVersion: camel.apache.org/v1
kind: Integration
metadata:
  annotations:
    camel.apache.org/operator.id: camel-k
  creationTimestamp: null
  name: %s
spec:
  sources:
  - content: "\nimport org.apache.camel.builder.RouteBuilder;\n\npublic class Sample
      extends RouteBuilder {\n  @Override\n  public void configure() throws Exception
      {\n\t  from(\"timer:tick\")\n        .log(\"Hello Camel K!\");\n  }\n}\n"
    name: %s
  traits: {}
status: {}
`, fileName, fileName), output)
}

func TestTrait(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = os.CreateTemp("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile.Close())
	require.NoError(t, os.WriteFile(tmpFile.Name(), []byte(TestSrcContent), 0o400))
	fileName := filepath.Base(tmpFile.Name())

	runCmdOptions, runCmd, _ := initializeRunCmdOptionsWithOutput(t)
	output, err := ExecuteCommand(runCmd, cmdRun, tmpFile.Name(), "-o", "yaml", "-t", "mount.configs=configmap:my-cm")
	assert.Equal(t, "yaml", runCmdOptions.OutputFormat)

	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(`apiVersion: camel.apache.org/v1
kind: Integration
metadata:
  annotations:
    camel.apache.org/operator.id: camel-k
  creationTimestamp: null
  name: %s
spec:
  sources:
  - content: "\nimport org.apache.camel.builder.RouteBuilder;\n\npublic class Sample
      extends RouteBuilder {\n  @Override\n  public void configure() throws Exception
      {\n\t  from(\"timer:tick\")\n        .log(\"Hello Camel K!\");\n  }\n}\n"
    name: %s
  traits:
    mount:
      configs:
      - configmap:my-cm
status: {}
`, fileName, fileName), output)
}

func TestMissingTrait(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = os.CreateTemp("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile.Close())
	require.NoError(t, os.WriteFile(tmpFile.Name(), []byte(TestSrcContent), 0o400))

	runCmdOptions, runCmd, _ := initializeRunCmdOptionsWithOutput(t)
	output, err := ExecuteCommand(runCmd, cmdRun, tmpFile.Name(), "-o", "yaml", "-t", "bogus.fail=i-must-fail")
	assert.Equal(t, "yaml", runCmdOptions.OutputFormat)
	assert.Equal(t, "Error: trait bogus does not exist in catalog\n", output)
	require.Error(t, err)
}

func TestResolveYamlPodTemplateWithSupplementalGroups(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	templateText := `
securityContext:
  supplementalGroups:
    - 666
`
	integrationSpec := v1.IntegrationSpec{}
	err := resolvePodTemplate(context.TODO(), rootCmd, templateText, &integrationSpec)
	require.NoError(t, err)
	assert.NotNil(t, integrationSpec.PodTemplate)
	assert.NotNil(t, integrationSpec.PodTemplate.Spec)
	assert.NotNil(t, integrationSpec.PodTemplate.Spec.SecurityContext)
	assert.NotNil(t, integrationSpec.PodTemplate.Spec.SecurityContext.SupplementalGroups)
	assert.Equal(t, 1, len(integrationSpec.PodTemplate.Spec.SecurityContext.SupplementalGroups))
	assert.Contains(t, integrationSpec.PodTemplate.Spec.SecurityContext.SupplementalGroups, int64(666))
}

func TestResolveJsonPodTemplateWithSupplementalGroups(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	minifiedYamlTemplate := `{"securityContext":{"supplementalGroups":[666]}}`

	integrationSpec := v1.IntegrationSpec{}
	err := resolvePodTemplate(context.TODO(), rootCmd, minifiedYamlTemplate, &integrationSpec)
	require.NoError(t, err)
	assert.NotNil(t, integrationSpec.PodTemplate)
	assert.NotNil(t, integrationSpec.PodTemplate.Spec)
	assert.NotNil(t, integrationSpec.PodTemplate.Spec.SecurityContext)
	assert.NotNil(t, integrationSpec.PodTemplate.Spec.SecurityContext.SupplementalGroups)
	assert.Equal(t, 1, len(integrationSpec.PodTemplate.Spec.SecurityContext.SupplementalGroups))
	assert.Contains(t, integrationSpec.PodTemplate.Spec.SecurityContext.SupplementalGroups, int64(666))
}

func TestIntegrationServiceAccountName(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = os.CreateTemp("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile.Close())
	require.NoError(t, os.WriteFile(tmpFile.Name(), []byte(TestSrcContent), 0o400))

	_, runCmd, _ := initializeRunCmdOptionsWithOutput(t)
	output, err := ExecuteCommand(runCmd, cmdRun, tmpFile.Name(), "-o", "yaml", "--service-account", "my-service-account")

	require.NoError(t, err)
	assert.Contains(t, output, "serviceAccountName: my-service-account")
}

func TestFileProperties(t *testing.T) {
	var tmpFile1 *os.File
	var err error
	if tmpFile1, err = os.CreateTemp("", "camel-k-*.properties"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile1.Close())
	require.NoError(t, os.WriteFile(tmpFile1.Name(), []byte(`
	key=${value}
	#key2=value2
	my.key=value
	`), 0o400))

	var tmpFile *os.File
	if tmpFile, err = os.CreateTemp("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile.Close())
	require.NoError(t, os.WriteFile(tmpFile.Name(), []byte(TestSrcContent), 0o400))
	_, runCmd, _ := initializeRunCmdOptionsWithOutput(t)
	output, err := ExecuteCommand(runCmd, cmdRun, tmpFile.Name(),
		"-p", "file:"+tmpFile1.Name(),
		"-o", "yaml",
	)
	require.NoError(t, err)
	assert.NotContains(t, output, "#key2")
	assert.Contains(t, output, "my.key = value")
	assert.Contains(t, output, "key = ${value}")
}

func TestPropertyShouldNotExpand(t *testing.T) {
	var tmpFile1 *os.File
	var err error
	if tmpFile1, err = os.CreateTemp("", "camel-k-*.properties"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile1.Close())
	require.NoError(t, os.WriteFile(tmpFile1.Name(), []byte(`
	key=${value}
	`), 0o400))

	var tmpFile *os.File
	if tmpFile, err = os.CreateTemp("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile.Close())
	require.NoError(t, os.WriteFile(tmpFile.Name(), []byte(TestSrcContent), 0o400))
	_, runCmd, _ := initializeRunCmdOptionsWithOutput(t)
	output, err := ExecuteCommand(runCmd, cmdRun, tmpFile.Name(),
		"-o", "yaml",
		"-p", "runtimeProp=${value}",
		"--build-property", "buildProp=${value}",
		"-p", "file:"+tmpFile1.Name(),
	)
	require.NoError(t, err)
	assert.Contains(t, output, "runtimeProp = ${value}")
	assert.Contains(t, output, "buildProp = ${value}")
	assert.Contains(t, output, "key = ${value}")
}

func TestRunOutput(t *testing.T) {
	var tmpFile1 *os.File
	var err error
	if tmpFile1, err = os.CreateTemp("", "camel-k-*.yaml"); err != nil {
		t.Error(err)
	}
	defer tmpFile1.Close()

	assert.Nil(t, tmpFile1.Sync())
	require.NoError(t, os.WriteFile(tmpFile1.Name(), []byte(yamlIntegration), 0o400))

	_, rootCmd, _ := initializeRunCmdOptionsWithOutput(t)
	output, err := ExecuteCommand(rootCmd, cmdRun, tmpFile1.Name())
	_, fileName := filepath.Split(tmpFile1.Name())
	integrationName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Integration \"%s\" created\n", integrationName), output)

	output, err = ExecuteCommand(rootCmd, cmdRun, tmpFile1.Name())
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Integration \"%s\" unchanged\n", integrationName), output)

	require.NoError(t, os.WriteFile(tmpFile1.Name(), []byte(strings.Replace(yamlIntegration, "Hello", "Hi", 1)), 0o400))
	assert.Nil(t, tmpFile1.Sync())
	output, err = ExecuteCommand(rootCmd, cmdRun, tmpFile1.Name())
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Integration \"%s\" updated\n", integrationName), output)
}

func TestRunGlob(t *testing.T) {
	dir, err := os.MkdirTemp("", "camel-k-TestRunGlob-*")
	if err != nil {
		t.Error(err)
	}

	pattern := "camel-k-*.yaml"

	tmpFile1, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Error(err)
	}
	defer tmpFile1.Close()
	assert.Nil(t, tmpFile1.Sync())
	require.NoError(t, os.WriteFile(tmpFile1.Name(), []byte(yamlIntegration), 0o400))

	tmpFile2, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Error(err)
	}
	defer tmpFile2.Close()
	assert.Nil(t, tmpFile2.Sync())
	require.NoError(t, os.WriteFile(tmpFile2.Name(), []byte(yamlIntegration), 0o400))

	integrationName := "myname"

	_, rootCmd, _ := initializeRunCmdOptionsWithOutput(t)

	file := fmt.Sprintf("%s%c%s*", dir, os.PathSeparator, "camel-k-*") // = dir/camel-k-*

	output, err := ExecuteCommand(rootCmd, cmdRun, "--name", integrationName, file)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Integration \"%s\" created\n", integrationName), output)
}

func TestRunGlobAllFiles(t *testing.T) {
	dir, err := os.MkdirTemp("", "camel-k-TestRunGlobAllFiles-*")
	if err != nil {
		t.Error(err)
	}

	pattern := "camel-k-*.yaml"

	tmpFile1, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Error(err)
	}
	defer tmpFile1.Close()
	assert.Nil(t, tmpFile1.Sync())
	require.NoError(t, os.WriteFile(tmpFile1.Name(), []byte(yamlIntegration), 0o400))

	tmpFile2, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Error(err)
	}
	defer tmpFile2.Close()
	assert.Nil(t, tmpFile2.Sync())
	require.NoError(t, os.WriteFile(tmpFile2.Name(), []byte(yamlIntegration), 0o400))

	integrationName := "myname"

	_, rootCmd, _ := initializeRunCmdOptionsWithOutput(t)

	file := fmt.Sprintf("%s%c*", dir, os.PathSeparator) // = dir/*

	output, err := ExecuteCommand(rootCmd, cmdRun, "--name", integrationName, file)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Integration \"%s\" created\n", integrationName), output)
}

func TestRunGlobChange(t *testing.T) {
	dir, err := os.MkdirTemp("", "camel-k-TestRunGlobChange-*")
	if err != nil {
		t.Error(err)
	}

	pattern := "camel-k-*.yaml"

	tmpFile1, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Error(err)
	}
	defer tmpFile1.Close()
	assert.Nil(t, tmpFile1.Sync())
	require.NoError(t, os.WriteFile(tmpFile1.Name(), []byte(yamlIntegration), 0o400))

	integrationName := "myname"

	_, rootCmd, _ := initializeRunCmdOptionsWithOutput(t)

	file := fmt.Sprintf("%s%c%s", dir, os.PathSeparator, "camel-k-*")

	output, err := ExecuteCommand(rootCmd, cmdRun, "--name", integrationName, file)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Integration \"%s\" created\n", integrationName), output)

	output, err = ExecuteCommand(rootCmd, cmdRun, "--name", integrationName, file)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Integration \"%s\" unchanged\n", integrationName), output)

	tmpFile2, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Error(err)
	}
	defer tmpFile2.Close()
	assert.Nil(t, tmpFile2.Sync())
	require.NoError(t, os.WriteFile(tmpFile2.Name(), []byte(yamlIntegration), 0o400))

	output, err = ExecuteCommand(rootCmd, cmdRun, "--name", integrationName, file)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Integration \"%s\" updated\n", integrationName), output)
}

func TestRunOutputWithoutKubernetesCluster(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "camel-k-kubeconfig-*")
	require.NoError(t, err)

	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	runCmdOptions._client = nil // remove the default fake client which can bypass this test
	runCmdOptions.KubeConfig = tmpFile.Name()
	_, err = ExecuteCommand(rootCmd, cmdRun, "-o", "yaml", integrationSource)
	require.NoError(t, err)
}

func TestSelfManagedBuildIntegration(t *testing.T) {
	runCmdOptions, runCmd, _ := initializeRunCmdOptionsWithOutput(t)
	output, err := ExecuteCommand(runCmd, cmdRun, "--image", "docker.io/my-org/my-app:1.0.0", "-o", "yaml", "-t", "mount.configs=configmap:my-cm")
	assert.Equal(t, "yaml", runCmdOptions.OutputFormat)

	require.NoError(t, err)
	assert.Equal(t, `apiVersion: camel.apache.org/v1
kind: Integration
metadata:
  annotations:
    camel.apache.org/operator.id: camel-k
  creationTimestamp: null
  name: my-app-v1
spec:
  traits:
    container:
      image: docker.io/my-org/my-app:1.0.0
    mount:
      configs:
      - configmap:my-cm
status: {}
`, output)
}

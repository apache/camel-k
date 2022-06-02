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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const (
	cmdRun            = "run"
	integrationSource = "example.js"
)

// nolint: unparam
func initializeRunCmdOptions(t *testing.T) (*runCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	options, rootCmd := kamelTestPreAddCommandInit()
	runCmdOptions := addTestRunCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	return runCmdOptions, rootCmd, *options
}

// nolint: unparam
func initializeRunCmdOptionsWithOutput(t *testing.T) (*runCmdOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	options, rootCmd := kamelTestPreAddCommandInit()
	runCmdOptions := addTestRunCmdWithOutput(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

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
	runCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(runCmd)
	return runOptions
}

func addTestRunCmdWithOutput(options RootCmdOptions, rootCmd *cobra.Command) *runCmdOptions {
	// add a testing version of run Command with output
	runCmd, runOptions := newCmdRun(&options)
	runCmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	runCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(runCmd)
	return runOptions
}

func TestRunNoFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, integrationSource)
	assert.Nil(t, err)
	// Check default expected values
	assert.Equal(t, false, runCmdOptions.Wait)
	assert.Equal(t, false, runCmdOptions.Logs)
	assert.Equal(t, false, runCmdOptions.Sync)
	assert.Equal(t, false, runCmdOptions.Dev)
	assert.Equal(t, true, runCmdOptions.UseFlows)
	assert.Equal(t, false, runCmdOptions.Compression)
	assert.Equal(t, false, runCmdOptions.Save)
}

func TestRunNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "--nonExistingFlag", integrationSource)
	assert.NotNil(t, err)
}

func TestRunCompressionFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "--compression", integrationSource)
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.Compression)
}

func TestRunDependencyFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun,
		"--dependency", "dependency1",
		"--dependency", "dependency2",
		"--dependency", "dependency3",
		integrationSource)
	assert.Nil(t, err)
	assert.Len(t, runCmdOptions.Dependencies, 3)
	assert.Equal(t, "dependency1", runCmdOptions.Dependencies[0])
	assert.Equal(t, "dependency2", runCmdOptions.Dependencies[1])
	assert.Equal(t, "dependency3", runCmdOptions.Dependencies[2])
}

func TestRunDevFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "--dev", integrationSource)
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.Dev)
}

func TestRunDevModeOutputFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "--dev", "-o", "yaml", integrationSource)
	assert.Equal(t, true, runCmdOptions.Dev)
	assert.Equal(t, "yaml", runCmdOptions.OutputFormat)
	assert.NotNil(t, err)
	assert.Equal(t, "cannot use --dev with -o/--output option",
		err.Error())
}

func TestRunEnvFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun,
		"--env", "env1",
		"--env", "env2",
		integrationSource)
	assert.Nil(t, err)
	assert.Len(t, runCmdOptions.EnvVars, 2)
	assert.Equal(t, "env1", runCmdOptions.EnvVars[0])
	assert.Equal(t, "env2", runCmdOptions.EnvVars[1])
}

func TestRunKitFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "--kit", "myKit", integrationSource)
	assert.Nil(t, err)
	assert.Equal(t, "myKit", runCmdOptions.IntegrationKit)
}

func TestRunLabelFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun,
		"--label", "label1=l1",
		"--label", "label2=l2",
		"--label", "label3=l3",
		integrationSource)
	assert.Nil(t, err)
	assert.Len(t, runCmdOptions.Labels, 3)
	assert.Equal(t, "label1=l1", runCmdOptions.Labels[0])
	assert.Equal(t, "label2=l2", runCmdOptions.Labels[1])
	assert.Equal(t, "label3=l3", runCmdOptions.Labels[2])
}

func TestRunLabelWrongFormatFlag(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "--label", "label1", integrationSource)
	assert.NotNil(t, err)
}

func TestRunLogsFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "--logs", integrationSource)
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.Logs)
}

func TestRunMavenRepositoryFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun,
		"--maven-repository", "repo1",
		"--maven-repository", "repo2",
		"--maven-repository", "repo3",
		integrationSource)
	assert.Nil(t, err)
	assert.Len(t, runCmdOptions.Repositories, 3)
	assert.Equal(t, "repo1", runCmdOptions.Repositories[0])
	assert.Equal(t, "repo2", runCmdOptions.Repositories[1])
	assert.Equal(t, "repo3", runCmdOptions.Repositories[2])
}

func TestRunNameFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "--name", "myIntegration", integrationSource)
	assert.Nil(t, err)
	assert.Equal(t, "myIntegration", runCmdOptions.IntegrationName)
}

func TestRunOpenApiFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun,
		"--open-api", "file:oapi1",
		"--open-api", "configmap:oapi2",
		integrationSource)
	assert.Nil(t, err)
	assert.Len(t, runCmdOptions.OpenAPIs, 2)
	assert.Equal(t, "file:oapi1", runCmdOptions.OpenAPIs[0])
	assert.Equal(t, "configmap:oapi2", runCmdOptions.OpenAPIs[1])
}

func TestRunOpenApiInvalidFlag(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun,
		"--open-api", "secret:oapi1",
		"--open-api", "oapi2",
		integrationSource)
	assert.NotNil(t, err)
}

func TestRunOutputFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "-o", "yaml", integrationSource)
	assert.Nil(t, err)
	assert.Equal(t, "yaml", runCmdOptions.OutputFormat)
}

func TestRunProfileFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "--profile", "myProfile", integrationSource)
	assert.Nil(t, err)
	assert.Equal(t, "myProfile", runCmdOptions.Profile)
}

func TestRunPropertyFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun,
		"--property", "property1",
		"--property", "property2",
		"--property", "property3",
		integrationSource)
	assert.Nil(t, err)
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

func TestAddPropertyFile(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = ioutil.TempFile("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile.Close())
	assert.Nil(t, ioutil.WriteFile(tmpFile.Name(), []byte(TestPropertyFileContent), 0o400))

	properties, err := convertToTraitParameter("file:"+tmpFile.Name(), "trait.properties")
	assert.Nil(t, err)
	assert.Equal(t, 3, len(properties))
	assert.Equal(t, `trait.properties=a = b`, properties[0])
	assert.Equal(t, `trait.properties=f = g:h`, properties[1])
	assert.Equal(t, `trait.properties=i = j\nk`, properties[2])
}

func TestRunProperty(t *testing.T) {
	properties, err := convertToTraitParameter(`key=value\nnewline`, "trait.properties")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(properties))
	assert.Equal(t, `trait.properties=key = value\nnewline`, properties[0])
}

func TestRunResourceFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun,
		"--resource", "res1",
		"--resource", "res2",
		integrationSource)
	assert.Nil(t, err)
	assert.Len(t, runCmdOptions.Resources, 2)
	assert.Equal(t, "res1", runCmdOptions.Resources[0])
	assert.Equal(t, "res2", runCmdOptions.Resources[1])
}

func TestRunSaveFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "--save", integrationSource)
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.Save)
}

func TestRunSourceFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "--source", "source1", integrationSource)
	assert.Nil(t, err)
	assert.Len(t, runCmdOptions.Sources, 1)
	assert.Equal(t, "source1", runCmdOptions.Sources[0])
}

func TestRunSyncFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "--sync", integrationSource)
	assert.Nil(t, err)
	assert.Equal(t, true, runCmdOptions.Sync)
}

func TestRunExistingTraitFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun,
		"--trait", "jvm.enabled",
		"--trait", "logging.enabled",
		integrationSource)
	assert.Nil(t, err)
	assert.Len(t, runCmdOptions.Traits, 2)
	assert.Equal(t, "jvm.enabled", runCmdOptions.Traits[0])
	assert.Equal(t, "logging.enabled", runCmdOptions.Traits[1])
}

func TestRunMissingTraitFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun,
		"--trait", "bogus.missing",
		integrationSource)
	assert.NotNil(t, err)
	assert.Equal(t, "bogus.missing is not a valid trait property", err.Error())
	assert.Len(t, runCmdOptions.Traits, 1)
	assert.Equal(t, "bogus.missing", runCmdOptions.Traits[0])
}

func TestConfigureTraits(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, "run",
		"--trait", "affinity.pod-affinity=false",
		"--trait", "container.probes-enabled=false",
		"--trait", "environment.container-meta=false",
		"--trait", "jvm.print-command=false",
		"--trait", "prometheus.pod-monitor=false",
		"example.js")
	if err != nil {
		t.Error(err)
	}
	client, err := runCmdOptions.GetCmdClient()
	if err != nil {
		t.Error(err)
	}
	catalog := trait.NewCatalog(client)

	traits, err := configureTraits(runCmdOptions.Traits, catalog)

	assert.Nil(t, err)
	assert.Len(t, traits, 5)
	assertTraitConfiguration(t, traits, "affinity", `{"podAffinity":false}`)
	assertTraitConfiguration(t, traits, "container", `{"probesEnabled":false}`)
	assertTraitConfiguration(t, traits, "environment", `{"containerMeta":false}`)
	assertTraitConfiguration(t, traits, "jvm", `{"printCommand":false}`)
	assertTraitConfiguration(t, traits, "prometheus", `{"podMonitor":false}`)
}

type customTrait struct {
	trait.BaseTrait `property:",squash"`
	// SimpleMap
	SimpleMap  map[string]string            `property:"simple-map" json:"simpleMap,omitempty"`
	DoubleMap  map[string]map[string]string `property:"double-map" json:"doubleMap,omitempty"`
	SliceOfMap []map[string]string          `property:"slice-of-map" json:"sliceOfMap,omitempty"`
}

func (c customTrait) Configure(environment *trait.Environment) (bool, error) {
	panic("implement me")
}
func (c customTrait) Apply(environment *trait.Environment) error {
	panic("implement me")
}

var _ trait.Trait = &customTrait{}

type customTraitFinder struct {
}

func (finder customTraitFinder) GetTrait(id string) trait.Trait {
	if id == "custom" {
		return &customTrait{}
	}
	return nil
}

func TestTraitsNestedConfig(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, "run",
		"--trait", "custom.simple-map.a=b",
		"--trait", "custom.simple-map.y=z",
		"--trait", "custom.double-map.m.n=q",
		"--trait", "custom.double-map.m.o=w",
		"--trait", "custom.slice-of-map[0].f=g",
		"--trait", "custom.slice-of-map[3].f=h",
		"--trait", "custom.slice-of-map[2].f=i",
		"example.js")
	// We will have an error because those traits are not existing
	// however we want to test how those properties are mapped in the configuration
	assert.NotNil(t, err)
	catalog := &customTraitFinder{}
	traits, err := configureTraits(runCmdOptions.Traits, catalog)

	assert.Nil(t, err)
	assert.Len(t, traits, 1)
	assertTraitConfiguration(t, traits, "custom", `{"simpleMap":{"a":"b","y":"z"},"doubleMap":{"m":{"n":"q","o":"w"}},"sliceOfMap":[{"f":"g"},null,{"f":"i"},{"f":"h"}]}`)
}

func assertTraitConfiguration(t *testing.T, traits map[string]v1.TraitSpec, trait string, expected string) {
	t.Helper()

	assert.Contains(t, traits, trait)
	assert.Equal(t, expected, string(traits[trait].Configuration.RawMessage))
}

func TestRunUseFlowsFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun, "--use-flows=false", integrationSource)
	assert.Nil(t, err)
	assert.Equal(t, false, runCmdOptions.UseFlows)
}

func TestRunVolumeFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun,
		"-v", "pvcname:/container1/path",
		"-v", "pvcname:/container2/path",
		integrationSource)
	assert.Nil(t, err)
	assert.Len(t, runCmdOptions.Volumes, 2)
	assert.Equal(t, "pvcname:/container1/path", runCmdOptions.Volumes[0])
	assert.Equal(t, "pvcname:/container2/path", runCmdOptions.Volumes[1])
}

func TestRunVolumeFlagWrongPVCFormat(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun,
		"-v", "pvcname",
		"-v", "pvcname/container2/path",
		integrationSource)
	assert.NotNil(t, err)
}

func TestRunBuildPropertyFlag(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdRun,
		"--build-property", "build-prop1=val1",
		"--build-property", "build-prop2=val2",
		integrationSource)
	assert.Nil(t, err)
	assert.Len(t, runCmdOptions.BuildProperties, 2)
	assert.Equal(t, "build-prop1=val1", runCmdOptions.BuildProperties[0])
	assert.Equal(t, "build-prop2=val2", runCmdOptions.BuildProperties[1])
}

func TestRunValidateArgs(t *testing.T) {
	runCmdOptions, rootCmd, _ := initializeRunCmdOptions(t)
	args := []string{}
	err := runCmdOptions.validateArgs(rootCmd, args)
	assert.NotNil(t, err)
	assert.Equal(t, "run expects at least 1 argument, received 0", err.Error())

	args = []string{"run_test.go"}
	err = runCmdOptions.validateArgs(rootCmd, args)
	assert.Nil(t, err)

	args = []string{"missing_file"}
	err = runCmdOptions.validateArgs(rootCmd, args)
	assert.NotNil(t, err)
	assert.Equal(t, "One of the provided sources is not reachable: missing file or unsupported scheme in missing_file", err.Error())
}

func TestRunBinaryResource(t *testing.T) {
	binaryResourceSpec, err := binaryOrTextResource("file.ext", []byte{1, 2, 3, 4}, "application/octet-stream", false, v1.ResourceTypeData, "")
	assert.Nil(t, err)
	assert.Equal(t, "", binaryResourceSpec.Content)
	assert.NotNil(t, binaryResourceSpec.RawContent)
	assert.Equal(t, "file.ext", binaryResourceSpec.Name)
	assert.Equal(t, "application/octet-stream", binaryResourceSpec.ContentType)
	assert.False(t, binaryResourceSpec.Compression)
}

func TestRunBinaryCompressedResource(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	base64Compressed, _ := compressToString(data)
	binaryResourceSpec, err := binaryOrTextResource("file.ext", data, "application/octet-stream", true, v1.ResourceTypeData, "")
	assert.Nil(t, err)
	assert.Equal(t, base64Compressed, binaryResourceSpec.Content)
	assert.Nil(t, binaryResourceSpec.RawContent)
	assert.Equal(t, "file.ext", binaryResourceSpec.Name)
	assert.Equal(t, "application/octet-stream", binaryResourceSpec.ContentType)
	assert.True(t, binaryResourceSpec.Compression)
}

func TestRunTextResource(t *testing.T) {
	textResourceSpec, err := binaryOrTextResource("file.ext", []byte("hello world"), "text/plain", false, v1.ResourceTypeData, "")
	assert.Nil(t, err)
	assert.Equal(t, "hello world", textResourceSpec.Content)
	assert.Nil(t, textResourceSpec.RawContent)
	assert.Equal(t, "file.ext", textResourceSpec.Name)
	assert.Equal(t, "text/plain", textResourceSpec.ContentType)
	assert.False(t, textResourceSpec.Compression)
}

func TestRunTextCompressedResource(t *testing.T) {
	data := []byte("hello horld")
	base64Compressed, _ := compressToString(data)
	textResourceSpec, err := binaryOrTextResource("file.ext", []byte("hello horld"), "text/plain", true, v1.ResourceTypeData, "")
	assert.Nil(t, err)
	assert.Equal(t, base64Compressed, textResourceSpec.Content)
	assert.Nil(t, textResourceSpec.RawContent)
	assert.Equal(t, "file.ext", textResourceSpec.Name)
	assert.Equal(t, "text/plain", textResourceSpec.ContentType)
	assert.True(t, textResourceSpec.Compression)
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
	assert.Nil(t, err)
	assert.NotNil(t, integrationSpec.PodTemplate)
	assert.Equal(t, 1, len(integrationSpec.PodTemplate.Spec.Containers))
	// assert.Equal(t, 1,len(integrationSpec.PodTemplate.Spec.Containers[0].VolumeMounts))
}

func TestResolveJsonPodTemplate(t *testing.T) {
	_, rootCmd, _ := initializeRunCmdOptions(t)
	integrationSpec := v1.IntegrationSpec{}
	minifiedYamlTemplate := `{"containers": [{"name": "second"}, {"name": "integration", "env": [{"name": "CAMEL_K_DIGEST", "value": "new_value"}]}]}`

	err := resolvePodTemplate(context.TODO(), rootCmd, minifiedYamlTemplate, &integrationSpec)

	assert.Nil(t, err)
	assert.NotNil(t, integrationSpec.PodTemplate)
	assert.Equal(t, 2, len(integrationSpec.PodTemplate.Spec.Containers))
}

func TestFilterBuildPropertyFiles(t *testing.T) {
	inputValues := []string{"file:/tmp/test", "key=val"}
	outputValues := filterBuildPropertyFiles(inputValues)

	assert.Equal(t, len(outputValues), 1)
	assert.Equal(t, outputValues[0], "/tmp/test")
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
	if tmpFile, err = ioutil.TempFile("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile.Close())
	assert.Nil(t, ioutil.WriteFile(tmpFile.Name(), []byte(TestSrcContent), 0o400))
	fileName := filepath.Base(tmpFile.Name())

	buildCmdOptions, runCmd, _ := initializeRunCmdOptionsWithOutput(t)
	output, err := test.ExecuteCommand(runCmd, cmdRun, tmpFile.Name(), "-o", "yaml")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	assert.Nil(t, err)
	assert.Equal(t, fmt.Sprintf(`apiVersion: camel.apache.org/v1
kind: Integration
metadata:
  creationTimestamp: null
  name: %s
spec:
  sources:
  - content: "\nimport org.apache.camel.builder.RouteBuilder;\n\npublic class Sample
      extends RouteBuilder {\n  @Override\n  public void configure() throws Exception
      {\n\t  from(\"timer:tick\")\n        .log(\"Hello Camel K!\");\n  }\n}\n"
    name: %s
status: {}
`, fileName, fileName), output)
}

func TestTrait(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = ioutil.TempFile("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile.Close())
	assert.Nil(t, ioutil.WriteFile(tmpFile.Name(), []byte(TestSrcContent), 0o400))
	fileName := filepath.Base(tmpFile.Name())

	buildCmdOptions, runCmd, _ := initializeRunCmdOptionsWithOutput(t)
	output, err := test.ExecuteCommand(runCmd, cmdRun, tmpFile.Name(), "-o", "yaml", "-t", "mount.configs=configmap:my-cm", "--connect", "my-service-binding")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)

	assert.Nil(t, err)
	assert.Equal(t, fmt.Sprintf(`apiVersion: camel.apache.org/v1
kind: Integration
metadata:
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
      configuration:
        configs:
        - configmap:my-cm
    service-binding:
      configuration:
        services:
        - my-service-binding
status: {}
`, fileName, fileName), output)
}

func TestMissingTrait(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = ioutil.TempFile("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile.Close())
	assert.Nil(t, ioutil.WriteFile(tmpFile.Name(), []byte(TestSrcContent), 0o400))

	buildCmdOptions, runCmd, _ := initializeRunCmdOptionsWithOutput(t)
	output, err := test.ExecuteCommand(runCmd, cmdRun, tmpFile.Name(), "-o", "yaml", "-t", "bogus.fail=i-must-fail")
	assert.Equal(t, "yaml", buildCmdOptions.OutputFormat)
	assert.Equal(t, "Error: bogus.fail=i-must-fail is not a valid trait property\n", output)
	assert.NotNil(t, err)
}

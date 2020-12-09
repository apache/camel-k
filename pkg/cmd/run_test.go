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
	"io/ioutil"
	"os"
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func addTestRunCmd(options *RootCmdOptions, rootCmd *cobra.Command) *runCmdOptions {
	//add a testing version of run Command
	runCmd, runCmdOptions := newCmdRun(options)
	runCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	runCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(runCmd)
	return runCmdOptions
}

func TestRunPropertyFlag(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()

	runCmdOptions := addTestRunCmd(options, rootCmd)

	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "run", "route.java", "--property", "key1=value,othervalue", "--property", "key2=value2")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(runCmdOptions.Properties) != 2 {
		t.Fatalf("Properties expected to contain: \n %v elements\nGot:\n %v elements\n", 2, len(runCmdOptions.Properties))
	}
	if runCmdOptions.Properties[0] != "key1=value,othervalue" || runCmdOptions.Properties[1] != "key2=value2" {
		t.Fatalf("Properties expected to be: \n %v\nGot:\n %v\n", "[key1=value,othervalue key2=value2]", runCmdOptions.Properties)
	}
}

func TestRunWithAdditionalSourcesFlag(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()

	runCmdOptions := addTestRunCmd(options, rootCmd)

	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "run", "route.java", "--source", "additional-source1.java", "--source", "additional-source2.java")

	assert.Nil(t, err)
	assert.Len(t, runCmdOptions.Sources, 2)
}

func TestRunWithTraitFlag(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()

	runCmdOptions := addTestRunCmd(options, rootCmd)

	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "run", "--trait", "sample.enabled=true", "example.js")

	assert.Nil(t, err)
	assert.Equal(t, 1, len(runCmdOptions.Traits))
	assert.Equal(t, "sample.enabled=true", runCmdOptions.Traits[0])
}

func TestRunWithAdditionalTraitFlag(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()

	runCmdOptions := addTestRunCmd(options, rootCmd)

	kamelTestPostAddCommandInit(t, rootCmd)

	_, err := test.ExecuteCommand(rootCmd, "run", "--trait", "sample.enabled=true", "--trait", "sample.second=true", "example.js")

	assert.Nil(t, err)
	assert.Equal(t, 2, len(runCmdOptions.Traits))
	assert.Equal(t, "sample.enabled=true", runCmdOptions.Traits[0])
	assert.Equal(t, "sample.second=true", runCmdOptions.Traits[1])
}

//
// This test does work when running as single test but fails
// otherwise as we are using a global viper instance
//

/*
const TestKamelConfigContent = `
kamel:
  install:
    olm: false
  run:
    integration:
      route:
        sources:
        - examples/dns.js
        - examples/Sample.java
`

func TestRunWithSavedValues(t *testing.T) {
	dir, err := ioutil.TempDir("", "run-")
	assert.Nil(t, err)

	defer func() {
		_ = os.RemoveAll(dir)
	}()

	assert.Nil(t, os.Setenv("KAMEL_CONFIG_PATH", dir))
	defer func() {
		_ = os.Unsetenv("KAMEL_CONFIG_PATH")
	}()

	assert.Nil(t, ioutil.WriteFile(path.Join(dir, "kamel-config.yaml"), []byte(TestKamelConfigContent), 0644))

	options, rootCmd := kamelTestPreAddCommandInit()

	runCmdOptions := addTestRunCmd(options, rootCmd)

	kamelTestPostAddCommandInit(t, rootCmd)

	_, err = test.ExecuteCommand(rootCmd, "run", "route.java")

	assert.Nil(t, err)
	assert.Len(t, runCmdOptions.Sources, 2)
}*/

const TestPropertyFileContent = `
a=b
c\=d=e
d=c\=e
#ignore=me
f=g\:h
`

func TestRunPropertyFileFlag(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = ioutil.TempFile("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile.Close())
	assert.Nil(t, ioutil.WriteFile(tmpFile.Name(), []byte(TestPropertyFileContent), 0644))

	spec := v1.IntegrationSpec{}
	assert.Nil(t, addPropertyFile(tmpFile.Name(), &spec))
	assert.Equal(t, 4, len(spec.Configuration))
	assert.Equal(t, `a=b`, spec.Configuration[0].Value)
	assert.Equal(t, `c\=d=e`, spec.Configuration[1].Value)
	assert.Equal(t, `d=c\=e`, spec.Configuration[2].Value)
	assert.Equal(t, `f=g\:h`, spec.Configuration[3].Value)
}

func TestConfigureTraits(t *testing.T) {
	options, rootCmd := kamelTestPreAddCommandInit()
	runCmdOptions := addTestRunCmd(options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)
	_, err := test.ExecuteCommand(rootCmd, "run",
		"--trait", "affinity.pod-affinity=false",
		"--trait", "container.probes-enabled=false",
		"--trait", "environment.container-meta=false",
		"--trait", "jvm.print-command=false",
		"--trait", "prometheus.service-monitor=false",
		"example.js")
	if err != nil {
		t.Error(err)
	}
	client, err := runCmdOptions.GetCmdClient()
	if err != nil {
		t.Error(err)
	}
	catalog := trait.NewCatalog(runCmdOptions.Context, client)

	traits, err := configureTraits(runCmdOptions.Traits, catalog)

	assert.Nil(t, err)
	assert.Len(t, traits, 5)
	assertTraitConfiguration(t, traits, "affinity", `{"podAffinity":false}`)
	assertTraitConfiguration(t, traits, "container", `{"probesEnabled":false}`)
	assertTraitConfiguration(t, traits, "environment", `{"containerMeta":false}`)
	assertTraitConfiguration(t, traits, "jvm", `{"printCommand":false}`)
	assertTraitConfiguration(t, traits, "prometheus", `{"serviceMonitor":false}`)
}

func assertTraitConfiguration(t *testing.T, traits map[string]v1.TraitSpec, trait string, expected string) {
	assert.Contains(t, traits, trait)
	assert.Equal(t, expected, string(traits[trait].Configuration.RawMessage))
}

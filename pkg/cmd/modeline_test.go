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
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelineRunSimple(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	file := `
		// camel-k: dependency=mvn:org.my:lib:1.0
	`
	fileName := path.Join(dir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, "--dependency=mvn:org.my:lib:1.0"}, flags)
}

func TestModelineRunHelp(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	// no file created
	fileName := path.Join(dir, "simple.groovy")

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName, "--help"})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, "--help"}, flags)
}

func TestModelineRunChain(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	file := `
		// camel-k: dependency=mvn:org.my:lib:2.0
	`
	fileName := path.Join(dir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", "-d", "mvn:org.my:lib2:1.0", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", "-d", "mvn:org.my:lib2:1.0", fileName, "--dependency=mvn:org.my:lib:2.0"}, flags)
}

func TestModelineRunMultipleFiles(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	file := `
		// camel-k: dependency=mvn:org.my:lib1:3.0
	`
	fileName := path.Join(dir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	file2 := `
		// camel-k: dependency=mvn:org.my:lib2:3.0
	`
	fileName2 := path.Join(dir, "ext.groovy")
	err = ioutil.WriteFile(fileName2, []byte(file2), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName, fileName2})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, fileName2, "--dependency=mvn:org.my:lib1:3.0", "--dependency=mvn:org.my:lib2:3.0"}, flags)
}

func TestModelineRunProperty(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	subDir := path.Join(dir, "sub")
	err = os.Mkdir(subDir, 0777)
	assert.NoError(t, err)

	file := `
		// camel-k: property=my-prop=my-val
	`
	fileName := path.Join(subDir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, "--property=my-prop=my-val"}, flags)
}

func TestModelineRunDuplicatedProperties(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	subDir := path.Join(dir, "sub")
	err = os.Mkdir(subDir, 0777)
	assert.NoError(t, err)

	file := `
		// camel-k: property=prop1=false
		// camel-k: property=prop2=false
		// camel-k: property=foo=bar
	`
	fileName := path.Join(subDir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName, "-p", "prop1=true", "--property", "prop2=true"})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, "-p", "prop1=true", "--property", "prop2=true", "--property=foo=bar"}, flags)
}

func TestModelineRunPropertyFiles(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	subDir := path.Join(dir, "sub")
	err = os.Mkdir(subDir, 0777)
	assert.NoError(t, err)

	file := `
		// camel-k: property=file:application.properties
	`
	fileName := path.Join(subDir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	propFile := `
		a=b
	`
	propFileName := path.Join(subDir, "application.properties")
	err = ioutil.WriteFile(propFileName, []byte(propFile), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, fmt.Sprintf("--property=file:%s", propFileName)}, flags)
}

func TestModelineRunBuildProperty(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	subDir := path.Join(dir, "sub")
	err = os.Mkdir(subDir, 0777)
	assert.NoError(t, err)

	file := `
		// camel-k: build-property=my-build-prop=my-val
	`
	fileName := path.Join(subDir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, "--build-property=my-build-prop=my-val"}, flags)
}

func TestModelineRunBuildPropertyFiles(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	subDir := path.Join(dir, "sub")
	err = os.Mkdir(subDir, 0777)
	assert.NoError(t, err)

	file := `
		// camel-k: build-property=file:application.properties
	`
	fileName := path.Join(subDir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	propFile := `
		a=b
	`
	propFileName := path.Join(subDir, "application.properties")
	err = ioutil.WriteFile(propFileName, []byte(propFile), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, fmt.Sprintf("--build-property=file:%s", propFileName)}, flags)
}

func TestModelineRunDuplicateTraits(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	subDir := path.Join(dir, "sub")
	err = os.Mkdir(subDir, 0777)
	assert.NoError(t, err)

	file := `
		// camel-k: trait=trait1=false
		// camel-k: trait=trait2=false
		// camel-k: trait=foo=bar
	`
	fileName := path.Join(subDir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName, "-t", "trait1=true", "--trait", "trait2=true"})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, "-t", "trait1=true", "--trait", "trait2=true", "--trait=foo=bar"}, flags)
}

func TestModelineRunConfigConfigmap(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	subDir := path.Join(dir, "sub")
	err = os.Mkdir(subDir, 0777)
	assert.NoError(t, err)

	file := `
		// camel-k: config=configmap:my-cm
	`
	fileName := path.Join(subDir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, "--config=configmap:my-cm"}, flags)
}

func TestModelineRunConfigSecret(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	subDir := path.Join(dir, "sub")
	err = os.Mkdir(subDir, 0777)
	assert.NoError(t, err)

	file := `
		// camel-k: config=secret:my-secret
	`
	fileName := path.Join(subDir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, "--config=secret:my-secret"}, flags)
}

func TestModelineRunConfigFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	subDir := path.Join(dir, "sub")
	err = os.Mkdir(subDir, 0777)
	assert.NoError(t, err)

	file := `
		// camel-k: config=file:application.properties
	`
	fileName := path.Join(subDir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	propFile := `
		a=b
	`
	propFileName := path.Join(subDir, "application.properties")
	err = ioutil.WriteFile(propFileName, []byte(propFile), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, fmt.Sprintf("--config=file:%s", propFileName)}, flags)
}

func TestModelineRunResourceConfigmap(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	subDir := path.Join(dir, "sub")
	err = os.Mkdir(subDir, 0777)
	assert.NoError(t, err)

	file := `
		// camel-k: resource=configmap:my-cm
	`
	fileName := path.Join(subDir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, "--resource=configmap:my-cm"}, flags)
}

func TestModelineRunResourceSecret(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	subDir := path.Join(dir, "sub")
	err = os.Mkdir(subDir, 0777)
	assert.NoError(t, err)

	file := `
		// camel-k: resource=secret:my-secret
	`
	fileName := path.Join(subDir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, "--resource=secret:my-secret"}, flags)
}

func TestModelineRunResourceFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	subDir := path.Join(dir, "sub")
	err = os.Mkdir(subDir, 0777)
	assert.NoError(t, err)

	file := `
		// camel-k: resource=file:application.properties
	`
	fileName := path.Join(subDir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	propFile := `
		a=b
	`
	propFileName := path.Join(subDir, "application.properties")
	err = ioutil.WriteFile(propFileName, []byte(propFile), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"run", fileName, fmt.Sprintf("--resource=file:%s", propFileName)}, flags)
}

func TestModelineInspectSimple(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	file := `
		// camel-k: dependency=mvn:org.my:lib:1.0
	`
	fileName := path.Join(dir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "local", "inspect", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"local", "inspect", fileName, "--dependency=mvn:org.my:lib:1.0"}, flags)
}

func TestModelineInspectMultipleDeps(t *testing.T) {
	dir, err := ioutil.TempDir("", "camel-k-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	file := `
		// camel-k: dependency=mvn:org.my:lib:1.0
		// camel-k: dependency=camel-k:camel-dep
	`
	fileName := path.Join(dir, "simple.groovy")
	err = ioutil.WriteFile(fileName, []byte(file), 0777)
	assert.NoError(t, err)

	cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "local", "inspect", fileName})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"local", "inspect", fileName, "--dependency=mvn:org.my:lib:1.0", "--dependency=camel-k:camel-dep"}, flags)
}

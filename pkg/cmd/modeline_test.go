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
	"testing"

	"github.com/apache/camel-k/v2/pkg/util"

	"github.com/stretchr/testify/assert"
)

func TestModelineRunSimple(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		file := `
		// camel-k: dependency=mvn:org.my:lib:1.0
	`
		fileName := filepath.Join(dir, "simple.groovy")
		err := os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, "--dependency=mvn:org.my:lib:1.0"}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunHelp(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		// no file created
		fileName := filepath.Join(dir, "simple.groovy")

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName, "--help"})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, "--help"}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunChain(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		file := `
		// camel-k: dependency=mvn:org.my:lib:2.0
	`
		fileName := filepath.Join(dir, "simple.groovy")
		err := os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", "-d", "mvn:org.my:lib2:1.0", fileName})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", "-d", "mvn:org.my:lib2:1.0", fileName, "--dependency=mvn:org.my:lib:2.0"}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunMultipleFiles(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		file := `
		// camel-k: dependency=mvn:org.my:lib1:3.0
	`
		fileName := filepath.Join(dir, "simple.groovy")
		err := os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		file2 := `
		// camel-k: dependency=mvn:org.my:lib2:3.0
	`
		fileName2 := filepath.Join(dir, "ext.groovy")
		err = os.WriteFile(fileName2, []byte(file2), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName, fileName2})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, fileName2, "--dependency=mvn:org.my:lib1:3.0", "--dependency=mvn:org.my:lib2:3.0"}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunProperty(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		subDir := filepath.Join(dir, "sub")
		err := os.Mkdir(subDir, 0o700)
		assert.NoError(t, err)

		file := `
		// camel-k: property=my-prop=my-val
	`
		fileName := filepath.Join(subDir, "simple.groovy")
		err = os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, "--property=my-prop=my-val"}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunDuplicatedProperties(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		subDir := filepath.Join(dir, "sub")
		err := os.Mkdir(subDir, 0o700)
		assert.NoError(t, err)

		file := `
		// camel-k: property=prop1=false
		// camel-k: property=prop2=false
		// camel-k: property=foo=bar
	`
		fileName := filepath.Join(subDir, "simple.groovy")
		err = os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName, "-p", "prop1=true", "--property", "prop2=true"})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, "-p", "prop1=true", "--property", "prop2=true", "--property=foo=bar"}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunDuplicatedBuildProperties(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		subDir := filepath.Join(dir, "sub")
		err := os.Mkdir(subDir, 0o700)
		assert.NoError(t, err)

		file := `
		// camel-k: build-property=prop1=false
		// camel-k: build-property=prop2=false
		// camel-k: build-property=foo=bar
	`
		fileName := filepath.Join(subDir, "simple.groovy")
		err = os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{
			"kamel", "run", fileName,
			"--build-property", "prop1=true", "--build-property", "prop2=true",
		})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{
			"run", fileName, "--build-property", "prop1=true", "--build-property", "prop2=true",
			"--build-property=foo=bar",
		}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunPropertyFiles(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		subDir := filepath.Join(dir, "sub")
		err := os.Mkdir(subDir, 0o700)
		assert.NoError(t, err)

		file := `
		// camel-k: property=file:application.properties
	`
		fileName := filepath.Join(subDir, "simple.groovy")
		err = os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		propFile := `
		a=b
	`
		propFileName := filepath.Join(subDir, "application.properties")
		err = os.WriteFile(propFileName, []byte(propFile), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, fmt.Sprintf("--property=file:%s", propFileName)}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunBuildProperty(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		subDir := filepath.Join(dir, "sub")
		err := os.Mkdir(subDir, 0o700)
		assert.NoError(t, err)

		file := `
		// camel-k: build-property=my-build-prop=my-val
	`
		fileName := filepath.Join(subDir, "simple.groovy")
		err = os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, "--build-property=my-build-prop=my-val"}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunBuildPropertyFiles(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		subDir := filepath.Join(dir, "sub")
		err := os.Mkdir(subDir, 0o700)
		assert.NoError(t, err)

		file := `
		// camel-k: build-property=file:application.properties
	`
		fileName := filepath.Join(subDir, "simple.groovy")
		err = os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		propFile := `
		a=b
	`
		propFileName := filepath.Join(subDir, "application.properties")
		err = os.WriteFile(propFileName, []byte(propFile), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, fmt.Sprintf("--build-property=file:%s", propFileName)}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunDuplicateTraits(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		subDir := filepath.Join(dir, "sub")
		err := os.Mkdir(subDir, 0o700)
		assert.NoError(t, err)

		file := `
		// camel-k: trait=trait1=false
		// camel-k: trait=trait2=false
		// camel-k: trait=foo=bar
	`
		fileName := filepath.Join(subDir, "simple.groovy")
		err = os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName, "-t", "trait1=true", "--trait", "trait2=true"})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, "-t", "trait1=true", "--trait", "trait2=true", "--trait=foo=bar"}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunConfigConfigmap(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		subDir := filepath.Join(dir, "sub")
		err := os.Mkdir(subDir, 0o700)
		assert.NoError(t, err)

		file := `
		// camel-k: config=configmap:my-cm
	`
		fileName := filepath.Join(subDir, "simple.groovy")
		err = os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, "--config=configmap:my-cm"}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunConfigSecret(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		subDir := filepath.Join(dir, "sub")
		err := os.Mkdir(subDir, 0o700)
		assert.NoError(t, err)

		file := `
		// camel-k: config=secret:my-secret
	`
		fileName := filepath.Join(subDir, "simple.groovy")
		err = os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, "--config=secret:my-secret"}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunResourceConfigmap(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		subDir := filepath.Join(dir, "sub")
		err := os.Mkdir(subDir, 0o700)
		assert.NoError(t, err)

		file := `
		// camel-k: resource=configmap:my-cm
	`
		fileName := filepath.Join(subDir, "simple.groovy")
		err = os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, "--resource=configmap:my-cm"}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineRunResourceSecret(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {
		subDir := filepath.Join(dir, "sub")
		err := os.Mkdir(subDir, 0o700)
		assert.NoError(t, err)

		file := `
		// camel-k: resource=secret:my-secret
	`
		fileName := filepath.Join(subDir, "simple.groovy")
		err = os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, "--resource=secret:my-secret"}, flags)

		return nil
	})

	assert.NoError(t, err)
}

func TestModelineQuotedPodTemplate(t *testing.T) {
	err := util.WithTempDir("camel-k-test-", func(dir string) error {

		file := `
		// camel-k: pod-template='{ "containers": [], "securityContext": { "supplementalGroups": [ 553 ] }, "volumes": [] } }'
	`
		fileName := filepath.Join(dir, "simple.groovy")
		err := os.WriteFile(fileName, []byte(file), 0o400)
		assert.NoError(t, err)

		cmd, flags, err := NewKamelWithModelineCommand(context.TODO(), []string{"kamel", "run", fileName})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, []string{"run", fileName, "--pod-template={ \"containers\": [], \"securityContext\": { \"supplementalGroups\": [ 553 ] }, \"volumes\": [] } }"}, flags)

		return nil
	})

	assert.NoError(t, err)
}

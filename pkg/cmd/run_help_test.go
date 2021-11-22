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

	"github.com/stretchr/testify/assert"
)

func TestParseConfigOption(t *testing.T) {
	validConfigMap := "configmap:my-config_map"
	validSecret := "secret:my-secret"
	validFile := "file:/tmp/my-file.txt"
	notValid := "someprotocol:wrong"
	validLocation := "file:my-file.txt@/tmp/another-name.xml"

	configmap, err := ParseConfigOption(validConfigMap)
	assert.Nil(t, err)
	assert.Equal(t, ConfigOptionTypeConfigmap, configmap.configType)
	assert.Equal(t, "my-config_map", configmap.Name())
	secret, err := ParseConfigOption(validSecret)
	assert.Nil(t, err)
	assert.Equal(t, ConfigOptionTypeSecret, secret.configType)
	assert.Equal(t, "my-secret", secret.Name())
	file, err := ParseConfigOption(validFile)
	assert.Nil(t, err)
	assert.Equal(t, ConfigOptionTypeFile, file.configType)
	assert.Equal(t, "/tmp/my-file.txt", file.Name())
	_, err = ParseConfigOption(notValid)
	assert.NotNil(t, err)
	location, err := ParseConfigOption(validLocation)
	assert.Nil(t, err)
	assert.Equal(t, ConfigOptionTypeFile, location.configType)
	assert.Equal(t, "my-file.txt", location.Name())
	assert.Equal(t, "/tmp/another-name.xml", location.DestinationPath())
}

func TestParseConfigOptionAllParams(t *testing.T) {
	cm1 := "configmap:my-config_map/key@/tmp/my"
	cm2 := "configmap:my-config_map/key"
	cm3 := "configmap:my-config_map@/tmp/my"
	cm4 := "configmap:my-config_map"
	sec1 := "secret:sec/key@/tmp/sec"
	sec2 := "secret:sec/key"
	sec3 := "secret:sec@/tmp/sec"
	sec4 := "secret:sec"
	file1 := "file:/path/to/my-file.txt@/tmp/file.txt"
	file2 := "file:/path/to/my-file.txt"
	file3 := "file:/path to/my-file.txt"

	parsedCm1, err := ParseConfigOption(cm1)
	assert.Nil(t, err)
	assert.Equal(t, "configmap", parsedCm1.Type())
	assert.Equal(t, "my-config_map", parsedCm1.Name())
	assert.Equal(t, "key", parsedCm1.Key())
	assert.Equal(t, "/tmp/my", parsedCm1.DestinationPath())

	parsedCm2, err := ParseConfigOption(cm2)
	assert.Nil(t, err)
	assert.Equal(t, "configmap", parsedCm2.Type())
	assert.Equal(t, "my-config_map", parsedCm2.Name())
	assert.Equal(t, "key", parsedCm2.Key())
	assert.Equal(t, "", parsedCm2.DestinationPath())

	parsedCm3, err := ParseConfigOption(cm3)
	assert.Nil(t, err)
	assert.Equal(t, "configmap", parsedCm3.Type())
	assert.Equal(t, "my-config_map", parsedCm3.Name())
	assert.Equal(t, "", parsedCm3.Key())
	assert.Equal(t, "/tmp/my", parsedCm3.DestinationPath())

	parsedCm4, err := ParseConfigOption(cm4)
	assert.Nil(t, err)
	assert.Equal(t, "configmap", parsedCm4.Type())
	assert.Equal(t, "my-config_map", parsedCm4.Name())
	assert.Equal(t, "", parsedCm4.Key())
	assert.Equal(t, "", parsedCm4.DestinationPath())

	parsedSec1, err := ParseConfigOption(sec1)
	assert.Nil(t, err)
	assert.Equal(t, "secret", parsedSec1.Type())
	assert.Equal(t, "sec", parsedSec1.Name())
	assert.Equal(t, "key", parsedSec1.Key())
	assert.Equal(t, "/tmp/sec", parsedSec1.DestinationPath())

	parsedSec2, err := ParseConfigOption(sec2)
	assert.Nil(t, err)
	assert.Equal(t, "secret", parsedSec2.Type())
	assert.Equal(t, "sec", parsedSec2.Name())
	assert.Equal(t, "key", parsedSec2.Key())
	assert.Equal(t, "", parsedSec2.DestinationPath())

	parsedSec3, err := ParseConfigOption(sec3)
	assert.Nil(t, err)
	assert.Equal(t, "secret", parsedSec3.Type())
	assert.Equal(t, "sec", parsedSec3.Name())
	assert.Equal(t, "", parsedSec3.Key())
	assert.Equal(t, "/tmp/sec", parsedSec3.DestinationPath())

	parsedSec4, err := ParseConfigOption(sec4)
	assert.Nil(t, err)
	assert.Equal(t, "secret", parsedSec4.Type())
	assert.Equal(t, "sec", parsedSec4.Name())
	assert.Equal(t, "", parsedSec4.Key())
	assert.Equal(t, "", parsedSec4.DestinationPath())

	parsedFile1, err := ParseConfigOption(file1)
	assert.Nil(t, err)
	assert.Equal(t, "file", parsedFile1.Type())
	assert.Equal(t, "/path/to/my-file.txt", parsedFile1.Name())
	assert.Equal(t, "", parsedFile1.Key())
	assert.Equal(t, "/tmp/file.txt", parsedFile1.DestinationPath())

	parsedFile2, err := ParseConfigOption(file2)
	assert.Nil(t, err)
	assert.Equal(t, "file", parsedFile2.Type())
	assert.Equal(t, "/path/to/my-file.txt", parsedFile2.Name())
	assert.Equal(t, "", parsedFile2.Key())
	assert.Equal(t, "", parsedFile2.DestinationPath())

	parsedFile3, err := ParseConfigOption(file3)
	assert.Nil(t, err)
	assert.Equal(t, "file", parsedFile3.Type())
	assert.Equal(t, "/path to/my-file.txt", parsedFile3.Name())
	assert.Equal(t, "", parsedFile3.Key())
	assert.Equal(t, "", parsedFile3.DestinationPath())
}

func TestFilterFileLocation(t *testing.T) {
	optionFileLocations := []string{
		"file:/path/to/valid/file",
		"file:app.properties",
		"configmap:my-configmap",
		"secret:my-secret",
		"file:/validfile@/tmp/destination",
	}

	filteredOptions := filterFileLocation(optionFileLocations)

	assert.Equal(t, 3, len(filteredOptions))
	assert.Equal(t, "/path/to/valid/file", filteredOptions[0])
	assert.Equal(t, "app.properties", filteredOptions[1])
	assert.Equal(t, "/validfile", filteredOptions[2])
}

func TestValidateFileLocation(t *testing.T) {
	validLocation := "file:my-file.txt@/tmp/another-name.xml"
	etcCamelLocation := "configmap:my-cm@/etc/camel/configmaps"
	deploymentsDepsLocation := "secret:my-sec@/deployments/dependencies"

	_, err := ParseConfigOption(validLocation)
	assert.Nil(t, err)
	_, err = ParseConfigOption(etcCamelLocation)
	assert.NotNil(t, err)
	assert.Equal(t, "you cannot mount a file under /etc/camel path", err.Error())
	_, err = ParseConfigOption(deploymentsDepsLocation)
	assert.NotNil(t, err)
	assert.Equal(t, "you cannot mount a file under /deployments/dependencies path", err.Error())
}

func TestExtractProperties_SingleKeyValue(t *testing.T) {
	correctValues := []string{"key=val", "key = val", "key= val", " key   =  val"}
	for _, val := range correctValues {
		prop, err := extractProperties(val)
		assert.Nil(t, err)
		value, ok := prop.Get("key")
		assert.True(t, ok)
		assert.Equal(t, "val", value)
	}
}

func TestExtractProperties_FromFile(t *testing.T) {
	var tmpFile1 *os.File
	var err error
	if tmpFile1, err = ioutil.TempFile("", "camel-k-*.properties"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile1.Close())
	assert.Nil(t, ioutil.WriteFile(tmpFile1.Name(), []byte(`
	key=value
	#key2=value2
	my.key=value
	`), 0o400))

	props, err := extractProperties("file:" + tmpFile1.Name())
	assert.Nil(t, err)
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
	if tmpFile1, err = ioutil.TempFile("", "camel-k-*.properties"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile1.Close())
	assert.Nil(t, ioutil.WriteFile(tmpFile1.Name(), []byte(`
	key=value
	#key2=value2
	my.key=value
	`), 0o400))

	properties := []string{"key=override", "file:" + tmpFile1.Name(), "my.key = override"}
	props, err := mergePropertiesWithPrecedence(properties)
	assert.Nil(t, err)
	assert.Equal(t, 2, props.Len())
	val, ok := props.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "override", val)
	val, ok = props.Get("my.key")
	assert.True(t, ok)
	assert.Equal(t, "override", val)
}

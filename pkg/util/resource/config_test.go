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

package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfigOption(t *testing.T) {
	validConfigMap := "configmap:my-config_map"
	validSecret := "secret:my-secret"
	validFile := "file:/tmp/my-file.txt"
	notValid := "someprotocol:wrong"
	validLocation := "file:my-file.txt@/tmp/another-name.xml"

	configmap, err := ParseConfig(validConfigMap)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeConfigmap, configmap.storageType)
	assert.Equal(t, "my-config_map", configmap.Name())
	secret, err := ParseConfig(validSecret)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeSecret, secret.storageType)
	assert.Equal(t, "my-secret", secret.Name())
	file, err := ParseConfig(validFile)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeFile, file.storageType)
	assert.Equal(t, "/tmp/my-file.txt", file.Name())
	_, err = ParseConfig(notValid)
	assert.NotNil(t, err)
	location, err := ParseConfig(validLocation)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeFile, location.storageType)
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

	parsedCm1, err := ParseConfig(cm1)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeConfigmap, parsedCm1.StorageType())
	assert.Equal(t, "my-config_map", parsedCm1.Name())
	assert.Equal(t, "key", parsedCm1.Key())
	assert.Equal(t, "/tmp/my", parsedCm1.DestinationPath())

	parsedCm2, err := ParseConfig(cm2)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeConfigmap, parsedCm2.StorageType())
	assert.Equal(t, "my-config_map", parsedCm2.Name())
	assert.Equal(t, "key", parsedCm2.Key())
	assert.Equal(t, "", parsedCm2.DestinationPath())

	parsedCm3, err := ParseConfig(cm3)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeConfigmap, parsedCm3.StorageType())
	assert.Equal(t, "my-config_map", parsedCm3.Name())
	assert.Equal(t, "", parsedCm3.Key())
	assert.Equal(t, "/tmp/my", parsedCm3.DestinationPath())

	parsedCm4, err := ParseConfig(cm4)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeConfigmap, parsedCm4.StorageType())
	assert.Equal(t, "my-config_map", parsedCm4.Name())
	assert.Equal(t, "", parsedCm4.Key())
	assert.Equal(t, "", parsedCm4.DestinationPath())

	parsedSec1, err := ParseConfig(sec1)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeSecret, parsedSec1.StorageType())
	assert.Equal(t, "sec", parsedSec1.Name())
	assert.Equal(t, "key", parsedSec1.Key())
	assert.Equal(t, "/tmp/sec", parsedSec1.DestinationPath())

	parsedSec2, err := ParseConfig(sec2)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeSecret, parsedSec2.StorageType())
	assert.Equal(t, "sec", parsedSec2.Name())
	assert.Equal(t, "key", parsedSec2.Key())
	assert.Equal(t, "", parsedSec2.DestinationPath())

	parsedSec3, err := ParseConfig(sec3)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeSecret, parsedSec3.StorageType())
	assert.Equal(t, "sec", parsedSec3.Name())
	assert.Equal(t, "", parsedSec3.Key())
	assert.Equal(t, "/tmp/sec", parsedSec3.DestinationPath())

	parsedSec4, err := ParseConfig(sec4)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeSecret, parsedSec4.StorageType())
	assert.Equal(t, "sec", parsedSec4.Name())
	assert.Equal(t, "", parsedSec4.Key())
	assert.Equal(t, "", parsedSec4.DestinationPath())

	parsedFile1, err := ParseConfig(file1)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeFile, parsedFile1.StorageType())
	assert.Equal(t, "/path/to/my-file.txt", parsedFile1.Name())
	assert.Equal(t, "", parsedFile1.Key())
	assert.Equal(t, "/tmp/file.txt", parsedFile1.DestinationPath())

	parsedFile2, err := ParseConfig(file2)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeFile, parsedFile2.StorageType())
	assert.Equal(t, "/path/to/my-file.txt", parsedFile2.Name())
	assert.Equal(t, "", parsedFile2.Key())
	assert.Equal(t, "", parsedFile2.DestinationPath())

	parsedFile3, err := ParseConfig(file3)
	assert.Nil(t, err)
	assert.Equal(t, StorageTypeFile, parsedFile3.StorageType())
	assert.Equal(t, "/path to/my-file.txt", parsedFile3.Name())
	assert.Equal(t, "", parsedFile3.Key())
	assert.Equal(t, "", parsedFile3.DestinationPath())
}

func TestValidateFileLocation(t *testing.T) {
	validLocation := "file:my-file.txt@/tmp/another-name.xml"
	etcCamelLocation := "configmap:my-cm@/etc/camel/configmaps"
	deploymentsDepsLocation := "secret:my-sec@/deployments/dependencies"

	_, err := ParseConfig(validLocation)
	assert.Nil(t, err)
	_, err = ParseConfig(etcCamelLocation)
	assert.NotNil(t, err)
	assert.Equal(t, "you cannot mount a file under /etc/camel path", err.Error())
	_, err = ParseConfig(deploymentsDepsLocation)
	assert.NotNil(t, err)
	assert.Equal(t, "you cannot mount a file under /deployments/dependencies path", err.Error())
}

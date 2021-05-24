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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePropertyFiles_ShouldSucceed(t *testing.T) {
	var tmpFile1 *os.File
	var err error
	if tmpFile1, err = ioutil.TempFile("", "camel-k-*.properties"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile1.Close())
	assert.Nil(t, ioutil.WriteFile(tmpFile1.Name(), []byte("key=value"), 0644))

	inputValues := []string{tmpFile1.Name()}
	err = validatePropertyFiles(inputValues)

	assert.Nil(t, err)
}

func TestValidatePropertyFiles_ShouldFailNotAPropertiesFile(t *testing.T) {
	var tmpFile1 *os.File
	var err error
	if tmpFile1, err = ioutil.TempFile("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile1.Close())
	assert.Nil(t, ioutil.WriteFile(tmpFile1.Name(), []byte("key=value"), 0644))

	inputValues := []string{tmpFile1.Name()}
	err = validatePropertyFiles(inputValues)

	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "supported property files must have a .properties extension"))
}

func TestValidatePropertyFiles_ShouldFailNotFound(t *testing.T) {
	inputValues := []string{"/tmp/not-found.properties"}
	err := validatePropertyFiles(inputValues)

	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "unable to access property file"))
}

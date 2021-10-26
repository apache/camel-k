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

func TestRawContentFileMissing(t *testing.T) {
	_, _, err := loadRawContent("dsadas")
	assert.NotNil(t, err)
}

func TestRawBinaryContentType(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = ioutil.TempFile("", "camel-k-*.json"); err != nil {
		t.Error(err)
	}
	assert.Nil(t, tmpFile.Close())
	assert.Nil(t, ioutil.WriteFile(tmpFile.Name(), []byte{1, 2, 3, 4, 5, 6}, 0o644))

	data, contentType, err := loadRawContent(tmpFile.Name())
	assert.Nil(t, err)
	assert.Equal(t, []byte{1, 2, 3, 4, 5, 6}, data)
	assert.True(t, isBinary(contentType))
}

func TestRawApplicationContentType(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = ioutil.TempFile("", "camel-k-*.json"); err != nil {
		t.Error(err)
	}
	assert.Nil(t, tmpFile.Close())
	assert.Nil(t, ioutil.WriteFile(tmpFile.Name(), []byte(`{"hello":"world"}`), 0o644))

	data, contentType, err := loadRawContent(tmpFile.Name())
	assert.Nil(t, err)
	assert.Equal(t, `{"hello":"world"}`, string(data))
	assert.False(t, isBinary(contentType))
}

func TestTextContentType(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = ioutil.TempFile("", "camel-k-*.json"); err != nil {
		t.Error(err)
	}
	assert.Nil(t, tmpFile.Close())
	assert.Nil(t, ioutil.WriteFile(tmpFile.Name(), []byte(`{"hello":"world"}`), 0o644))

	data, contentType, compressed, err := loadTextContent(tmpFile.Name(), false)
	assert.Nil(t, err)
	assert.Equal(t, `{"hello":"world"}`, data)
	assert.False(t, isBinary(contentType))
	assert.False(t, compressed)
}

func TestTextCompressed(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = ioutil.TempFile("", "camel-k-*.json"); err != nil {
		t.Error(err)
	}
	assert.Nil(t, tmpFile.Close())
	assert.Nil(t, ioutil.WriteFile(tmpFile.Name(), []byte(`{"hello":"world"}`), 0o644))

	data, contentType, compressed, err := loadTextContent(tmpFile.Name(), true)
	assert.Nil(t, err)
	assert.NotEqual(t, `{"hello":"world"}`, data)
	assert.False(t, isBinary(contentType))
	assert.True(t, compressed)
}

func TestIsBinary(t *testing.T) {
	assert.True(t, isBinary("image/jpeg"))
	assert.True(t, isBinary("application/zip"))
	assert.False(t, isBinary("text/plain"))
}

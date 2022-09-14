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
	"github.com/apache/camel-k/pkg/cmd/source"
	"github.com/stretchr/testify/assert"
)

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
	base64Compressed, _ := source.CompressToString(data)
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
	base64Compressed, _ := source.CompressToString(data)
	textResourceSpec, err := binaryOrTextResource("file.ext", []byte("hello horld"), "text/plain", true, v1.ResourceTypeData, "")
	assert.Nil(t, err)
	assert.Equal(t, base64Compressed, textResourceSpec.Content)
	assert.Nil(t, textResourceSpec.RawContent)
	assert.Equal(t, "file.ext", textResourceSpec.Name)
	assert.Equal(t, "text/plain", textResourceSpec.ContentType)
	assert.True(t, textResourceSpec.Compression)
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

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
	"net/url"
	"os"
	"strings"
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

func TestDownloadDependencyWithBadURL(t *testing.T) {
	u, _ := url.Parse("http://foo")
	_, err := downloadDependency(context.Background(), *u)
	assert.NotNil(t, err)
}

func TestDownloadDependencyWithFileNameInURL(t *testing.T) {
	u, _ := url.Parse("https://repo1.maven.org/maven2/org/apache/camel/camel-core/3.18.2/camel-core-3.18.2.jar")
	path, err := downloadDependency(context.Background(), *u)
	t.Cleanup(func() { os.Remove(path) })
	assert.Nil(t, err)
	assert.True(t, strings.HasSuffix(path, "camel-core-3.18.2.jar"), "The name of the jar file is expected")
	_, err = os.Stat(path)
	assert.Nil(t, err)
}

func TestDownloadDependencyWithFileNameInQuery(t *testing.T) {
	u, _ := url.Parse("https://search.maven.org/remotecontent?filepath=org/apache/camel/quarkus/camel-quarkus-file/2.12.0/camel-quarkus-file-2.12.0.jar")
	path, err := downloadDependency(context.Background(), *u)
	t.Cleanup(func() { os.Remove(path) })
	assert.Nil(t, err)
	assert.True(t, strings.HasSuffix(path, "camel-quarkus-file-2.12.0.jar"), "The name of the jar file is expected")
	_, err = os.Stat(path)
	assert.Nil(t, err)
}

func TestDownloadDependencyWithoutFileName(t *testing.T) {
	u, _ := url.Parse("https://search.maven.org")
	path, err := downloadDependency(context.Background(), *u)
	t.Cleanup(func() { os.Remove(path) })
	assert.Nil(t, err)
	_, err = os.Stat(path)
	assert.Nil(t, err)
}

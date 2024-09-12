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

package integrationplatform

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadKamelet(t *testing.T) {
	itp := v1.NewIntegrationPlatform("itp-ns", "my-itp")
	var tmpKameletFile *os.File
	var err error
	tmpKameletFile, err = os.CreateTemp("/tmp", "timer-source-*.kamelet.yaml")
	require.NoError(t, err)
	require.NoError(t, tmpKameletFile.Close())
	require.NoError(t, os.WriteFile(tmpKameletFile.Name(), []byte(`apiVersion: camel.apache.org/v1
kind: Kamelet
metadata:
  name: timer-source
  annotations:
    camel.apache.org/kamelet.icon: "data:image/svg+xml;base64,XYZABC123"
  labels:
    camel.apache.org/kamelet.type: "source"
spec:
  definition:
    title: "Timer Source"
    description: "Produces periodic events with a custom payload"
    required:
      - message
    properties:
      period:
        title: Period
        description: The interval between two events
        type: integer
        default: 1000
      message:
        title: Message
        description: The message to generate
        type: string
        example: "hello world"
  dataTypes:
    out:
      default: text
      types:
        text:
          mediaType: text/plain
  template:
    from:
      uri: timer:tick
      parameters:
        period: "{{period}}"
      steps:
      - setBody:
          constant: "{{message}}"
      - to: "kamelet:sink"
`), 0o400))

	kamelet, err := loadKamelet(tmpKameletFile.Name(), &itp)

	assert.NotNil(t, kamelet)
	require.NoError(t, err)
	assert.Equal(t, "timer-source", kamelet.GetName())
	assert.Equal(t, "itp-ns", kamelet.GetNamespace())
	assert.Len(t, kamelet.GetLabels(), 3)
	assert.Equal(t, "true", kamelet.GetLabels()[v1.KameletBundledLabel])
	assert.Equal(t, "true", kamelet.GetLabels()[v1.KameletReadOnlyLabel])
	assert.Len(t, kamelet.GetAnnotations(), 2)
	assert.NotNil(t, kamelet.GetAnnotations()[kamelVersionAnnotation])
	assert.Equal(t, "my-itp", kamelet.GetOwnerReferences()[0].Name)
}

func TestPrepareKameletsPermissions(t *testing.T) {
	c, err := test.NewFakeClient()
	assert.NoError(t, err)
	err = prepareKameletsPermissions(context.TODO(), c, "camel-k")
	assert.NoError(t, err)
}

func TestPrepareKameletsDirectory(t *testing.T) {
	kameletDir := prepareKameletDirectory()
	assert.Equal(t, defaultKameletDir, kameletDir)
}

func TestDownloadKameletDependencyAndExtract(t *testing.T) {
	// use local Maven executable in tests
	t.Setenv("MAVEN_WRAPPER", boolean.FalseString)
	_, ok := os.LookupEnv("MAVEN_CMD")
	if !ok {
		t.Setenv("MAVEN_CMD", "mvn")
	}

	tmpDir, err := os.MkdirTemp("/tmp", "kamelets*")
	assert.NoError(t, err)
	// Load default catalog in order to get the default Camel version
	c, err := camel.DefaultCatalog()
	assert.NoError(t, err)
	camelVersion := c.Runtime.Metadata["camel.version"]
	assert.NotEqual(t, "", camelVersion)
	err = downloadKameletDependency(context.TODO(), camelVersion, tmpDir)
	assert.NoError(t, err)
	downloadedDependency, err := os.Stat(path.Join(tmpDir, fmt.Sprintf("camel-kamelets-%s.jar", camelVersion)))
	assert.NoError(t, err)

	assert.Equal(t, fmt.Sprintf("camel-kamelets-%s.jar", camelVersion), downloadedDependency.Name())

	// We can extract the Kamelets now
	err = extractKameletsFromDependency(context.TODO(), camelVersion, tmpDir)
	assert.NoError(t, err)
	kameletsDir, err := os.Stat(path.Join(tmpDir, "kamelets"))
	assert.NoError(t, err)
	assert.True(t, kameletsDir.IsDir())
	count := 0
	err = filepath.WalkDir(path.Join(tmpDir, "kamelets"), func(p string, f fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(f.Name(), ".yaml") || strings.HasSuffix(f.Name(), ".yml") {
			count++
		}
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, count > 0)
}

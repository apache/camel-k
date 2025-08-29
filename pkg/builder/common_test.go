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

package builder

import (
	"os"
	"path"
	"testing"

	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessDependenciesSkipNested(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDirNested, err := os.MkdirTemp(tmpDir, "nested")
	require.NoError(t, err)
	err = util.WriteFileWithContent(path.Join(tmpDir, "deps.jar"), []byte("bogus"))
	require.NoError(t, err)
	err = util.WriteFileWithContent(path.Join(tmpDir, "deps.txt"), []byte("bogus"))
	require.NoError(t, err)
	err = util.WriteFileWithContent(path.Join(tmpDirNested, "depsNested.jar"), []byte("bogus"))
	require.NoError(t, err)
	err = util.WriteFileWithContent(path.Join(tmpDirNested, "depsNested.txt"), []byte("bogus"))
	require.NoError(t, err)

	artifactsList, err := processDependencies(tmpDir, true)
	require.NoError(t, err)
	assert.Len(t, artifactsList, 1)
	assert.Equal(t, "dependencies/deps.jar", artifactsList[0].Target)
}

func TestProcessDependenciesIncludeNested(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDirNested, err := os.MkdirTemp(tmpDir, "nested")
	require.NoError(t, err)
	err = util.WriteFileWithContent(path.Join(tmpDir, "deps.jar"), []byte("bogus"))
	require.NoError(t, err)
	err = util.WriteFileWithContent(path.Join(tmpDir, "deps.txt"), []byte("bogus"))
	require.NoError(t, err)
	err = util.WriteFileWithContent(path.Join(tmpDirNested, "depsNested.jar"), []byte("bogus"))
	require.NoError(t, err)
	err = util.WriteFileWithContent(path.Join(tmpDirNested, "depsNested.txt"), []byte("bogus"))
	require.NoError(t, err)

	artifactsList, err := processDependencies(tmpDir, false)
	require.NoError(t, err)
	assert.Len(t, artifactsList, 2)
	assert.Equal(t, "dependencies/deps.jar", artifactsList[0].Target)
	assert.Contains(t, artifactsList[1].Target, "depsNested.jar")
}

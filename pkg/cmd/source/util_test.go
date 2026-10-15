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

package source

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCorrectFileValuesButNotFound(t *testing.T) {
	value1, err1 := IsLocalAndFileExists("c:\\test")
	value2, err2 := IsLocalAndFileExists("path/to/file")

	// they are all not found, but it must not panic
	require.NoError(t, err1)
	assert.False(t, value1)
	require.NoError(t, err2)
	assert.False(t, value2)
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

func denyAccess(t *testing.T, dir string) func() {
	t.Helper()
	if isWindows() {
		out, err := exec.Command("icacls", dir, "/deny", "*S-1-1-0:(OI)(CI)(F)").CombinedOutput()
		require.NoError(t, err, "failed to deny directory access: %s", string(out))
		return func() {
			out, err := exec.Command("icacls", dir, "/remove:d", "*S-1-1-0").CombinedOutput()
			assert.NoError(t, err, "failed to restore directory access: %s", string(out))
		}
	}

	fileInfo, err := os.Stat(dir)
	require.NoError(t, err)
	originalDirMode := fileInfo.Mode()

	err = os.Chmod(dir, 0000)
	require.NoError(t, err)

	return func() {
		err = os.Chmod(dir, originalDirMode)
		assert.NoError(t, err)
	}
}

func TestPermissionDenied(t *testing.T) {
	dir := t.TempDir()

	filename := filepath.Join(dir, "file.txt")
	f, err := os.Create(filename)
	require.NoError(t, err)
	_ = f.Close()

	restore := denyAccess(t, dir)
	defer restore()

	value, err := IsLocalAndFileExists(filename)
	// must not panic because a permission error
	require.Error(t, err)
	assert.False(t, value)
}

func TestSupportedScheme(t *testing.T) {
	gistValue, err1 := IsLocalAndFileExists("gist:some/gist/resource")
	githubValue, err2 := IsLocalAndFileExists("github:some/github/resource")
	httpValue, err3 := IsLocalAndFileExists("http://some/http/resource")
	httpsValue, err4 := IsLocalAndFileExists("https://some/https/resource")

	require.NoError(t, err1)
	assert.False(t, gistValue)
	require.NoError(t, err2)
	assert.False(t, githubValue)
	require.NoError(t, err3)
	assert.False(t, httpValue)
	require.NoError(t, err4)
	assert.False(t, httpsValue)
}

func TestUnSupportedScheme(t *testing.T) {
	value, err := IsLocalAndFileExists("bad_scheme:some/bad/resource")
	// must not report an error
	require.NoError(t, err)
	assert.False(t, value)
}

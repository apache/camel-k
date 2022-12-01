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

package util

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringContainsPrefix(t *testing.T) {
	args := []string{"install", "--operator-image=xxx/yyy:zzz", "--registry", "image-registry.openshift-image-registry.svc:5000"}
	assert.True(t, StringContainsPrefix(args, "--operator-image="))
	assert.False(t, StringContainsPrefix(args, "--olm"))
}

func TestRandomString(t *testing.T) {
	assert.Equal(t, 10, len(RandomString(10)))
	assert.NotEqual(t, RandomString(10), RandomString(10))
}

func TestSubstringFrom(t *testing.T) {
	assert.Equal(t, "/bbb/ccc", SubstringFrom("aaa/bbb/ccc", "/"))
	assert.Empty(t, SubstringFrom("aaa/bbb/ccc", "?"))
}

func TestSubstringBefore(t *testing.T) {
	assert.Equal(t, "aaa/bbb", SubstringBefore("aaa/bbb/ccc", "/"))
	assert.Equal(t, "aaa/bbb", SubstringBefore("aaa/bbb?ccc=ddd", "?"))
	assert.Empty(t, SubstringBefore("aaa/bbb/ccc", "?"))
}

func TestCopyDir(t *testing.T) {
	srcDir := "../../install"
	tmpDir, err := os.MkdirTemp("", "TestCopyDir-*")
	defer os.RemoveAll(tmpDir)
	destDir := filepath.Join(tmpDir, "install")

	require.NoError(t, err)
	fmt.Println(destDir)
	err = CopyDir(srcDir, destDir)
	assert.NoError(t, err)
}

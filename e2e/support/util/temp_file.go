//go:build integration
// +build integration

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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apache/camel-k/pkg/util"
)

func MakeTempCopy(t *testing.T, fileName string) string {
	t.Helper()

	_, simpleName := filepath.Split(fileName)
	tmpDir := MakeTempDir(t)
	tmpFileName := filepath.Join(tmpDir, simpleName)
	if _, err := util.CopyFile(fileName, tmpFileName); err != nil {
		t.Error(err)
		t.FailNow()
	}
	return tmpFileName
}

func MakeTempCopyDir(t *testing.T, dirName string) string {
	t.Helper()

	dirName = strings.TrimSuffix(dirName, "/")
	_, simpleName := filepath.Split(dirName)
	tmpDir := MakeTempDir(t)
	tmpDirName := filepath.Join(tmpDir, simpleName)
	if err := util.CopyDir(dirName, tmpDirName); err != nil {
		t.Error(err)
		t.FailNow()
	}
	return tmpDirName
}

// ReplaceInFile replace strings in a file with new values
func ReplaceInFile(t *testing.T, fileName string, old, new string) {
	t.Helper()

	content, err := os.ReadFile(fileName)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	res := strings.ReplaceAll(string(content), old, new)
	if err = os.WriteFile(fileName, []byte(res), os.FileMode(0777)); err != nil {
		t.Error(err)
		t.FailNow()
	}
}

func MakeTempDir(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "camel-k-")
	if err != nil {
		t.Error(err)
		t.FailNow()
		return ""
	}
	return tmpDir
}

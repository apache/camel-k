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
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
)

func MakeTempCopy(t *testing.T, fileName string) string {
	_, simpleName := path.Split(fileName)
	var err error
	tmpDir := MakeTempDir(t)
	tmpFileName := path.Join(tmpDir, simpleName)
	var content []byte
	if content, err = ioutil.ReadFile(fileName); err != nil {
		t.Error(err)
		t.FailNow()
	}
	if err = ioutil.WriteFile(tmpFileName, content, os.FileMode(0777)); err != nil {
		t.Error(err)
		t.FailNow()
	}
	return tmpFileName
}

// ReplaceInFile replace strings in a file with new values
func ReplaceInFile(t *testing.T, fileName string, old, new string) {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	res := strings.ReplaceAll(string(content), old, new)
	if err = ioutil.WriteFile(fileName, []byte(res), os.FileMode(0777)); err != nil {
		t.Error(err)
		t.FailNow()
	}
}

func MakeTempDir(t *testing.T) string {
	var tmpDir string
	var err error
	if tmpDir, err = ioutil.TempDir("", "camel-k-"); err != nil {
		t.Error(err)
		t.FailNow()
		return ""
	}
	return tmpDir
}

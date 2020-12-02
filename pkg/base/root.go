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
package base

import (
	"os"
	"path/filepath"
)

var (
	GoModDirectory string
)

func FileExists(name string) bool {
	stat, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return !stat.IsDir()
}

func init() {

	// Save the original directory the process started in.
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	initialDir, err := filepath.Abs(wd)
	if err != nil {
		panic(err)
	}

	// Find the module dir..
	current := ""
	for next := initialDir; current != next; next = filepath.Dir(current) {
		current = next
		if FileExists(filepath.Join(current, "go.mod")) && FileExists(filepath.Join(current, "go.sum")) {
			GoModDirectory = current
			break
		}
	}

	if GoModDirectory == "" {
		panic("could not find the root module directory")
	}
}

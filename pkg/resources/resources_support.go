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

package resources

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/apache/camel-k/pkg/util/log"
)

//
//go:generate go run ../../cmd/util/vfs-gen resources config
//
// ResourceAsString returns the named resource content as string.
func ResourceAsString(name string) string {
	return string(Resource(name))
}

// Resource provides an easy access to embedded assets.
func Resource(name string) []byte {
	name = strings.Trim(name, " ")
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}

	file, err := assets.Open(name)
	if err != nil {
		log.Error(err, "cannot access resource file", "file", name)
		return nil
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Error(err, "error while reading resource file", "file", name)
		return nil
	}
	return data
}

// TemplateResource loads a file resource as go template and processes it using the given parameters.
func TemplateResource(name string, params interface{}) (string, error) {
	rawData := ResourceAsString(name)
	if rawData == "" {
		return "", nil
	}
	tmpl, err := template.New(name).Parse(rawData)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// DirExists tells if a directory exists and can be listed for files.
func DirExists(dirName string) bool {
	if _, err := assets.Open(dirName); err != nil {
		return false
	}
	return true
}

// WithPrefix lists all file names that begins with the give path prefix
// If pathPrefix is a path of directories then be sure to end it with a '/'.
func WithPrefix(pathPrefix string) []string {
	dirPath := filepath.Dir(pathPrefix)

	var res []string
	for _, path := range Resources(dirPath) {
		if result, _ := filepath.Match(pathPrefix+"*", path); result {
			res = append(res, path)
		}
	}

	return res
}

// Resources lists all file names in the given path (starts with '/').
func Resources(dirName string) []string {
	dir, err := assets.Open(dirName)
	if err != nil {
		log.Error(err, "error while listing resource files", "dir", dirName)
		return nil
	}
	defer dir.Close()
	info, err := dir.Stat()
	if err != nil {
		log.Error(err, "error while doing stat on directory", "dir", dirName)
		return nil
	}
	if !info.IsDir() {
		log.Error(err, "location is not a directory", "dir", dirName)
		return nil
	}
	files, err := dir.Readdir(-1)
	if err != nil {
		log.Error(err, "error while listing files on directory", "dir", dirName)
		return nil
	}
	var res []string
	for _, f := range files {
		if !f.IsDir() {
			res = append(res, filepath.Join(dirName, f.Name()))
		}
	}
	return res
}

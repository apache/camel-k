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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/apache/camel-k/pkg/util"

	"github.com/pkg/errors"
)

//
//go:generate go run ../../cmd/util/vfs-gen resources config
//
// ResourceAsString returns the named resource content as string.
func ResourceAsString(name string) (string, error) {
	data, err := Resource(name)
	return string(data), err
}

// Resource provides an easy way to access to embedded assets.
func Resource(name string) ([]byte, error) {
	name = strings.Trim(name, " ")
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}

	file, err := assets.Open(fixPath(name))
	if err != nil {
		return nil, errors.Wrapf(err, "cannot access resource file %s", name)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		_ = file.Close()
		return nil, errors.Wrapf(err, "cannot access resource file %s", name)
	}

	return data, file.Close()
}

// TemplateResource loads a file resource as go template and processes it using the given parameters.
func TemplateResource(name string, params interface{}) (string, error) {
	rawData, err := ResourceAsString(name)
	if err != nil {
		return "", err
	}
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
	if _, err := assets.Open(fixPath(dirName)); err != nil {
		return false
	}
	return true
}

// WithPrefix lists all file names that begins with the give path prefix
// If pathPrefix is a path of directories then be sure to end it with a '/'.
func WithPrefix(pathPrefix string) ([]string, error) {
	dirPath := filepath.Dir(pathPrefix)

	paths, err := Resources(dirPath)
	if err != nil {
		return nil, err
	}

	var res []string
	for i := range paths {
		path := fixPath(paths[i])
		if result, _ := filepath.Match(pathPrefix+"*", path); result {
			res = append(res, path)
		}
	}

	return res, nil
}

// Resources lists all file names in the given path (starts with '/').
func Resources(dirName string) ([]string, error) {
	dir, err := assets.Open(fixPath(dirName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, errors.Wrapf(err, "error while listing resource files %s", dirName)
	}

	info, err := dir.Stat()
	if err != nil {
		return nil, dir.Close()
	}
	if !info.IsDir() {
		util.CloseQuietly(dir)
		return nil, errors.Wrapf(err, "location %s is not a directory", dirName)
	}

	files, err := dir.Readdir(-1)
	if err != nil {
		util.CloseQuietly(dir)
		return nil, errors.Wrapf(err, "error while listing files on directory %s", dirName)
	}

	var res []string
	for _, f := range files {
		if !f.IsDir() {
			res = append(res, filepath.Join(dirName, f.Name()))
		}
	}

	return res, dir.Close()
}

func fixPath(path string) string {
	if runtime.GOOS == "windows" {
		path = strings.ReplaceAll(path, "\\", "/")
	}

	return path
}

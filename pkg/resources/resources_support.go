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
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/apache/camel-k/v2/pkg/util"
)

// ResourceAsString returns the named resource content as string.
// Warning : do not remove the go:generate line as it is used to update the resources file in case of CRD changes.
//
//go:generate go run ../../cmd/util/vfs-gen resources config
func ResourceAsString(name string) (string, error) {
	data, err := Resource(name)
	return string(data), err
}

// Resource provides an easy way to access to embedded assets.
func Resource(name string) ([]byte, error) {
	name = strings.Trim(name, " ")
	name = filepath.ToSlash(name)
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}

	file, err := openAsset(name)
	if err != nil {
		return nil, fmt.Errorf("cannot access resource file %s: %w", name, err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("cannot access resource file %s: %w", name, err)
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
	if _, err := openAsset(dirName); err != nil {
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
		path := filepath.ToSlash(paths[i])
		if result, _ := filepath.Match(pathPrefix+"*", path); result {
			res = append(res, path)
		}
	}

	return res, nil
}

// Resources lists all file names in the given path (starts with '/').
func Resources(dirName string) ([]string, error) {
	dirName = filepath.ToSlash(dirName)
	dir, err := openAsset(dirName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("error while listing resource files %s: %w", dirName, err)
	}

	info, err := dir.Stat()
	if err != nil {
		return nil, dir.Close()
	}
	if !info.IsDir() {
		util.CloseQuietly(dir)
		return nil, fmt.Errorf("location %s is not a directory: %w", dirName, err)
	}

	files, err := dir.Readdir(-1)
	if err != nil {
		util.CloseQuietly(dir)
		return nil, fmt.Errorf("error while listing files on directory %s: %w", dirName, err)
	}

	var res []string
	for _, f := range files {
		if !f.IsDir() {
			res = append(res, path.Join(dirName, f.Name()))
		}
	}

	return res, dir.Close()
}

func openAsset(path string) (http.File, error) {
	return assets.Open(filepath.ToSlash(path))
}

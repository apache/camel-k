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
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/scylladb/go-set/strset"
	yaml2 "gopkg.in/yaml.v2"
)

/// Directories and file names:

// MavenWorkingDirectory --  Directory used by Maven for an invocation of the kamel local command. By default a temporary folder will be used.
var MavenWorkingDirectory string = ""

// DefaultDependenciesDirectoryName --
const DefaultDependenciesDirectoryName = "dependencies"

// DefaultPropertiesDirectoryName --
const DefaultPropertiesDirectoryName = "properties"

// DefaultRoutesDirectoryName --
const DefaultRoutesDirectoryName = "routes"

// DefaultWorkingDirectoryName --
const DefaultWorkingDirectoryName = "workspace"

// ContainerDependenciesDirectory --
var ContainerDependenciesDirectory = "/deployments/dependencies"

// ContainerPropertiesDirectory --
var ContainerPropertiesDirectory = "/etc/camel/conf.d"

// ContainerRoutesDirectory --
var ContainerRoutesDirectory = "/etc/camel/sources"

// ContainerResourcesDirectory --
var ContainerResourcesDirectory = "/etc/camel/resources"

// StringSliceJoin --
func StringSliceJoin(slices ...[]string) []string {
	size := 0

	for _, s := range slices {
		size += len(s)
	}

	result := make([]string, 0, size)

	for _, s := range slices {
		result = append(result, s...)
	}

	return result
}

// StringSliceContains --
func StringSliceContains(slice []string, items []string) bool {
	for i := 0; i < len(items); i++ {
		if !StringSliceExists(slice, items[i]) {
			return false
		}
	}

	return true
}

// StringSliceExists --
func StringSliceExists(slice []string, item string) bool {
	for i := 0; i < len(slice); i++ {
		if slice[i] == item {
			return true
		}
	}

	return false
}

// StringSliceContainsAnyOf --
func StringSliceContainsAnyOf(slice []string, items ...string) bool {
	for i := 0; i < len(slice); i++ {
		for j := 0; j < len(items); j++ {
			if strings.Contains(slice[i], items[j]) {
				return true
			}
		}
	}

	return false
}

// StringSliceUniqueAdd append the given item if not already present in the slice
func StringSliceUniqueAdd(slice *[]string, item string) bool {
	if slice == nil {
		newSlice := make([]string, 0)
		slice = &newSlice
	}
	for _, i := range *slice {
		if i == item {
			return false
		}
	}

	*slice = append(*slice, item)

	return true
}

// StringSliceUniqueConcat append all the items of the "items" slice if they are not already present in the slice
func StringSliceUniqueConcat(slice *[]string, items []string) bool {
	changed := false
	for _, item := range items {
		if StringSliceUniqueAdd(slice, item) {
			changed = true
		}
	}
	return changed
}

// EncodeXML --
func EncodeXML(content interface{}) ([]byte, error) {
	w := &bytes.Buffer{}
	w.WriteString(xml.Header)

	e := xml.NewEncoder(w)
	e.Indent("", "  ")

	err := e.Encode(content)
	if err != nil {
		return []byte{}, err
	}

	return w.Bytes(), nil
}

// CopyFile --
func CopyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	err = os.MkdirAll(path.Dir(dst), 0777)
	if err != nil {
		return 0, err
	}

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

// WriteFileWithContent --
func WriteFileWithContent(buildDir string, relativePath string, content []byte) error {
	filePath := path.Join(buildDir, relativePath)
	fileDir := path.Dir(filePath)
	// Create dir if not present
	err := os.MkdirAll(fileDir, 0777)
	if err != nil {
		return errors.Wrap(err, "could not create dir for file "+relativePath)
	}
	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return errors.Wrap(err, "could not create file "+relativePath)
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		return errors.Wrap(err, "could not write to file "+relativePath)
	}
	return nil
}

// WriteFileWithBytesMarshallerContent --
func WriteFileWithBytesMarshallerContent(buildDir string, relativePath string, content BytesMarshaller) error {
	data, err := content.MarshalBytes()
	if err != nil {
		return err
	}

	return WriteFileWithContent(buildDir, relativePath, data)
}

// FindAllDistinctStringSubmatch --
func FindAllDistinctStringSubmatch(data string, regexps ...*regexp.Regexp) []string {
	submatchs := strset.New()

	for _, reg := range regexps {
		hits := reg.FindAllStringSubmatch(data, -1)
		for _, hit := range hits {
			if len(hit) > 1 {
				for _, match := range hit[1:] {
					submatchs.Add(match)
				}
			}
		}
	}
	return submatchs.List()
}

// FindNamedMatches ---
func FindNamedMatches(expr string, str string) map[string]string {
	regex := regexp.MustCompile(expr)
	match := regex.FindStringSubmatch(str)

	results := map[string]string{}
	for i, name := range match {
		results[regex.SubexpNames()[i]] = name
	}
	return results
}

// FileExists --
func FileExists(name string) (bool, error) {
	info, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}

	return !info.IsDir(), err
}

// DirectoryExists --
func DirectoryExists(directory string) (bool, error) {
	info, err := os.Stat(directory)
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return info.IsDir(), nil
}

// CreateDirectory --
func CreateDirectory(directory string) error {
	// If directory does not exist, create it.
	directoryExists, err := DirectoryExists(directory)
	if err != nil {
		return err
	}

	if !directoryExists {
		err := os.MkdirAll(directory, 0777)
		if err != nil {
			return err
		}
	}

	return nil
}

// BytesMarshaller --
type BytesMarshaller interface {
	MarshalBytes() ([]byte, error)
}

// SortedMapKeys --
func SortedMapKeys(m map[string]interface{}) []string {
	res := make([]string, len(m))
	i := 0
	for k := range m {
		res[i] = k
		i++
	}
	sort.Strings(res)
	return res
}

// SortedStringMapKeys --
func SortedStringMapKeys(m map[string]string) []string {
	res := make([]string, len(m))
	i := 0
	for k := range m {
		res[i] = k
		i++
	}
	sort.Strings(res)
	return res
}

// DependenciesToJSON --
func DependenciesToJSON(list []string) ([]byte, error) {
	jsondata := map[string]interface{}{}
	jsondata["dependencies"] = list
	return json.Marshal(jsondata)
}

// DependenciesToYAML --
func DependenciesToYAML(list []string) ([]byte, error) {
	data, err := DependenciesToJSON(list)
	if err != nil {
		return nil, err
	}

	return JSONToYAML(data)
}

// JSONToYAML --
func JSONToYAML(src []byte) ([]byte, error) {
	jsondata := map[string]interface{}{}
	err := json.Unmarshal(src, &jsondata)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json: %v", err)
	}
	yamldata, err := yaml2.Marshal(&jsondata)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to yaml: %v", err)
	}

	return yamldata, nil
}

// WriteToFile --
func WriteToFile(filePath string, fileContents string) error {
	err := ioutil.WriteFile(filePath, []byte(fileContents), 0777)
	if err != nil {
		return errors.Errorf("error writing file: %v", filePath)
	}

	// All went well, return true.
	return nil
}

/// Local directories:

// GetLocalPropertiesDir -- <mavenWorkingDirectory>/properties
func GetLocalPropertiesDir() string {
	return path.Join(MavenWorkingDirectory, DefaultPropertiesDirectoryName)
}

// GetLocalDependenciesDir --<mavenWorkingDirectory>/dependencies
func GetLocalDependenciesDir() string {
	return path.Join(MavenWorkingDirectory, DefaultDependenciesDirectoryName)
}

// GetLocalRoutesDir -- <mavenWorkingDirectory>/routes
func GetLocalRoutesDir() string {
	return path.Join(MavenWorkingDirectory, DefaultRoutesDirectoryName)
}

// CreateLocalPropertiesDirectory --
func CreateLocalPropertiesDirectory() error {
	// Do not create a directory unless the maven directory contains a valid value.
	if MavenWorkingDirectory == "" {
		return nil
	}

	directoryExists, err := DirectoryExists(GetLocalPropertiesDir())
	if err != nil {
		return err
	}

	if !directoryExists {
		err := os.MkdirAll(GetLocalPropertiesDir(), 0777)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateLocalDependenciesDirectory --
func CreateLocalDependenciesDirectory() error {
	// Do not create a directory unless the maven directory contains a valid value.
	if MavenWorkingDirectory == "" {
		return nil
	}

	directoryExists, err := DirectoryExists(GetLocalDependenciesDir())
	if err != nil {
		return err
	}

	if !directoryExists {
		err := os.MkdirAll(GetLocalDependenciesDir(), 0777)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateLocalRoutesDirectory --
func CreateLocalRoutesDirectory() error {
	// Do not create a directory unless the maven directory contains a valid value.
	if MavenWorkingDirectory == "" {
		return nil
	}

	directoryExists, err := DirectoryExists(GetLocalRoutesDir())
	if err != nil {
		return err
	}

	if !directoryExists {
		err := os.MkdirAll(GetLocalRoutesDir(), 0777)
		if err != nil {
			return err
		}
	}
	return nil
}

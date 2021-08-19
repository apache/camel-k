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

	yaml2 "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
	"github.com/scylladb/go-set/strset"
)

// Directories and file names:

// MavenWorkingDirectory is the directory used by Maven for an invocation of the kamel local command.
// By default, a temporary folder will be used.
var MavenWorkingDirectory = ""

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

// QuarkusDependenciesBaseDirectory --
var QuarkusDependenciesBaseDirectory = "/quarkus-app"

// ListOfLazyEvaluatedEnvVars -- List of unevaluated environment variables.
// These are sensitive values or values that may have different values depending on
// where the integration is run (locally vs. the cloud). These environment variables
// are evaluated at the time of the integration invocation.
var ListOfLazyEvaluatedEnvVars = []string{}

// CLIEnvVars -- List of CLI provided environment variables. They take precedence over
// any environment variables with the same name.
var CLIEnvVars = make([]string, 0)

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

func StringSliceContains(slice []string, items []string) bool {
	for i := 0; i < len(items); i++ {
		if !StringSliceExists(slice, items[i]) {
			return false
		}
	}

	return true
}

func StringSliceExists(slice []string, item string) bool {
	for i := 0; i < len(slice); i++ {
		if slice[i] == item {
			return true
		}
	}

	return false
}

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

// StringSliceUniqueAdd appends the given item if not already present in the slice
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

// StringSliceUniqueConcat appends all the items of the "items" slice if they are not already present in the slice
func StringSliceUniqueConcat(slice *[]string, items []string) bool {
	changed := false
	for _, item := range items {
		if StringSliceUniqueAdd(slice, item) {
			changed = true
		}
	}
	return changed
}

func SubstringFrom(s string, substr string) string {
	index := strings.Index(s, substr)
	if index != -1 {
		return s[index:]
	}
	return ""
}

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

func CopyFile(src, dst string) (int64, error) {
	stat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !stat.Mode().IsRegular() {
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

	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, stat.Mode())
	if err != nil {
		return 0, err
	}

	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

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

func WriteFileWithBytesMarshallerContent(buildDir string, relativePath string, content BytesMarshaller) error {
	data, err := content.MarshalBytes()
	if err != nil {
		return err
	}

	return WriteFileWithContent(buildDir, relativePath, data)
}

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

func FindNamedMatches(expr string, str string) map[string]string {
	regex := regexp.MustCompile(expr)
	match := regex.FindStringSubmatch(str)

	results := map[string]string{}
	for i, name := range match {
		results[regex.SubexpNames()[i]] = name
	}
	return results
}

func FileExists(name string) (bool, error) {
	info, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}

	return !info.IsDir(), err
}

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

func DirectoryEmpty(directory string) (bool, error) {
	f, err := os.Open(directory)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func CreateDirectory(directory string) error {
	if directory != "" {
		// If directory does not exist, create it
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
	}

	return nil
}

type BytesMarshaller interface {
	MarshalBytes() ([]byte, error)
}

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

// CopyMap clones a map of strings
func CopyMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	dest := make(map[string]string, len(source))
	for k, v := range source {
		dest[k] = v
	}
	return dest
}

func DependenciesToJSON(list []string) ([]byte, error) {
	jsondata := map[string]interface{}{}
	jsondata["dependencies"] = list
	return json.Marshal(jsondata)
}

func DependenciesToYAML(list []string) ([]byte, error) {
	data, err := DependenciesToJSON(list)
	if err != nil {
		return nil, err
	}

	return JSONToYAML(data)
}

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

func WriteToFile(filePath string, fileContents string) error {
	err := ioutil.WriteFile(filePath, []byte(fileContents), 0777)
	if err != nil {
		return errors.Errorf("error writing file: %v", filePath)
	}

	return nil
}

// Local directories:

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

func GetEnvironmentVariable(variable string) (string, error) {
	value, isPresent := os.LookupEnv(variable)
	if !isPresent {
		return "", errors.Errorf("environment variable %v does not exist", variable)
	}

	if value == "" {
		return "", errors.Errorf("environment variable %v is not set", variable)
	}

	return value, nil
}

// EvaluateCLIAndLazyEnvVars creates a list of environment
// variables with entries VAR=value that can be passed when running the integration.
func EvaluateCLIAndLazyEnvVars() ([]string, error) {
	evaluatedEnvVars := []string{}

	// Add CLI environment variables
	setEnvVars := []string{}
	for _, cliEnvVar := range CLIEnvVars {
		// Mark variable name as set.
		varAndValue := strings.Split(cliEnvVar, "=")
		setEnvVars = append(setEnvVars, varAndValue[0])

		evaluatedEnvVars = append(evaluatedEnvVars, cliEnvVar)
	}

	// Add lazily evaluated environment variables if they have not
	// already been set via the CLI --env option.
	for _, lazyEnvVar := range ListOfLazyEvaluatedEnvVars {
		alreadySet := false
		for _, setEnvVar := range setEnvVars {
			if setEnvVar == lazyEnvVar {
				alreadySet = true
				break
			}
		}

		if !alreadySet {
			value, err := GetEnvironmentVariable(lazyEnvVar)
			if err != nil {
				return nil, err
			}
			evaluatedEnvVars = append(evaluatedEnvVars, lazyEnvVar+"="+value)
		}
	}

	return evaluatedEnvVars, nil
}

func CopyIntegrationFilesToDirectory(files []string, directory string) ([]string, error) {
	// Create directory if one does not already exist
	err := CreateDirectory(directory)
	if err != nil {
		return nil, err
	}

	// Copy files to new location. Also create the list with relocated files.
	relocatedFilesList := make([]string, len(files))
	for _, filePath := range files {
		newFilePath := path.Join(directory, path.Base(filePath))
		_, err := CopyFile(filePath, newFilePath)
		if err != nil {
			return relocatedFilesList, err
		}
		relocatedFilesList = append(relocatedFilesList, newFilePath)
	}

	return relocatedFilesList, nil
}

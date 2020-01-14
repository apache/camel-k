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
	"encoding/xml"
	"os"
	"os/signal"
	"path"
	"regexp"
	"sort"
	"syscall"

	"github.com/scylladb/go-set/strset"

	corev1 "k8s.io/api/core/v1"

	"github.com/pkg/errors"
)

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

// WaitForSignal --
func WaitForSignal(sig chan os.Signal, exit func(int)) {
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGPIPE)
	go func() {
		s := <-sig
		switch s {
		case syscall.SIGINT, syscall.SIGTERM:
			exit(130) // Ctrl+c
		case syscall.SIGPIPE:
			exit(0)
		}
		exit(1)
	}()
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

// LookupEnvVar --
func LookupEnvVar(vars []corev1.EnvVar, name string) *corev1.EnvVar {
	for _, e := range vars {
		if e.Name == name {
			ev := e
			return &ev
		}
	}

	return nil
}

// FindAllDistinctStringSubmatch ..
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

// FileExists --
func FileExists(name string) (bool, error) {
	info, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}

	return !info.IsDir(), err
}

// MarshalBytes --
type BytesMarshaller interface {
	MarshalBytes() ([]byte, error)
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

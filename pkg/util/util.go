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
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	io2 "github.com/apache/camel-k/v2/pkg/util/io"

	"github.com/apache/camel-k/v2/pkg/util/sets"
	"go.uber.org/multierr"

	yaml2 "gopkg.in/yaml.v2"
)

// ListOfLazyEvaluatedEnvVars -- List of unevaluated environment variables.
// These are sensitive values or values that may have different values depending on
// where the integration is run (locally vs. the cloud). These environment variables
// are evaluated at the time of the integration invocation.
var ListOfLazyEvaluatedEnvVars []string

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
	for i := range items {
		if !StringSliceExists(slice, items[i]) {
			return false
		}
	}

	return true
}

func StringSliceExists(slice []string, item string) bool {
	for i := range slice {
		if slice[i] == item {
			return true
		}
	}

	return false
}

func StringSliceContainsAnyOf(slice []string, items ...string) bool {
	for i := range slice {
		for j := range items {
			if strings.Contains(slice[i], items[j]) {
				return true
			}
		}
	}

	return false
}

// StringSliceUniqueAdd appends the given item if not already present in the slice.
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

// StringSliceUniqueConcat appends all the items of the "items" slice if they are not already present in the slice.
func StringSliceUniqueConcat(slice *[]string, items []string) bool {
	changed := false
	for _, item := range items {
		if StringSliceUniqueAdd(slice, item) {
			changed = true
		}
	}

	return changed
}

func SubstringBefore(s string, substr string) string {
	index := strings.LastIndex(s, substr)
	if index != -1 {
		return s[:index]
	}

	return ""
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var randomSource = rand.NewSource(time.Now().UnixNano())

func RandomString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i, cache, remain := n-1, randomSource.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = randomSource.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}

func EncodeXMLWithoutHeader(content interface{}) ([]byte, error) {
	return encodeXML(content, "")
}

func EncodeXML(content interface{}) ([]byte, error) {

	return encodeXML(content, xml.Header)
}

func encodeXML(content interface{}, xmlHeader string) ([]byte, error) {
	w := &bytes.Buffer{}
	w.WriteString(xmlHeader)

	e := xml.NewEncoder(w)
	e.Indent("", "  ")

	if err := e.Encode(content); err != nil {
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

	source, err := Open(src)
	if err != nil {
		return 0, err
	}

	defer func() {
		err = Close(err, source)
	}()

	// we need to have group and other to be able to access the directory as the user
	// in the container may not be the same as the one owning the files
	//
	// #nosec G301
	if err = os.MkdirAll(path.Dir(dst), io2.FilePerm755); err != nil {
		return 0, err
	}

	destination, err := OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, stat.Mode())
	if err != nil {
		return 0, err
	}

	defer func() {
		err = Close(err, destination)
	}()

	nBytes, err := io.Copy(destination, source)

	return nBytes, err
}

func WriteFileWithBytesMarshallerContent(basePath string, filePath string, content BytesMarshaller) error {
	data, err := content.MarshalBytes()
	if err != nil {
		return err
	}

	return WriteFileWithContent(filepath.Join(basePath, filePath), data)
}

func FindAllDistinctStringSubmatch(data string, regexps ...*regexp.Regexp) []string {
	submatchs := sets.NewSet()

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
	f, err := Open(directory)
	if err != nil {
		return false, err
	}

	defer func() {
		err = Close(err, f)
	}()

	ok := false
	if _, err = f.Readdirnames(1); errors.Is(err, io.EOF) {
		ok = true
	}

	return ok, err
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

// CopyMap clones a map of strings.
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

func JSONToYAML(src []byte) ([]byte, error) {
	mapdata, err := JSONToMap(src)
	if err != nil {
		return nil, err
	}

	return MapToYAML(mapdata)
}

func JSONToMap(src []byte) (map[string]interface{}, error) {
	jsondata := map[string]interface{}{}
	err := json.Unmarshal(src, &jsondata)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json: %w", err)
	}

	return jsondata, nil
}

func MapToYAML(src map[string]interface{}) ([]byte, error) {
	yamldata, err := yaml2.Marshal(&src)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to yaml: %w", err)
	}

	return yamldata, nil
}

func GetEnvironmentVariable(variable string) (string, error) {
	value, isPresent := os.LookupEnv(variable)
	if !isPresent {
		return "", fmt.Errorf("environment variable %v does not exist", variable)
	}

	if value == "" {
		return "", fmt.Errorf("environment variable %v is not set", variable)
	}

	return value, nil
}

// Open a safe wrapper of os.Open.
func Open(name string) (*os.File, error) {
	return os.Open(filepath.Clean(name))
}

// OpenFile a safe wrapper of os.OpenFile.
func OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	// #nosec G304
	return os.OpenFile(filepath.Clean(name), flag, perm)
}

// ReadFile a safe wrapper of os.ReadFile.
func ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filepath.Clean(filename))
}

func Close(err error, closer io.Closer) error {
	return multierr.Append(err, closer.Close())
}

// CloseQuietly unconditionally close an io.Closer
// It should not be used to replace the Close statement(s).
func CloseQuietly(closer io.Closer) {
	_ = closer.Close()
}

// WithFile a safe wrapper to process a file.
func WithFile(name string, flag int, perm os.FileMode, consumer func(file *os.File) error) error {
	// #nosec G304
	file, err := os.OpenFile(filepath.Clean(name), flag, perm)
	if err == nil {
		err = consumer(file)
	}

	return Close(err, file)
}

// WithFileReader a safe wrapper to process a file.
func WithFileReader(name string, consumer func(reader io.Reader) error) error {
	// #nosec G304
	file, err := os.Open(filepath.Clean(name))
	if err == nil {
		err = consumer(file)
	}

	return Close(err, file)
}

// WithFileContent a safe wrapper to process a file content.
func WithFileContent(name string, consumer func(file *os.File, data []byte) error) error {
	return WithFile(name, os.O_RDWR|os.O_CREATE, io2.FilePerm644, func(file *os.File) error {
		content, err := ReadFile(name)
		if err != nil {
			return err
		}

		return consumer(file, content)
	})
}

// WriteFileWithContent a safe wrapper to write content to a file.
func WriteFileWithContent(filePath string, content []byte) error {
	fileDir := path.Dir(filePath)

	// Create dir if not present
	err := os.MkdirAll(fileDir, io2.FilePerm755)
	if err != nil {
		return fmt.Errorf("could not create dir for file "+filePath+": %w", err)
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("could not create file "+filePath+": %w", err)
	}

	_, err = file.Write(content)
	if err != nil {
		err = fmt.Errorf("could not write to file "+filePath+": %w", err)
	}

	return Close(err, file)
}

// WithTempDir a safe wrapper to deal with temporary directories.
func WithTempDir(pattern string, consumer func(string) error) error {
	tmpDir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return err
	}

	consumerErr := consumer(tmpDir)
	removeErr := os.RemoveAll(tmpDir)

	return multierr.Append(consumerErr, removeErr)
}

var propertyRegex = regexp.MustCompile("'.+'|\".+\"|[^.]+")

// ConfigTreePropertySplit Parses a property spec and returns its parts.
func ConfigTreePropertySplit(property string) []string {
	var res = make([]string, 0)
	initialParts := propertyRegex.FindAllString(property, -1)
	for _, p := range initialParts {
		cur := trimQuotes(p)
		var tmp []string
		for strings.Contains(cur[1:], "[") && strings.HasSuffix(cur, "]") {
			pos := strings.LastIndex(cur, "[")
			tmp = append(tmp, cur[pos:])
			cur = cur[0:pos]
		}
		if len(cur) > 0 {
			tmp = append(tmp, cur)
		}
		for i := len(tmp) - 1; i >= 0; i-- {
			res = append(res, tmp[i])
		}
	}
	return res
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if c := s[len(s)-1]; s[0] == c && (c == '"' || c == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// NavigateConfigTree switch to the element in the tree represented by the "nodes" spec and creates intermediary
// nodes if missing. Nodes specs starting with "[" and ending in "]" are treated as slice indexes.
func NavigateConfigTree(current interface{}, nodes []string) (interface{}, error) {
	if len(nodes) == 0 {
		return current, nil
	}
	isSlice := func(idx int) bool {
		if idx >= len(nodes) {
			return false
		}
		return strings.HasPrefix(nodes[idx], "[") && strings.HasSuffix(nodes[idx], "]")
	}
	makeNext := func() interface{} {
		if isSlice(1) {
			slice := make([]interface{}, 0)
			return &slice
		}
		return make(map[string]interface{})
	}
	switch c := current.(type) {
	case map[string]interface{}:
		var next interface{}
		if n, ok := c[nodes[0]]; ok {
			next = n
		} else {
			next = makeNext()
			c[nodes[0]] = next
		}
		return NavigateConfigTree(next, nodes[1:])
	case *[]interface{}:
		if !isSlice(0) {
			return nil, fmt.Errorf("attempting to set map value %q into a slice", nodes[0])
		}
		pos, err := strconv.Atoi(nodes[0][1 : len(nodes[0])-1])
		if err != nil {
			return nil, fmt.Errorf("value %q inside brackets is not numeric: %w", nodes[0], err)
		}
		var next interface{}
		if len(*c) > pos && (*c)[pos] != nil {
			next = (*c)[pos]
		} else {
			next = makeNext()
			for len(*c) <= pos {
				*c = append(*c, nil)
			}
			(*c)[pos] = next
		}
		return NavigateConfigTree(next, nodes[1:])
	default:
		return nil, errors.New("invalid node type in configuration")
	}
}

// IToInt32 attempts to convert safely an int to an int32.
func IToInt32(x int) (*int32, error) {
	if x < math.MinInt32 || x > math.MaxInt32 {
		return nil, fmt.Errorf("integer overflow casting to int32 type")
	}
	casted := int32(x)

	return &casted, nil
}

// IToInt8 attempts to convert safely an int to an int8.
func IToInt8(x int) (*int8, error) {
	if x < math.MinInt8 || x > math.MaxInt8 {
		return nil, fmt.Errorf("integer overflow casting to int8 type")
	}
	casted := int8(x)

	return &casted, nil
}

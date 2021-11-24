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

package property

import (
	"bytes"
	"strings"

	"github.com/magiconair/properties"
	"github.com/pkg/errors"
)

// EncodePropertyFileEntry converts the given key/value pair into a .properties file entry.
func EncodePropertyFileEntry(key, value string) (string, error) {
	p := properties.NewProperties()
	p.DisableExpansion = true
	if _, _, err := p.Set(key, value); err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	if _, err := p.Write(buf, properties.UTF8); err != nil {
		return "", err
	}
	pair := strings.TrimSuffix(buf.String(), "\n")
	return pair, nil
}

// EncodePropertyFile encodes a property map into a .properties file.
func EncodePropertyFile(sourceProperties map[string]string) (string, error) {
	props := properties.LoadMap(sourceProperties)
	props.DisableExpansion = true
	props.Sort()
	buf := new(bytes.Buffer)
	_, err := props.Write(buf, properties.UTF8)
	if err != nil {
		return "", errors.Wrapf(err, "could not compute application properties")
	}
	return buf.String(), nil
}

// SplitPropertyFileEntry splits an encoded property into key/value pair, without decoding the content.
func SplitPropertyFileEntry(entry string) (string, string) {
	pair := strings.SplitN(entry, "=", 2)
	var k, v string
	if len(pair) >= 1 {
		k = strings.TrimSpace(pair[0])
	}
	if len(pair) == 2 {
		v = strings.TrimSpace(pair[1])
	}
	return k, v
}

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

package gzip

import (
	"bytes"
	g "compress/gzip"
	"encoding/base64"
	"io"
	"io/ioutil"

	"github.com/apache/camel-k/pkg/util"
)

// Compress --.
func Compress(buffer io.Writer, data []byte) error {
	gz := g.NewWriter(buffer)

	if _, err := gz.Write(data); err != nil {
		return err
	}
	if err := gz.Flush(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}

	return nil
}

// CompressBase64 --.
func CompressBase64(data []byte) ([]byte, error) {
	var b bytes.Buffer

	if err := Compress(&b, data); err != nil {
		return []byte{}, err
	}

	return []byte(base64.StdEncoding.EncodeToString(b.Bytes())), nil
}

// Uncompress --.
func Uncompress(buffer io.Writer, data []byte) error {
	b := bytes.NewBuffer(data)
	gz, err := g.NewReader(b)
	if err != nil {
		return err
	}

	data, err = ioutil.ReadAll(gz)
	if err != nil {
		util.CloseQuietly(gz)
		return err
	}

	_, err = buffer.Write(data)
	if err != nil {
		util.CloseQuietly(gz)
		return err
	}

	return gz.Close()
}

// UncompressBase64 --.
func UncompressBase64(data []byte) ([]byte, error) {
	d, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return []byte{}, err
	}

	var b bytes.Buffer
	err = Uncompress(&b, d)
	if err != nil {
		return []byte{}, err
	}

	return b.Bytes(), nil
}

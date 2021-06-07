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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPropertyEncoding(t *testing.T) {
	enc, err := EncodePropertyFileEntry("a", "b")
	assert.NoError(t, err)
	assert.Equal(t, "a = b", enc)
	enc, err = EncodePropertyFileEntry("c", "d\ne")
	assert.NoError(t, err)
	assert.Equal(t, "c = d\\ne", enc)
}

func TestPropertyFileEncoding(t *testing.T) {
	props := map[string]string{
		"c": "d\ne",
		"a": "b",
	}
	enc, err := EncodePropertyFile(props)
	assert.NoError(t, err)
	assert.Equal(t, "a = b\nc = d\\ne\n", enc)
}

func TestSplitPropertyEntry(t *testing.T) {
	entry := "c = d\\ne"
	k, v := SplitPropertyFileEntry(entry)
	assert.Equal(t, "c", k)
	assert.Equal(t, "d\\ne", v)
}

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

package sets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	s := NewSet()
	assert.True(t, s.IsEmpty())
	s.Add("hello")
	assert.False(t, s.IsEmpty())
	assert.Equal(t, 1, s.Size())
	s.Add("hello another")
	assert.Equal(t, 2, s.Size())
	s.Add("hello")
	assert.Equal(t, 2, s.Size())

	s2 := NewSet()
	s2.Add("hello")
	s2.Add("ciao")
	assert.Equal(t, 2, s2.Size())

	s3 := Union(s, s2)
	assert.Equal(t, 3, s3.Size())
	assert.Contains(t, s3.List(), "hello", "ciao", "hello another")
	assert.True(t, s3.Has("ciao"))
	assert.False(t, s3.Has("bye"))

	s.Merge(s2)
	assert.Equal(t, 3, s.Size())
	assert.Contains(t, s.List(), "hello", "ciao", "hello another")

	processedList := make([]string, 0)

	s.Each(func(item string) bool {
		processedList = append(processedList, "processed:"+item)
		return true
	})

	assert.Contains(t, processedList, "processed:hello", "processed:ciao", "processed:hello another")
}

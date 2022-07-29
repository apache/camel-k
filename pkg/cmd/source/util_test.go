//go:build !darwin
// +build !darwin

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

package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCorrectFileValuesButNotFound(t *testing.T) {
	value1, err1 := IsLocalAndFileExists("c:\\test")
	value2, err2 := IsLocalAndFileExists("path/to/file")

	// they are all not found, but it must not panic
	assert.Nil(t, err1)
	assert.False(t, value1)
	assert.Nil(t, err2)
	assert.False(t, value2)
}

func TestPermissionDenied(t *testing.T) {
	value, err := IsLocalAndFileExists("/root/test")
	// must not panic because a permission error
	assert.NotNil(t, err)
	assert.False(t, value)
}

func TestSupportedScheme(t *testing.T) {
	gistValue, err1 := IsLocalAndFileExists("gist:some/gist/resource")
	githubValue, err2 := IsLocalAndFileExists("github:some/github/resource")
	httpValue, err3 := IsLocalAndFileExists("http://some/http/resource")
	httpsValue, err4 := IsLocalAndFileExists("https://some/https/resource")

	assert.Nil(t, err1)
	assert.False(t, gistValue)
	assert.Nil(t, err2)
	assert.False(t, githubValue)
	assert.Nil(t, err3)
	assert.False(t, httpValue)
	assert.Nil(t, err4)
	assert.False(t, httpsValue)
}

func TestUnSupportedScheme(t *testing.T) {
	value, err := IsLocalAndFileExists("bad_scheme:some/bad/resource")
	// must not report an error
	assert.Nil(t, err)
	assert.False(t, value)
}

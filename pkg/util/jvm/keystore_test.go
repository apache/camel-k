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

package jvm

import (
	"context"
	"os"
	"testing"

	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestGenerateKeystore(t *testing.T) {

	// No Data
	var data [][]byte
	ctx := context.Background()
	err := GenerateKeystore(ctx, "", "/tmp/keystore", NewKeystorePassword(), data)
	assert.Nil(t, err)

	// Correct input
	data = [][]byte{{0}, {1}}
	err = GenerateKeystore(ctx, "", "/tmp/keystore", NewKeystorePassword(), data)
	assert.NotNil(t, err)

	// Incorrect input format
	err = GenerateKeystore(ctx, "", "/tmp/keystore", "", data)
	assert.NotNil(t, err)

	testFileExists, _ := util.FileExists("/tmp/keystore")
	if testFileExists {
		os.Remove("/tmp/keystore")
	}
}

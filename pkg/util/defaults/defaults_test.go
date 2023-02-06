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

package defaults

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultBaseImage(t *testing.T) {
	assert.NotEmpty(t, BaseImage())
}

func TestOverriddenBaseImage(t *testing.T) {
	env := "RELATED_IMAGE_BASE"
	oldEnvVal := os.Getenv(env)
	overriddenImage := "xxx"
	assert.NoError(t, os.Setenv(env, overriddenImage))
	assert.Equal(t, overriddenImage, BaseImage())
	assert.NoError(t, os.Setenv(env, oldEnvVal))
}

func TestOverriddenInstallDefaultKamelets(t *testing.T) {
	env := "KAMEL_INSTALL_DEFAULT_KAMELETS"
	oldEnvVal := os.Getenv(env)
	assert.NoError(t, os.Setenv(env, strconv.FormatBool(false)))
	assert.False(t, InstallDefaultKamelets())
	assert.NoError(t, os.Setenv(env, strconv.FormatBool(true)))
	assert.True(t, InstallDefaultKamelets())
	assert.NoError(t, os.Setenv(env, "wrongval"))
	assert.False(t, InstallDefaultKamelets())
	assert.NoError(t, os.Setenv(env, oldEnvVal))
}

func TestOverriddenOperatorID(t *testing.T) {
	env := "KAMEL_OPERATOR_ID"
	oldEnvVal := os.Getenv(env)
	overriddenID := "operator-1"
	assert.NoError(t, os.Setenv(env, overriddenID))
	assert.Equal(t, overriddenID, OperatorID())
	assert.NoError(t, os.Setenv(env, oldEnvVal))
}

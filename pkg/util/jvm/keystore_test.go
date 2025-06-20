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
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKeystore(t *testing.T) {
	ksDir := t.TempDir()

	// Nil Data
	var data [][]byte
	ctx := context.Background()
	err := GenerateKeystore(ctx, ksDir, "my-ks", NewKeystorePassword(), data)
	require.NoError(t, err)

	// Non-Nil Data
	data = [][]byte{{0}, {1}}
	err = GenerateKeystore(ctx, ksDir, "my-ks", NewKeystorePassword(), data)
	require.Error(t, err)
	assert.Equal(t, "keytool error: java.io.IOException: keystore password was incorrect: exit status 1", err.Error())

	// Incorrect password format
	err = GenerateKeystore(ctx, ksDir, "my-ks", "", data)
	require.Error(t, err)
	assert.Equal(t, "Illegal option:  my-ks: exit status 1", err.Error())

}

func TestGenerateValidKeystore(t *testing.T) {
	ksDir := t.TempDir()

	// Generate a valid sample certificate on the fly
	ctx := context.Background()
	args := []string{
		"-genkeypair",
		"-alias", "mycert",
		"-keyalg", "RSA",
		"-keysize", "2048",
		"-validity", "7",
		"-keystore", "my.jks",
		"-storepass", "changeit",
		"-keypass", "changeit",
		"-dname", "CN=example.com, OU=IT, O=MyCompany, L=City, ST=State, C=US",
	}
	cmd := exec.CommandContext(ctx, "keytool", args...)
	cmd.Dir = ksDir
	err := util.RunAndLog(ctx, cmd, loggerInfo, loggerNil)
	require.NoError(t, err)
	args = strings.Fields("-exportcert -alias mycert -keystore my.jks -storepass changeit -rfc -file mycert.crt")
	cmd = exec.CommandContext(ctx, "keytool", args...)
	cmd.Dir = ksDir
	err = util.RunAndLog(ctx, cmd, loggerInfo, loggerNil)
	require.NoError(t, err)

	content, err := util.ReadFile(path.Join(ksDir, "mycert.crt"))
	require.NoError(t, err)

	data := [][]byte{content}
	err = GenerateKeystore(ctx, ksDir, "imported.jks", NewKeystorePassword(), data)
	require.NoError(t, err)

	exists, err := util.FileExists(path.Join(ksDir, "imported.jks"))
	require.NoError(t, err)
	assert.True(t, exists, "missing the generated keystore")
}

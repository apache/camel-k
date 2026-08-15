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

package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSecretConfFromEnvVar(t *testing.T) {
	t.Setenv(RegistrySecretConfEnvVar, "some s3cr3t")
	dir, err := MountSecretRegistryConfig(t.Context(), nil, "ns", "test", "secret")
	require.NoError(t, err)
	assert.NotEmpty(t, dir)

	data, err := os.ReadFile(filepath.Join(dir, jibConfigExtension))
	require.NoError(t, err)
	assert.Equal(t, []byte("some s3cr3t"), data)
}

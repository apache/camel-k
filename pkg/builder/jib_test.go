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

package builder

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJibBuildMavenMissingContext(t *testing.T) {
	args, err := buildJibMavenArgs("missing-dir", "my-image", "my-base-image", true, nil)
	require.Error(t, err)
	assert.Nil(t, args)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestJibBuildMavenArgs(t *testing.T) {
	tmpMvnCtxDir, err := os.MkdirTemp("", "my-build-test")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tmpMvnCtxDir+"/MAVEN_CONTEXT", []byte(`-x some-maven-option`), 0o400))
	args, err := buildJibMavenArgs(tmpMvnCtxDir, "my-image", "my-base-image", true, nil)
	require.NoError(t, err)
	expectedParams := strings.Split(
		fmt.Sprintf("jib:build -Djib.disableUpdateChecks=true -x some-maven-option -P jib -Djib.to.image=my-image "+
			"-Djib.from.image=my-base-image -Djib.baseImageCache=%s -Djib.container.user=1000 -Djib.allowInsecureRegistries=true", tmpMvnCtxDir+"/jib"),
		" ")
	assert.Equal(t, expectedParams, args)
}

func TestJibBuildMavenArgsWithPlatforms(t *testing.T) {
	tmpMvnCtxDir, err := os.MkdirTemp("", "my-build-test")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tmpMvnCtxDir+"/MAVEN_CONTEXT", []byte(`-x some-maven-option`), 0o400))
	args, err := buildJibMavenArgs(tmpMvnCtxDir, "my-image", "my-base-image", true, []string{"amd64", "arm64"})
	require.NoError(t, err)
	expectedParams := strings.Split(
		fmt.Sprintf("jib:build -Djib.disableUpdateChecks=true -x some-maven-option -P jib -Djib.to.image=my-image "+
			"-Djib.from.image=my-base-image -Djib.baseImageCache=%s -Djib.container.user=1000 -Djib.from.platforms=amd64,arm64 -Djib.allowInsecureRegistries=true",
			tmpMvnCtxDir+"/jib"),
		" ")
	assert.Equal(t, expectedParams, args)
}

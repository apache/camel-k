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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/jib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJibBuildMavenArgs(t *testing.T) {
	tmpMvnCtxDir, err := os.MkdirTemp("", "my-build-test")
	require.NoError(t, err)
	args := buildJibMavenArgs(tmpMvnCtxDir, "my-image", "my-base-image", true, nil)
	expectedParams := strings.Split(
		fmt.Sprintf("jib:build -Djib.disableUpdateChecks=true -P jib -Djib.to.image=my-image "+
			"-Djib.from.image=my-base-image -Djib.baseImageCache=%s -Djib.container.user=1000 -Djib.allowInsecureRegistries=true", tmpMvnCtxDir+"/jib"),
		" ")
	assert.Equal(t, expectedParams, args)
}

func TestJibBuildMavenArgsWithPlatforms(t *testing.T) {
	tmpMvnCtxDir, err := os.MkdirTemp("", "my-build-test")
	require.NoError(t, err)
	args := buildJibMavenArgs(tmpMvnCtxDir, "my-image", "my-base-image", true, []string{"amd64", "arm64"})
	expectedParams := strings.Split(
		fmt.Sprintf("jib:build -Djib.disableUpdateChecks=true -P jib -Djib.to.image=my-image "+
			"-Djib.from.image=my-base-image -Djib.baseImageCache=%s -Djib.container.user=1000 -Djib.from.platforms=amd64,arm64 -Djib.allowInsecureRegistries=true",
			tmpMvnCtxDir+"/jib"),
		" ")
	assert.Equal(t, expectedParams, args)
}

func TestInjectJibProfileMissingPom(t *testing.T) {
	tmpMvnCtxDir, err := os.MkdirTemp("", "ck-jib-profile-test")
	require.NoError(t, err)
	builderContext := builderContext{
		C:    context.TODO(),
		Path: tmpMvnCtxDir,
	}
	err = injectJibProfile(&builderContext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

var poms = []string{
	"<project>something</project>",
	"<project>something<profiles><profile></profile></profiles></project>",
	"<no-project-tag></no-project-tag>",
}

func TestInjectJibProfiles(t *testing.T) {
	tmpMvnCtxDir, err := os.MkdirTemp("", "ck-jib-profile-test")
	require.NoError(t, err)
	builderContext := builderContext{
		C:    context.TODO(),
		Path: tmpMvnCtxDir,
	}

	for _, p := range poms {
		pom := filepath.Join(tmpMvnCtxDir, "maven", "pom.xml")
		err = util.WriteFileWithContent(pom, []byte(p))
		require.NoError(t, err)
		err = injectJibProfile(&builderContext)
		require.NoError(t, err)

		newPom, err := util.ReadFile(pom)
		require.NoError(t, err)
		// This case should never happen but we check if the user set a non valid pom
		if p != "<no-project-tag></no-project-tag>" {
			assert.Contains(t, string(newPom), jib.XMLJibProfile)
		}
	}

}

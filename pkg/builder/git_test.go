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
	"os"
	"path"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/jib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitPublicRepo(t *testing.T) {
	tmpGitDir := t.TempDir()

	ctx := &builderContext{
		C:    context.TODO(),
		Path: tmpGitDir,
		Build: v1.BuilderTask{
			Git: &v1.GitConfigSpec{
				URL: "https://github.com/squakez/sample.git",
			},
		},
	}

	err := cloneProject(ctx)
	require.NoError(t, err)
	f, err := os.Stat(path.Join(tmpGitDir, "maven", "pom.xml"))
	require.NoError(t, err)
	assert.Contains(t, f.Name(), "pom.xml")

	// Inject profile test: reused the same test to avoid cloning a project again
	err = injectJibProfile(ctx)
	require.NoError(t, err)
	pomContent, err := util.ReadFile(path.Join(tmpGitDir, "maven", "pom.xml"))
	require.NoError(t, err)
	assert.Contains(t, string(pomContent), jib.XMLJibProfile)

	// Build Mavent Context test: reused the same test to avoid cloning a project again
	// use local Maven executable in tests
	t.Setenv("MAVEN_WRAPPER", boolean.FalseString)
	_, ok := os.LookupEnv("MAVEN_CMD")
	if !ok {
		t.Setenv("MAVEN_CMD", "mvn")
	}
	err = buildMavenContextSettings(ctx)
	require.NoError(t, err)
	err = executeMavenPackageCommand(ctx)
	require.NoError(t, err)
	f, err = os.Stat(path.Join(tmpGitDir, "maven", "target", "test-1.0-SNAPSHOT.jar"))
	require.NoError(t, err)
	assert.Equal(t, "test-1.0-SNAPSHOT.jar", f.Name())
}

func TestGitPrivateRepoFail(t *testing.T) {
	tmpGitDir := t.TempDir()

	ctx := &builderContext{
		Path: tmpGitDir,
		Build: v1.BuilderTask{
			Git: &v1.GitConfigSpec{
				URL: "https://github.com/squakez/private-sample.git",
			},
		},
	}

	err := cloneProject(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication required")
	_, err = os.Stat(path.Join(tmpGitDir, "maven", "pom.xml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestGitCloneBranch(t *testing.T) {
	tmpGitDir := t.TempDir()

	ctx := &builderContext{
		C:    context.TODO(),
		Path: tmpGitDir,
		Build: v1.BuilderTask{
			Git: &v1.GitConfigSpec{
				// the project URL for the https://github.com/squakez/sample.git fork
				URL: "https://github.com/michalvavrik/sample.git",
				// only difference between the main branch and this branch is the 'this_is_expected_branch' empty file
				Branch: "feature/branch-checkout-test",
			},
		},
	}

	err := cloneProject(ctx)
	require.NoError(t, err)
	f, err := os.Stat(path.Join(tmpGitDir, "maven", "pom.xml"))
	require.NoError(t, err)
	assert.Contains(t, f.Name(), "pom.xml")

	f, err = os.Stat(path.Join(tmpGitDir, "maven", "this_is_expected_branch"))
	require.NoError(t, err)
	assert.Contains(t, f.Name(), "this_is_expected_branch")
}

func TestGitCloneTag(t *testing.T) {
	tmpGitDir := t.TempDir()

	ctx := &builderContext{
		C:    context.TODO(),
		Path: tmpGitDir,
		Build: v1.BuilderTask{
			Git: &v1.GitConfigSpec{
				// the project URL for the https://github.com/squakez/sample.git fork
				URL: "https://github.com/michalvavrik/sample.git",
				// only difference between the main branch and this tag is the 'this_is_expected_tag' empty file
				Tag: "v1.2.3",
			},
		},
	}

	err := cloneProject(ctx)
	require.NoError(t, err)
	f, err := os.Stat(path.Join(tmpGitDir, "maven", "pom.xml"))
	require.NoError(t, err)
	assert.Contains(t, f.Name(), "pom.xml")

	f, err = os.Stat(path.Join(tmpGitDir, "maven", "this_is_expected_tag"))
	require.NoError(t, err)
	assert.Contains(t, f.Name(), "this_is_expected_tag")
}

func TestGitCloneCommit(t *testing.T) {
	tmpGitDir := t.TempDir()

	ctx := &builderContext{
		C:    context.TODO(),
		Path: tmpGitDir,
		Build: v1.BuilderTask{
			Git: &v1.GitConfigSpec{
				// the project URL for the https://github.com/squakez/sample.git fork
				URL: "https://github.com/michalvavrik/sample.git",
				// only difference between the main branch and this commit is the 'this_is_expected_commit' empty file
				Commit: "f2b9bd064a62263ab53b3bfe6ac2b71e68dba45b",
			},
		},
	}

	err := cloneProject(ctx)
	require.NoError(t, err)
	f, err := os.Stat(path.Join(tmpGitDir, "maven", "pom.xml"))
	require.NoError(t, err)
	assert.Contains(t, f.Name(), "pom.xml")

	f, err = os.Stat(path.Join(tmpGitDir, "maven", "this_is_expected_commit"))
	require.NoError(t, err)
	assert.Contains(t, f.Name(), "this_is_expected_commit")
}

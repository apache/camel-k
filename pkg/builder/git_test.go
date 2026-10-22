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
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"sync"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/jib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestGitPublicRepoWithPath(t *testing.T) {
	tmpGitDir := t.TempDir()

	ctx := &builderContext{
		C:    context.TODO(),
		Path: tmpGitDir,
		Build: v1.BuilderTask{
			Git: &v1.GitConfigSpec{
				URL:  "https://github.com/squakez/sample.git",
				Path: "src/main/resources/camel",
			},
		},
	}

	err := cloneProject(ctx)
	require.NoError(t, err)
	f, err := os.Stat(path.Join(tmpGitDir, "maven", "test.yaml"))
	require.NoError(t, err)
	assert.Contains(t, f.Name(), "test.yaml")
}

// gitBasicAuthRecorder is a fake Git HTTP server which records the basic auth credentials it receives
// and rejects every request.
type gitBasicAuthRecorder struct {
	mu       sync.Mutex
	called   bool
	ok       bool
	username string
	password string
}

func (r *gitBasicAuthRecorder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.called = true
	r.username, r.password, r.ok = req.BasicAuth()
	w.WriteHeader(http.StatusUnauthorized)
}

func TestGitCloneUsername(t *testing.T) {
	tests := []struct {
		name             string
		username         string
		expectedUsername string
	}{
		{
			name:             "default username",
			expectedUsername: "camel-k",
		},
		{
			name:             "custom username",
			username:         "x-token-auth",
			expectedUsername: "x-token-auth",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &gitBasicAuthRecorder{}
			server := httptest.NewServer(recorder)
			defer server.Close()

			c, err := internal.NewFakeClient(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Name:      "git-secret",
				},
				Data: map[string][]byte{"token": []byte("my-token")},
			})
			require.NoError(t, err)

			ctx := &builderContext{
				C:         context.TODO(),
				Client:    c,
				Namespace: "ns",
				Path:      t.TempDir(),
				Build: v1.BuilderTask{
					Git: &v1.GitConfigSpec{
						URL:      server.URL + "/repo.git",
						Secret:   "git-secret",
						Username: tc.username,
					},
				},
			}

			// the fake Git server rejects any request, we only care about the credentials received
			require.Error(t, cloneProject(ctx))
			recorder.mu.Lock()
			defer recorder.mu.Unlock()
			assert.True(t, recorder.called)
			assert.True(t, recorder.ok)
			assert.Equal(t, tc.expectedUsername, recorder.username)
			assert.Equal(t, "my-token", recorder.password)
		})
	}
}

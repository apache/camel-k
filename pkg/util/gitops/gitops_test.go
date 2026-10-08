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

package util

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// basicAuthRecorder is a fake Git HTTP server which records the basic auth credentials it receives
// and rejects every request.
type basicAuthRecorder struct {
	mu       sync.Mutex
	called   bool
	ok       bool
	username string
	password string
}

func (r *basicAuthRecorder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.called = true
	r.username, r.password, r.ok = req.BasicAuth()
	w.WriteHeader(http.StatusUnauthorized)
}

func cloneWithRecorder(t *testing.T, clone func(conf v1.GitConfigSpec, dir string) error) *basicAuthRecorder {
	t.Helper()
	recorder := &basicAuthRecorder{}
	server := httptest.NewServer(recorder)
	defer server.Close()

	err := clone(v1.GitConfigSpec{URL: server.URL + "/repo.git"}, t.TempDir())
	require.Error(t, err)
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	assert.True(t, recorder.called)

	return recorder
}

func TestCloneGitProjectDefaultUsername(t *testing.T) {
	recorder := cloneWithRecorder(t, func(conf v1.GitConfigSpec, dir string) error {
		_, err := CloneGitProject(conf, dir, DefaultGitUsername, "my-token")

		return err
	})
	assert.True(t, recorder.ok)
	assert.Equal(t, DefaultGitUsername, recorder.username)
	assert.Equal(t, "my-token", recorder.password)
}

func TestCloneGitProject(t *testing.T) {
	recorder := cloneWithRecorder(t, func(conf v1.GitConfigSpec, dir string) error {
		_, err := CloneGitProject(conf, dir, "x-token-auth", "my-token")

		return err
	})
	assert.True(t, recorder.ok)
	assert.Equal(t, "x-token-auth", recorder.username)
	assert.Equal(t, "my-token", recorder.password)
}

func TestCloneGitProjectWithoutToken(t *testing.T) {
	recorder := cloneWithRecorder(t, func(conf v1.GitConfigSpec, dir string) error {
		_, err := CloneGitProject(conf, dir, "x-token-auth", "")

		return err
	})
	assert.False(t, recorder.ok)
}

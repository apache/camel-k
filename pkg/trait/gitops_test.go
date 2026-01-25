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

package trait

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	git "github.com/go-git/go-git/v5"
)

func TestGitOpsAddAction(t *testing.T) {
	trait, _ := newGitOpsTrait().(*gitOpsTrait)
	trait.Enabled = ptr.To(true)
	env := &Environment{
		Integration: &v1.Integration{
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		PostActions: []func(*Environment) error{},
	}
	ok, _, err := trait.Configure(env)
	require.NoError(t, err)
	assert.True(t, ok)
	err = trait.Apply(env)
	require.NoError(t, err)
	assert.Len(t, env.PostActions, 1)
}

func TestGitOpsPushRepoDefault(t *testing.T) {
	trait, _ := newGitOpsTrait().(*gitOpsTrait)
	trait.Overlays = []string{"dev", "prod"}
	trait.IntegrationDirectory = "integrations"
	// As this test would require to access to a private repository,
	// We are simulating a remote repository pointing to a local fake repository.
	srcGitDir := t.TempDir()
	tmpGitDir := t.TempDir()
	err := initFakeGitRepo(srcGitDir)
	require.NoError(t, err)
	it := v1.NewIntegration("default", "test")
	conf := &v1.GitConfigSpec{
		URL: srcGitDir,
	}
	it.Spec = v1.IntegrationSpec{
		Git:     conf,
		Sources: []v1.SourceSpec{v1.NewSourceSpec("Test.java", "bogus, irrelevant for test", v1.LanguageJavaSource)},
	}
	now := metav1.Now().Rfc3339Copy()
	it.Status = v1.IntegrationStatus{
		Image:               "my-img-recently-baked",
		DeploymentTimestamp: &now,
	}

	err = trait.pushGitOpsItInGitRepo(context.TODO(), &it, tmpGitDir, "fake")
	require.NoError(t, err)
	assert.Contains(t,
		it.Status.GetCondition(v1.IntegrationConditionType("GitPushed")).Message,
		"Integration changes pushed to branch cicd/candidate-release",
	)

	lastCommitMessage, err := getLastCommitMessage(tmpGitDir)
	require.NoError(t, err)
	assert.Contains(t, lastCommitMessage, "feat(ci): build complete")
	branchName, err := getBranchName(tmpGitDir)
	require.NoError(t, err)
	assert.Contains(t, branchName, "cicd/candidate-release")
	remoteUrl, err := getRemoteURL(tmpGitDir)
	require.NoError(t, err)
	assert.Equal(t, srcGitDir, remoteUrl)
	gitopsDir, err := os.Stat(filepath.Join(tmpGitDir, "integrations", it.Name))
	require.NoError(t, err)
	assert.True(t, gitopsDir.IsDir())
	gitopsDir, err = os.Stat(filepath.Join(tmpGitDir, "integrations", it.Name, "overlays", "dev"))
	require.NoError(t, err)
	assert.True(t, gitopsDir.IsDir())
	gitopsDir, err = os.Stat(filepath.Join(tmpGitDir, "integrations", it.Name, "overlays", "prod"))
	require.NoError(t, err)
	assert.True(t, gitopsDir.IsDir())
}

// initFakeGitInmemoryRepo has the goal to create a fake a git repository into a given directory.
// We can use this to simulate pull and push activities.
func initFakeGitRepo(dirPath string) error {
	repo, err := git.PlainInit(dirPath, false)
	if err != nil {
		return err
	}
	filePath := filepath.Join(dirPath, "README")
	if err := os.WriteFile(filePath, []byte("Hello test!"), 0644); err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	_, err = wt.Add("README")
	if err != nil {
		return err
	}
	_, err = wt.Commit("init commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Tester",
			Email: "tester@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// getLastCommitMessage returns the latest commit message of the Git repository in dirPath.
func getLastCommitMessage(dirPath string) (string, error) {
	repo, err := git.PlainOpen(dirPath)
	if err != nil {
		return "", err
	}
	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}
	commit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", err
	}

	return commit.Message, nil
}

// getBranchName returns the branch name of a given directory.
func getBranchName(dirPath string) (string, error) {
	repo, err := git.PlainOpen(dirPath)
	if err != nil {
		return "", err
	}
	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}
	return headRef.Name().Short(), nil
}

func getRemoteURL(dirPath string) (string, error) {
	repo, err := git.PlainOpen(dirPath)
	if err != nil {
		return "", err
	}
	remote, err := repo.Remote("origin")
	if err != nil {
		return "", err
	}
	urls := remote.Config().URLs
	if len(urls) == 0 {
		return "", errors.New("no URLs found for remote 'origin'")
	}

	return urls[0], nil
}

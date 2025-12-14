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
	"time"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	util "github.com/apache/camel-k/v2/pkg/util/gitops"
	"github.com/apache/camel-k/v2/pkg/util/io"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"k8s.io/utils/ptr"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

const (
	gitOpsTraitID    = "gitops"
	gitOpsTraitOrder = 1700
)

type gitOpsTrait struct {
	BaseTrait
	traitv1.GitOpsTrait `property:",squash"`
}

func newGitOpsTrait() Trait {
	return &gitOpsTrait{
		BaseTrait: NewBaseTrait(gitOpsTraitID, gitOpsTraitOrder),
	}
}

func (t *gitOpsTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil || !ptr.Deref(t.Enabled, false) {
		return false, nil, nil
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseDeploying), nil, nil
}

func (t *gitOpsTrait) Apply(e *Environment) error {
	// Register a post action that is in charge to create a PR on the Git project.
	// It must be done on Deploying phase in order to catch the Integration status changed
	// after all traits executed in that phase.
	e.PostActions = append(e.PostActions, func(env *Environment) error {
		gitToken, err := util.GitToken(env.Ctx, env.Client, env.Integration.Namespace, t.Secret)
		if err != nil {
			return err
		}
		if gitToken == "" {
			return errors.New("no git token provided")
		}

		return t.pushGitOpsRepo(env.Ctx, env.Integration, gitToken)
	})

	return nil
}

// withTempDir wraps the execution of a function making sure to create a temporary directory and cleaning it when finishing
// the function.
func withTempDir(fn func(dir string) error) error {
	dir, err := os.MkdirTemp("tmp", "integration-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	return fn(dir)
}

// pushGitOpsRepo makes sure to use a temporary directory.
func (t *gitOpsTrait) pushGitOpsRepo(ctx context.Context, it *v1.Integration, token string) error {
	return withTempDir(func(dir string) error {
		return t.pushGitOpsItInGitRepo(ctx, it, dir, token)
	})
}

// pushGitOpsItInGitRepo is in charge to clone the repo, do the kustomize overlays and push the changes
// to a new branch.
func (t *gitOpsTrait) pushGitOpsItInGitRepo(ctx context.Context, it *v1.Integration, dir, token string) error {
	gitConf := t.gitConf(it)
	// Clone repo
	repo, err := util.CloneGitProject(gitConf, dir, token)
	if err != nil {
		return err
	}
	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	// Create a new branch
	nowDate := time.Now().Format("20060102-150405")
	branchName := t.BranchPush
	if branchName == "" {
		branchName = "cicd/candidate-release-" + nowDate
	}
	commitMessage := "feat(ci): build completed on " + nowDate
	branchRef := plumbing.NewBranchReferenceName(branchName)

	err = w.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
		Create: true,
	})
	if err != nil {
		return err
	}

	// Generate Kustomize content (it may override upstream content)
	ciCdDir := filepath.Join(dir, t.IntegrationDirectory)
	err = os.MkdirAll(ciCdDir, io.FilePerm755)
	if err != nil {
		return err
	}

	kit, err := getIntegrationKit(ctx, t.Client, it)
	if err != nil {
		return err
	}

	for _, overlay := range t.Overlays {
		destIntegration := util.EditIntegration(it, kit, overlay, "")
		err = util.AppendKustomizeIntegration(destIntegration, ciCdDir, t.OverwriteOverlay)
		if err != nil {
			return err
		}
	}

	// Commit and push new content
	_, err = w.Add(t.IntegrationDirectory)
	if err != nil {
		return err
	}
	_, err = w.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  t.CommiterName,
			Email: t.CommiterEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	gitPushOptions := &git.PushOptions{
		RemoteURL: gitConf.URL,
		Auth: &http.BasicAuth{
			Username: "camel-k",
			Password: token,
		},
		RefSpecs: []config.RefSpec{
			config.RefSpec(branchRef + ":" + branchRef),
		},
	}

	return repo.Push(gitPushOptions)
}

// gitConf returns the git repo configuration where to pull the project from. If no value is provided, then, it takes
// the value coming from Integration git project (if specified).
func (t *gitOpsTrait) gitConf(it *v1.Integration) v1.GitConfigSpec {
	gitConf := v1.GitConfigSpec{
		URL:    t.URL,
		Branch: t.Branch,
		Tag:    t.Tag,
		Commit: t.Commit,
	}
	if it.Spec.Git != nil {
		if gitConf.URL == "" {
			gitConf.URL = it.Spec.Git.URL
		}
		if gitConf.Branch == "" {
			gitConf.Branch = it.Spec.Git.Branch
		}
		if gitConf.Tag == "" {
			gitConf.Tag = it.Spec.Git.Tag
		}
		if gitConf.Commit == "" {
			gitConf.Commit = it.Spec.Git.Commit
		}
	}

	return gitConf
}

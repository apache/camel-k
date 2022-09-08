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

package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/test"
)

const cmdKameletAddRepo = "add-repo"

// nolint: unparam
func initializeKameletAddRepoCmdOptions(t *testing.T) (*kameletAddRepoCommandOptions, *cobra.Command, RootCmdOptions) {
	t.Helper()

	options, rootCmd := kamelTestPreAddCommandInit()
	kameletAddRepoCommandOptions := addTestKameletAddRepoCmd(*options, rootCmd)
	kamelTestPostAddCommandInit(t, rootCmd)

	return kameletAddRepoCommandOptions, rootCmd, *options
}

func addTestKameletAddRepoCmd(options RootCmdOptions, rootCmd *cobra.Command) *kameletAddRepoCommandOptions {
	// Add a testing version of kamelet add-repo Command
	kameletAddRepoCmd, kameletAddRepoOptions := newKameletAddRepoCmd(&options)
	kameletAddRepoCmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	kameletAddRepoCmd.PostRunE = func(c *cobra.Command, args []string) error {
		return nil
	}
	kameletAddRepoCmd.Args = test.ArbitraryArgs
	rootCmd.AddCommand(kameletAddRepoCmd)
	return kameletAddRepoOptions
}

func TestKameletAddRepoNoFlag(t *testing.T) {
	_, rootCmd, _ := initializeKameletAddRepoCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdKameletAddRepo, "foo")
	assert.Nil(t, err)
}

func TestKameletAddRepoNonExistingFlag(t *testing.T) {
	_, rootCmd, _ := initializeKameletAddRepoCmdOptions(t)
	_, err := test.ExecuteCommand(rootCmd, cmdKameletAddRepo, "--nonExistingFlag", "foo")
	assert.NotNil(t, err)
}

func TestKameletAddRepoInvalidRepositoryURI(t *testing.T) {
	repositories := []v1.IntegrationPlatformKameletRepositorySpec{}
	assert.NotNil(t, checkURI("foo", repositories))
	assert.NotNil(t, checkURI("github", repositories))
	assert.NotNil(t, checkURI("github:", repositories))
	assert.NotNil(t, checkURI("github:foo", repositories))
	assert.NotNil(t, checkURI("github:foo/", repositories))
}

func TestKameletAddRepoValidRepositoryURI(t *testing.T) {
	repositories := []v1.IntegrationPlatformKameletRepositorySpec{}
	assert.Nil(t, checkURI("github:foo/bar", repositories))
	assert.Nil(t, checkURI("github:foo/bar/some/path", repositories))
	assert.Nil(t, checkURI("github:foo/bar@1.0", repositories))
	assert.Nil(t, checkURI("github:foo/bar/some/path@1.0", repositories))
}

func TestKameletAddRepoDuplicateRepositoryURI(t *testing.T) {
	repositories := []v1.IntegrationPlatformKameletRepositorySpec{{URI: "github:foo/bar"}}
	assert.NotNil(t, checkURI("github:foo/bar", repositories))
	assert.Nil(t, checkURI("github:foo/bar2", repositories))
}

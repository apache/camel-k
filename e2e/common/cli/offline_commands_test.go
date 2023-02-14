//go:build integration
// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/apache/camel-k/e2e/support"
	. "github.com/onsi/gomega"
)

func TestKamelVersionWorksOffline(t *testing.T) {
	RegisterTestingT(t)

	assert.Nil(t, Kamel("version", "--kube-config", "non-existent-kubeconfig-file").Execute())
}

func TestKamelHelpTraitWorksOffline(t *testing.T) {
	RegisterTestingT(t)

	traitCmd := Kamel("help", "trait", "--all", "--kube-config", "non-existent-kubeconfig-file")
	traitCmd.SetOut(io.Discard)
	assert.Nil(t, traitCmd.Execute())
}

func TestKamelHelpOptionWorksOffline(t *testing.T) {
	RegisterTestingT(t)

	traitCmd := Kamel("run", "Xxx.java", "--help")
	traitCmd.SetOut(io.Discard)
	assert.Nil(t, traitCmd.Execute())
}

func TestKamelCompletionWorksOffline(t *testing.T) {
	RegisterTestingT(t)

	bashCmd := Kamel("completion", "bash", "--kube-config", "non-existent-kubeconfig-file")
	bashCmd.SetOut(io.Discard)
	zshCmd := Kamel("completion", "zsh", "--kube-config", "non-existent-kubeconfig-file")
	zshCmd.SetOut(io.Discard)
	assert.Nil(t, bashCmd.Execute())
	assert.Nil(t, zshCmd.Execute())
}

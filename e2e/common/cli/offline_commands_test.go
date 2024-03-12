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

	. "github.com/apache/camel-k/v2/e2e/support"
	. "github.com/onsi/gomega"
)

func TestKamelVersionWorksOffline(t *testing.T) {
	g := NewWithT(t)
	g.Expect(Kamel(t, "version", "--kube-config", "non-existent-kubeconfig-file").Execute()).To(Succeed())
}

func TestKamelHelpOptionWorksOffline(t *testing.T) {
	g := NewWithT(t)

	traitCmd := Kamel(t, "run", "Xxx.java", "--help")
	traitCmd.SetOut(io.Discard)
	g.Expect(traitCmd.Execute()).To(Succeed())
}

func TestKamelCompletionWorksOffline(t *testing.T) {
	g := NewWithT(t)

	bashCmd := Kamel(t, "completion", "bash", "--kube-config", "non-existent-kubeconfig-file")
	bashCmd.SetOut(io.Discard)
	zshCmd := Kamel(t, "completion", "zsh", "--kube-config", "non-existent-kubeconfig-file")
	zshCmd.SetOut(io.Discard)
	g.Expect(bashCmd.Execute()).To(Succeed())
	g.Expect(zshCmd.Execute()).To(Succeed())
}

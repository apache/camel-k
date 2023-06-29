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
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	. "github.com/apache/camel-k/v2/e2e/support"
	"github.com/apache/camel-k/v2/e2e/support/util"
)

func TestPruneIntegrationKits(t *testing.T) {
	RegisterTestingT(t)

	t.Run("Prune IntegrationKits", func(t *testing.T) {
		// make sure kits are deleted
		Expect(DeleteKits(ns)).To(Succeed())
		tests := []struct {
			title    string
			kits     string
			toDelete string
		}{
			{
				title:    "Nothing to do",
				kits:     "",
				toDelete: "",
			},
			{
				title:    "Single kit that is used",
				kits:     "a(t)",
				toDelete: "",
			},
			{
				title:    "Single kit that is unused",
				kits:     "a(f)",
				toDelete: "a",
			},
			{
				title:    "Basic tree",
				kits:     "a(f)b(t)",
				toDelete: "a",
			},
			{
				title:    "Simple tree line",
				kits:     "a(f)b(f)c(f)d(f)e(t)",
				toDelete: "abcd",
			},
			// Simple Tree
			{
				// Syntax for n ary tree in preorder traversal is NAME(f|t)| where:
				// Name is the name of the kit,
				// t : kit is used
				// f : kit is unused
				// | marks the end of children
				title:    "Simple tree",
				kits:     "a(f)b(f)e(f)|f(f)k(t)|||c(t)|d(f)g(t)|h(f)|i(f)|j(t)|||",
				toDelete: "abefdhi",
			},
		}

		for _, curr := range tests {
			thetest := curr
			t.Run(thetest.title, func(t *testing.T) {
				// build kit tree
				nodes := buildKits(thetest.kits, operatorID, ns, t)

				// check dry run
				checkPruneDryRunLogs(ns, thetest.toDelete, nodes, t)

				// check real run
				Expect(Kamel("kit", "prune", "-y", "-n", ns).Execute()).To(Succeed())

				for _, r := range thetest.toDelete {
					kitName := string(r)
					Eventually(Kit(ns, kitName), TestTimeoutShort).Should(BeNil())
				}
				for _, node := range nodes {
					// It we shouldn't delete it then it should still exist
					if !strings.Contains(thetest.toDelete, node.Name) {
						assert.NotNil(t, Kit(ns, node.Kit.Name)())
					}
					if node.Used {
						// check that the integration still runs fine
						for _, message := range node.ExpectedMessages {
							Eventually(IntegrationLogs(ns, node.Name), TestTimeoutShort).Should(ContainSubstring(message))
						}
					}
				}
				// Clean up
				Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
				Expect(DeleteKits(ns)).To(Succeed())
			})
		}
	})
}

func checkPruneDryRunLogs(ns string, toDelete string, nodes map[string]*TestNode, t *testing.T) {
	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()
	kamelBuild := KamelWithContext(TestContext, "kit", "prune", "-d", "-n", ns)
	kamelBuild.SetOut(pipew)
	kamelBuild.SetErr(pipew)

	deleteness := make([]string, 0)
	if len(toDelete) > 0 {
		deleteness = append(deleteness, "The following Integration Kits will be deleted:")
		deleteness = append(deleteness, "The following Images will no longer be used by camel-k and can be deleted from the Image Registry:")
		for _, r := range toDelete {
			node := nodes[string(r)]
			deleteness = append(deleteness, fmt.Sprintf("%s in namespace: %s", node.Kit.Name, ns))
			deleteness = append(deleteness, node.Kit.Status.Image)
		}
	}
	if len(toDelete) == 0 {
		deleteness = append(deleteness, "Nothing to do")
	}
	logScanner := util.NewStrictLogScanner(ctx, piper, true, deleteness...)
	go func() {
		err := kamelBuild.Execute()
		assert.NoError(t, err)
		logScanner.Done()
		cancel()
	}()
	Eventually(logScanner.ExactMatch(), TestTimeoutShort).Should(BeTrue())
}

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

package kustomize

import (
	"fmt"
	"os"
	"testing"

	. "github.com/apache/camel-k/v2/e2e/support"
	testutil "github.com/apache/camel-k/v2/e2e/support/util"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	. "github.com/onsi/gomega"
)

func TestSetupKustomizeBasic(t *testing.T) {
	g := NewWithT(t)
	makeDir := testutil.MakeTempCopyDir(t, "../../../install")
	os.Setenv("CAMEL_K_TEST_MAKE_DIR", makeDir)

	// Ensure no CRDs are already installed
	g.Expect(UninstallAll(t)).To(Succeed())
	g.Eventually(CRDs(t)).Should(HaveLen(0))

	// Return the cluster to previous state
	defer Cleanup(t)

	WithNewTestNamespace(t, func(g *WithT, ns string) {
		namespaceArg := fmt.Sprintf("NAMESPACE=%s", ns)
		ExpectExecSucceed(t, g, Make(t, "setup-cluster", namespaceArg))
		g.Eventually(CRDs(t)).Should(HaveLen(GetExpectedCRDs(defaults.Version)))

		ExpectExecSucceed(t, g, Make(t, "setup", namespaceArg))

		kpRoles := ExpectedKubePromoteRoles
		opRoles := kpRoles + ExpectedOSPromoteRoles
		g.Eventually(Role(t, ns)).Should(Or(HaveLen(kpRoles), HaveLen(opRoles)))

		kcRoles := ExpectedKubeClusterRoles
		ocRoles := kcRoles + ExpectedOSClusterRoles
		g.Eventually(ClusterRole(t)).Should(Or(HaveLen(kcRoles), HaveLen(ocRoles)))

		// Tidy up to ensure next test works
		g.Expect(Kamel(t, "uninstall", "-n", ns).Execute()).To(Succeed())
	})

}

func TestSetupKustomizeGlobal(t *testing.T) {
	makeDir := testutil.MakeTempCopyDir(t, "../../../install")
	os.Setenv("CAMEL_K_TEST_MAKE_DIR", makeDir)

	// Ensure no CRDs are already installed
	g := NewWithT(t)
	g.Expect(UninstallAll(t)).To(Succeed())
	g.Eventually(CRDs(t)).Should(HaveLen(0))

	// Return the cluster to previous state
	defer Cleanup(t)

	WithNewTestNamespace(t, func(g *WithT, ns string) {
		namespaceArg := fmt.Sprintf("NAMESPACE=%s", ns)
		ExpectExecSucceed(t, g, Make(t, "setup-cluster", namespaceArg))
		g.Eventually(CRDs(t)).Should(HaveLen(GetExpectedCRDs(defaults.Version)))

		ExpectExecSucceed(t, g, Make(t, "setup", "GLOBAL=true", namespaceArg))

		g.Eventually(Role(t, ns)).Should(HaveLen(0))

		kcpRoles := ExpectedKubeClusterRoles + ExpectedKubePromoteRoles
		ocpRoles := kcpRoles + ExpectedOSClusterRoles + ExpectedOSPromoteRoles
		g.Eventually(ClusterRole(t)).Should(Or(HaveLen(kcpRoles), HaveLen(ocpRoles)))
	})
}

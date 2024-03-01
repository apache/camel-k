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

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	testutil "github.com/apache/camel-k/v2/e2e/support/util"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	. "github.com/onsi/gomega"
)

func TestKustomizeUninstallBasic(t *testing.T) {
	RegisterTestingT(t)
	makeDir := testutil.MakeTempCopyDir(t, "../../../install")
	os.Setenv("CAMEL_K_TEST_MAKE_DIR", makeDir)

	// Ensure no CRDs are already installed
	Expect(UninstallAll(t)).To(Succeed())
	Eventually(CRDs(t)).Should(HaveLen(0))

	// Return the cluster to previous state
	defer Cleanup(t)

	WithNewTestNamespace(t, func(ns string) {
		namespaceArg := fmt.Sprintf("NAMESPACE=%s", ns)
		ExpectExecSucceed(t, Make(t, "setup-cluster", namespaceArg))
		ExpectExecSucceed(t, Make(t, "setup", namespaceArg))
		ExpectExecSucceed(t, Make(t, "platform", namespaceArg))
		// Skip default kamelets installation for faster test runs
		ExpectExecSucceed(t, Make(t, "operator", namespaceArg, "INSTALL_DEFAULT_KAMELETS=false"))
		Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		Eventually(OperatorPodPhase(t, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

		// Do uninstall
		ExpectExecSucceed(t, Make(t, "uninstall", namespaceArg))

		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)

		Eventually(OperatorPod(t, ns)).Should(BeNil())
		Eventually(Platform(t, ns)).Should(BeNil())
		// The operator can dynamically create a for its builders
		// so, in case there is a build strategy "pod", expect this to have 1 role
		Eventually(Role(t, ns)).Should(BeNil())
		Eventually(ClusterRole(t)).Should(BeNil())
		// CRDs should be still there
		Eventually(CRDs(t)).Should(HaveLen(GetExpectedCRDs(defaults.Version)))

		// Do uninstall all
		ExpectExecSucceed(t, Make(t, "uninstall", namespaceArg, "UNINSTALL_ALL=true"))

		Eventually(CRDs(t)).Should(BeNil())
	})

}

func TestUninstallGlobal(t *testing.T) {
	RegisterTestingT(t)
	makeDir := testutil.MakeTempCopyDir(t, "../../../install")
	os.Setenv("CAMEL_K_TEST_MAKE_DIR", makeDir)

	// Ensure no CRDs are already installed
	Expect(UninstallAll(t)).To(Succeed())
	Eventually(CRDs(t)).Should(HaveLen(0))

	// Return the cluster to previous state
	defer Cleanup(t)

	WithNewTestNamespace(t, func(ns string) {
		namespaceArg := fmt.Sprintf("NAMESPACE=%s", ns)
		ExpectExecSucceed(t, Make(t, "setup-cluster", namespaceArg))
		ExpectExecSucceed(t, Make(t, "setup", namespaceArg, "GLOBAL=true"))
		ExpectExecSucceed(t, Make(t, "platform", namespaceArg))
		// Skip default kamelets installation for faster test runs
		ExpectExecSucceed(t, Make(t, "operator", namespaceArg, "GLOBAL=true", "INSTALL_DEFAULT_KAMELETS=false"))
		Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		Eventually(OperatorPodPhase(t, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

		// Do uninstall
		ExpectExecSucceed(t, Make(t, "uninstall", namespaceArg))

		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)

		Eventually(OperatorPod(t, ns)).Should(BeNil())
		Eventually(Platform(t, ns)).Should(BeNil())
		Eventually(Role(t, ns)).Should(BeNil())
		Eventually(ClusterRole(t)).Should(BeNil())
		// CRDs should be still there
		Eventually(CRDs(t)).Should(HaveLen(GetExpectedCRDs(defaults.Version)))

		// Do uninstall all
		ExpectExecSucceed(t, Make(t, "uninstall", namespaceArg, "UNINSTALL_ALL=true"))

		Eventually(CRDs(t)).Should(BeNil())
	})
}

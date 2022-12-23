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

	. "github.com/apache/camel-k/e2e/support"
	testutil "github.com/apache/camel-k/e2e/support/util"
	. "github.com/onsi/gomega"
)

func TestUninstallBasic(t *testing.T) {
	makeDir := testutil.MakeTempCopyDir(t, "../../../../install")
	os.Setenv("CAMEL_K_TEST_MAKE_DIR", makeDir)

	// Ensure no CRDs are already installed
	UninstallAll()

	// Return the cluster to previous state
	defer Cleanup()

	WithNewTestNamespace(t, func(ns string) {
		namespaceArg := fmt.Sprintf("NAMESPACE=%s", ns)
		ExpectExecSucceed(t, Make("setup-cluster", namespaceArg))
		ExpectExecSucceed(t, Make("setup", namespaceArg))
		ExpectExecSucceed(t, Make("platform", namespaceArg))
		// Skip default kamelets installation for faster test runs
		ExpectExecSucceed(t, Make("operator", namespaceArg, "INSTALL_DEFAULT_KAMELETS=false"))
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		Eventually(OperatorPodPhase(ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

		// Do uninstall
		ExpectExecSucceed(t, Make("uninstall", namespaceArg))

		// Refresh the test client to account for the newly installed CRDs
		SyncClient()

		Eventually(OperatorPod(ns)).Should(BeNil())
		Eventually(Platform(ns)).Should(BeNil())
		Eventually(Role(ns)).Should(BeNil())
		Eventually(ClusterRole()).Should(BeNil())
		// CRDs should be still there
		Eventually(CRDs()).Should(HaveLen(ExpectedCRDs))

		// Do uninstall all
		ExpectExecSucceed(t, Make("uninstall", namespaceArg, "UNINSTALL_ALL=true"))

		Eventually(CRDs()).Should(BeNil())
	})

}

func TestUninstallGlobal(t *testing.T) {
	makeDir := testutil.MakeTempCopyDir(t, "../../../../install")
	os.Setenv("CAMEL_K_TEST_MAKE_DIR", makeDir)

	// Ensure no CRDs are already installed
	UninstallAll()

	// Return the cluster to previous state
	defer Cleanup()

	WithNewTestNamespace(t, func(ns string) {
		namespaceArg := fmt.Sprintf("NAMESPACE=%s", ns)
		ExpectExecSucceed(t, Make("setup-cluster", namespaceArg))
		ExpectExecSucceed(t, Make("setup", namespaceArg, "GLOBAL=true"))
		ExpectExecSucceed(t, Make("platform", namespaceArg))
		// Skip default kamelets installation for faster test runs
		ExpectExecSucceed(t, Make("operator", namespaceArg, "GLOBAL=true", "INSTALL_DEFAULT_KAMELETS=false"))
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		Eventually(OperatorPodPhase(ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

		// Do uninstall
		ExpectExecSucceed(t, Make("uninstall", namespaceArg))

		// Refresh the test client to account for the newly installed CRDs
		SyncClient()

		Eventually(OperatorPod(ns)).Should(BeNil())
		Eventually(Platform(ns)).Should(BeNil())
		Eventually(Role(ns)).Should(BeNil())
		Eventually(ClusterRole()).Should(BeNil())
		// CRDs should be still there
		Eventually(CRDs()).Should(HaveLen(ExpectedCRDs))

		// Do uninstall all
		ExpectExecSucceed(t, Make("uninstall", namespaceArg, "UNINSTALL_ALL=true"))

		Eventually(CRDs()).Should(BeNil())
	})
}

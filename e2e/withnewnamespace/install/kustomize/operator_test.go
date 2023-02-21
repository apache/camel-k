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

func TestOperatorBasic(t *testing.T) {
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
		// Skip default kamelets installation for faster test runs
		ExpectExecSucceed(t, Make("operator",
			namespaceArg,
			"INSTALL_DEFAULT_KAMELETS=false"))

		// Refresh the test client to account for the newly installed CRDs
		SyncClient()

		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		Eventually(OperatorPodPhase(ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		Eventually(Platform(ns)).ShouldNot(BeNil())
	})
}

func TestOperatorAlternativeImage(t *testing.T) {
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

		// Skip default kamelets installation for faster test runs
		newImage := "quay.io/kameltest/kamel-operator"
		newTag := "1.1.1"
		ExpectExecSucceed(t, Make("operator",
			fmt.Sprintf("CUSTOM_IMAGE=%s", newImage),
			fmt.Sprintf("CUSTOM_VERSION=%s", newTag),
			namespaceArg,
			"INSTALL_DEFAULT_KAMELETS=false"))

		// Refresh the test client to account for the newly installed CRDs
		SyncClient()

		Eventually(OperatorImage(ns)).Should(Equal(fmt.Sprintf("%s:%s", newImage, newTag)))
	})
}

func TestOperatorGlobal(t *testing.T) {
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

		// Skip default kamelets installation for faster test runs
		ExpectExecSucceed(t, Make("operator",
			namespaceArg,
			"GLOBAL=true",
			"INSTALL_DEFAULT_KAMELETS=false"))

		// Refresh the test client to account for the newly installed CRDs
		SyncClient()

		podFunc := OperatorPod(ns)
		Eventually(podFunc).ShouldNot(BeNil())
		Eventually(OperatorPodPhase(ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		pod := podFunc()

		containers := pod.Spec.Containers
		Expect(containers).NotTo(BeEmpty())

		envvars := containers[0].Env
		Expect(envvars).NotTo(BeEmpty())

		found := false
		for _, v := range envvars {
			if v.Name == "WATCH_NAMESPACE" {
				Expect(v.Value).To(Equal("\"\""))
				found = true
				break
			}
		}
		Expect(found).To(BeTrue())

		Eventually(Platform(ns)).ShouldNot(BeNil())
	})
}

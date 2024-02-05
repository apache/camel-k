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
	"context"
	"fmt"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	testutil "github.com/apache/camel-k/v2/e2e/support/util"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"

	. "github.com/onsi/gomega"
)

func TestOperatorBasic(t *testing.T) {
	makeDir := testutil.MakeTempCopyDir(t, "../../../install")
	os.Setenv("CAMEL_K_TEST_MAKE_DIR", makeDir)

	ctx := TestContext()

	// Ensure no CRDs are already installed
	g := NewWithT(t)
	g.Expect(UninstallAll(t, ctx)).To(Succeed())

	// Return the cluster to previous state
	defer Cleanup(t, ctx)

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		namespaceArg := fmt.Sprintf("NAMESPACE=%s", ns)
		ExpectExecSucceed(t, g, Make(t, "setup-cluster", namespaceArg))
		ExpectExecSucceed(t, g, Make(t, "setup", namespaceArg))
		// Skip default kamelets installation for faster test runs
		ExpectExecSucceed(t, g, Make(t, "operator",
			namespaceArg,
			"INSTALL_DEFAULT_KAMELETS=false"))

		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)

		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		g.Eventually(OperatorPodPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

		// Check if restricted security context has been applyed
		operatorPod := OperatorPod(t, ctx, ns)()
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.RunAsNonRoot).To(Equal(kubernetes.DefaultOperatorSecurityContext().RunAsNonRoot))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.Capabilities).To(Equal(kubernetes.DefaultOperatorSecurityContext().Capabilities))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.SeccompProfile).To(Equal(kubernetes.DefaultOperatorSecurityContext().SeccompProfile))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation).To(Equal(kubernetes.DefaultOperatorSecurityContext().AllowPrivilegeEscalation))

		g.Eventually(Platform(t, ctx, ns)).ShouldNot(BeNil())
		registry := os.Getenv("KIND_REGISTRY")
		if registry != "" {
			platform := Platform(t, ctx, ns)()
			g.Expect(platform.Status.Build.Registry).ShouldNot(BeNil())
			g.Expect(platform.Status.Build.Registry.Address).To(Equal(registry))
		}

	})
}

func TestOperatorKustomizeAlternativeImage(t *testing.T) {
	makeDir := testutil.MakeTempCopyDir(t, "../../../install")
	os.Setenv("CAMEL_K_TEST_MAKE_DIR", makeDir)

	ctx := TestContext()

	// Ensure no CRDs are already installed
	g := NewWithT(t)
	g.Expect(UninstallAll(t, ctx)).To(Succeed())

	// Return the cluster to previous state
	defer Cleanup(t, ctx)

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		namespaceArg := fmt.Sprintf("NAMESPACE=%s", ns)
		ExpectExecSucceed(t, g, Make(t, "setup-cluster", namespaceArg))
		ExpectExecSucceed(t, g, Make(t, "setup", namespaceArg))

		// Skip default kamelets installation for faster test runs
		newImage := "quay.io/kameltest/kamel-operator"
		newTag := "1.1.1"
		ExpectExecSucceed(t, g, Make(t, "operator",
			fmt.Sprintf("CUSTOM_IMAGE=%s", newImage),
			fmt.Sprintf("CUSTOM_VERSION=%s", newTag),
			namespaceArg,
			"INSTALL_DEFAULT_KAMELETS=false"))

		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)

		g.Eventually(OperatorImage(t, ctx, ns)).Should(Equal(fmt.Sprintf("%s:%s", newImage, newTag)))
	})
}

func TestOperatorKustomizeGlobal(t *testing.T) {
	makeDir := testutil.MakeTempCopyDir(t, "../../../install")
	os.Setenv("CAMEL_K_TEST_MAKE_DIR", makeDir)

	ctx := TestContext()

	// Ensure no CRDs are already installed
	g := NewWithT(t)
	g.Expect(UninstallAll(t, ctx)).To(Succeed())

	// Return the cluster to previous state
	defer Cleanup(t, ctx)

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		namespaceArg := fmt.Sprintf("NAMESPACE=%s", ns)
		ExpectExecSucceed(t, g, Make(t, "setup-cluster", namespaceArg))
		ExpectExecSucceed(t, g, Make(t, "setup", namespaceArg, "GLOBAL=true"))

		// Skip default kamelets installation for faster test runs
		ExpectExecSucceed(t, g, Make(t, "operator",
			namespaceArg,
			"GLOBAL=true",
			"INSTALL_DEFAULT_KAMELETS=false"))

		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)

		podFunc := OperatorPod(t, ctx, ns)
		g.Eventually(podFunc).ShouldNot(BeNil())
		g.Eventually(OperatorPodPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		pod := podFunc()

		containers := pod.Spec.Containers
		g.Expect(containers).NotTo(BeEmpty())

		envvars := containers[0].Env
		g.Expect(envvars).NotTo(BeEmpty())

		found := false
		for _, v := range envvars {
			if v.Name == "WATCH_NAMESPACE" {
				g.Expect(v.Value).To(Equal("\"\""))
				found = true
				break
			}
		}
		g.Expect(found).To(BeTrue())

		g.Eventually(Platform(t, ctx, ns)).ShouldNot(BeNil())
	})
}

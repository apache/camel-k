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
	"os/exec"
	"testing"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	testutil "github.com/apache/camel-k/v2/e2e/support/util"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"

	. "github.com/onsi/gomega"
)

func TestKustomizeNamespaced(t *testing.T) {
	KAMEL_INSTALL_REGISTRY := os.Getenv("KAMEL_INSTALL_REGISTRY")
	kustomizeDir := testutil.MakeTempCopyDir(t, "../../../install")
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		g.Expect(KAMEL_INSTALL_REGISTRY).NotTo(Equal(""))
		// We must change a few values in the Kustomize config
		ExpectExecSucceed(t, g,
			exec.Command(
				"sed",
				"-i",
				fmt.Sprintf("s/namespace: .*/namespace: %s/", ns),
				fmt.Sprintf("%s/overlays/kubernetes/namespaced/kustomization.yaml", kustomizeDir),
			))
		ExpectExecSucceed(t, g, Kubectl(
			"apply",
			"-k",
			fmt.Sprintf("%s/overlays/kubernetes/namespaced", kustomizeDir),
			"--server-side",
		))
		ExpectExecSucceed(t, g,
			exec.Command(
				"sed",
				"-i",
				fmt.Sprintf("s/address: .*/address: %s/", KAMEL_INSTALL_REGISTRY),
				fmt.Sprintf("%s/overlays/platform/integration-platform.yaml", kustomizeDir),
			))
		ExpectExecSucceed(t, g, Kubectl(
			"apply",
			"-k",
			fmt.Sprintf("%s/overlays/platform", kustomizeDir),
			"-n",
			ns,
		))
		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)
		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		g.Eventually(OperatorPodPhase(t, ctx, ns)).Should(Equal(corev1.PodRunning))
		// Check if restricted security context has been applied
		operatorPod := OperatorPod(t, ctx, ns)()
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.RunAsNonRoot).To(
			Equal(kubernetes.DefaultOperatorSecurityContext().RunAsNonRoot),
		)
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.Capabilities).To(
			Equal(kubernetes.DefaultOperatorSecurityContext().Capabilities),
		)
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.SeccompProfile).To(
			Equal(kubernetes.DefaultOperatorSecurityContext().SeccompProfile),
		)
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation).To(
			Equal(kubernetes.DefaultOperatorSecurityContext().AllowPrivilegeEscalation),
		)
		g.Eventually(Platform(t, ctx, ns)).ShouldNot(BeNil())
		g.Eventually(PlatformHas(t, ctx, ns, func(pl *v1.IntegrationPlatform) bool {
			return pl.Status.Build.Registry.Address == KAMEL_INSTALL_REGISTRY
		}), TestTimeoutShort).Should(BeTrue())

		// Test a simple integration is running
		g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml")).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "yaml", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		// Test operator only uninstall
		ExpectExecSucceed(t, g, Kubectl(
			"delete",
			"deploy,configmap,secret,sa,rolebindings,clusterrolebindings,roles,clusterroles,integrationplatform",
			"-l",
			"app=camel-k",
			"-n",
			ns,
		))
		g.Eventually(OperatorPod(t, ctx, ns)).Should(BeNil())
		g.Eventually(Platform(t, ctx, ns)).Should(BeNil())
		g.Eventually(Integration(t, ctx, ns, "yaml"), TestTimeoutShort).ShouldNot(BeNil())
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "yaml", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		// Test CRD uninstall (will remove Integrations as well)
		ExpectExecSucceed(t, g, Kubectl(
			"delete",
			"crd",
			"-l",
			"app=camel-k",
			"-n",
			ns,
		))
		g.Eventually(OperatorPod(t, ctx, ns)).Should(BeNil())
		g.Eventually(Integration(t, ctx, ns, "yaml"), TestTimeoutShort).Should(BeNil())
		g.Eventually(CRDs(t)).Should(BeNil())
	})
}

func TestKustomizeDescoped(t *testing.T) {
	KAMEL_INSTALL_REGISTRY := os.Getenv("KAMEL_INSTALL_REGISTRY")
	kustomizeDir := testutil.MakeTempCopyDir(t, "../../../install")
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		g.Expect(KAMEL_INSTALL_REGISTRY).NotTo(Equal(""))
		// We must change a few values in the Kustomize config
		ExpectExecSucceed(t, g,
			exec.Command(
				"sed",
				"-i",
				fmt.Sprintf("s/namespace: .*/namespace: %s/", ns),
				fmt.Sprintf("%s/overlays/kubernetes/descoped/kustomization.yaml", kustomizeDir),
			))
		ExpectExecSucceed(t, g, Kubectl(
			"apply",
			"-k",
			fmt.Sprintf("%s/overlays/kubernetes/descoped", kustomizeDir),
			"--server-side",
		))
		ExpectExecSucceed(t, g,
			exec.Command(
				"sed",
				"-i",
				fmt.Sprintf("s/address: .*/address: %s/", KAMEL_INSTALL_REGISTRY),
				fmt.Sprintf("%s/overlays/platform/integration-platform.yaml", kustomizeDir),
			))
		ExpectExecSucceed(t, g, Kubectl(
			"apply",
			"-k",
			fmt.Sprintf("%s/overlays/platform", kustomizeDir),
			"-n",
			ns,
		))

		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)

		podFunc := OperatorPod(t, ctx, ns)
		g.Eventually(podFunc).ShouldNot(BeNil())
		g.Eventually(OperatorPodPhase(t, ctx, ns)).Should(Equal(corev1.PodRunning))
		pod := podFunc()

		containers := pod.Spec.Containers
		g.Expect(containers).NotTo(BeEmpty())

		envvars := containers[0].Env
		g.Expect(envvars).NotTo(BeEmpty())

		found := false
		for _, v := range envvars {
			if v.Name == "WATCH_NAMESPACE" {
				g.Expect(v.Value).To(Equal(""))
				found = true
				break
			}
		}
		g.Expect(found).To(BeTrue())
		// Check if restricted security context has been applied
		operatorPod := OperatorPod(t, ctx, ns)()
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.RunAsNonRoot).To(
			Equal(kubernetes.DefaultOperatorSecurityContext().RunAsNonRoot),
		)
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.Capabilities).To(
			Equal(kubernetes.DefaultOperatorSecurityContext().Capabilities),
		)
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.SeccompProfile).To(
			Equal(kubernetes.DefaultOperatorSecurityContext().SeccompProfile),
		)
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation).To(
			Equal(kubernetes.DefaultOperatorSecurityContext().AllowPrivilegeEscalation),
		)
		g.Eventually(Platform(t, ctx, ns)).ShouldNot(BeNil())

		// We need a different namespace from the global operator
		WithNewTestNamespace(t, func(ctx context.Context, g *WithT, nsIntegration string) {
			// Test a simple integration is running
			g.Expect(KamelRun(t, ctx, nsIntegration, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, nsIntegration, "yaml")).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, nsIntegration, "yaml", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, nsIntegration, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			// Test operator only uninstall
			ExpectExecSucceed(t, g, Kubectl(
				"delete",
				"deploy,configmap,secret,sa,rolebindings,clusterrolebindings,roles,clusterroles,integrationplatform",
				"-l",
				"app=camel-k",
				"-n",
				ns,
			))
			g.Eventually(OperatorPod(t, ctx, ns)).Should(BeNil())
			g.Eventually(Platform(t, ctx, ns)).Should(BeNil())
			g.Eventually(Integration(t, ctx, nsIntegration, "yaml"), TestTimeoutShort).ShouldNot(BeNil())
			g.Eventually(IntegrationConditionStatus(t, ctx, nsIntegration, "yaml", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

			// Test CRD uninstall (will remove Integrations as well)
			ExpectExecSucceed(t, g, Kubectl(
				"delete",
				"crd",
				"-l",
				"app=camel-k",
				"-n",
				ns,
			))
			g.Eventually(OperatorPod(t, ctx, ns)).Should(BeNil())
			g.Eventually(Integration(t, ctx, nsIntegration, "yaml"), TestTimeoutShort).Should(BeNil())
			g.Eventually(CRDs(t)).Should(BeNil())
		})
	})
}

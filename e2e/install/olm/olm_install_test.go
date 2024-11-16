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

package olm

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/apache/camel-k/v2/e2e/support"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

const installCatalogSourceName = "test-camel-k-source"

func TestOLMInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// Let's make sure no CRD is yet available in the cluster
		// as we must make the procedure to install them accordingly
		g.Eventually(CRDs(t)).Should(BeNil(), "No Camel K CRDs should be previously installed for this test")
		bundleImageName, ok := os.LookupEnv("BUNDLE_IMAGE_NAME")
		g.Expect(ok).To(BeTrue(), "Missing bundle image: you need to build and push to a container registry and set BUNDLE_IMAGE_NAME env var")
		containerRegistry, ok := os.LookupEnv("KAMEL_INSTALL_REGISTRY")
		g.Expect(ok).To(BeTrue(), "Missing local container registry: you need to set it into KAMEL_INSTALL_REGISTRY env var")
		os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../../")
		// Install staged bundle (it must be available by building it before running the test)
		// You can build it locally via `make bundle-push` action
		ExpectExecSucceedWithTimeout(t, g,
			Make(t,
				"bundle-test",
				fmt.Sprintf("BUNDLE_IMAGE_NAME=%s", bundleImageName),
				fmt.Sprintf("NAMESPACE=%s", ns),
			),
			"180s",
		)
		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)
		// Find the only one Camel K CSV
		noAdditionalConditions := func(csv olm.ClusterServiceVersion) bool {
			return true
		}
		g.Eventually(ClusterServiceVersionPhase(t, ctx, noAdditionalConditions, ns), TestTimeoutMedium).
			Should(Equal(olm.CSVPhaseSucceeded))
		// Check the operator pod is running
		g.Eventually(OperatorPodPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		g.Eventually(OperatorImage(t, ctx, ns), TestTimeoutShort).Should(Equal(operatorImage()))

		integrationPlatform := v1.NewIntegrationPlatform(ns, "camel-k")
		integrationPlatform.Spec.Build.Registry = v1.RegistrySpec{
			Address:  containerRegistry,
			Insecure: true,
		}
		g.Expect(CreateIntegrationPlatform(t, ctx, &integrationPlatform)).To(Succeed())
		g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
		g.Eventually(PlatformVersion(t, ctx, ns), TestTimeoutMedium).Should(Equal(defaults.Version))

		// Check if restricted security context has been applyed
		operatorPod := OperatorPod(t, ctx, ns)()
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.RunAsNonRoot).To(Equal(kubernetes.DefaultOperatorSecurityContext().RunAsNonRoot))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.Capabilities).To(Equal(kubernetes.DefaultOperatorSecurityContext().Capabilities))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.SeccompProfile).To(Equal(kubernetes.DefaultOperatorSecurityContext().SeccompProfile))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation).To(Equal(kubernetes.DefaultOperatorSecurityContext().AllowPrivilegeEscalation))

		// Test a simple integration is running
		g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "yaml", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		// Remove OLM CSV and test Integration is still existing
		csv := ClusterServiceVersion(t, ctx, noAdditionalConditions, ns)()
		g.Expect(TestClient(t).Delete(ctx, csv)).To(Succeed())
		g.Eventually(OperatorPod(t, ctx, ns)).Should(BeNil())

		g.Consistently(Integration(t, ctx, ns, "yaml"), 15*time.Second, 5*time.Second).ShouldNot(BeNil())
		g.Consistently(
			IntegrationConditionStatus(t, ctx, ns, "yaml", v1.IntegrationConditionReady), 15*time.Second, 5*time.Second).
			Should(Equal(corev1.ConditionTrue))

		// Test CRD uninstall (will remove Integrations as well)
		UninstallCRDs(t, ctx, g, "../../../")
		g.Eventually(CRDs(t)).Should(BeNil())
	})
}

func operatorImage() string {
	return envOrDefault(fmt.Sprintf("%s:%s", defaults.ImageName, defaults.Version), "KAMEL_OPERATOR_IMAGE", "KAMEL_K_TEST_OPERATOR_CURRENT_IMAGE")
}

func envOrDefault(def string, envs ...string) string {
	for i := range envs {
		if val := os.Getenv(envs[i]); val != "" {
			return val
		}
	}

	return def
}

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
		bundleImageName, ok := os.LookupEnv("BUNDLE_IMAGE_NAME")
		g.Expect(ok).To(BeTrue())
		containerRegistry, ok := os.LookupEnv("KAMEL_INSTALL_REGISTRY")
		g.Expect(ok).To(BeTrue())
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
		// Find the only one Camel K CSV
		noAdditionalConditions := func(csv olm.ClusterServiceVersion) bool {
			return true
		}
		g.Eventually(ClusterServiceVersionPhase(t, ctx, noAdditionalConditions, ns), TestTimeoutMedium).
			Should(Equal(olm.CSVPhaseSucceeded))
		// Check the operator pod is running
		g.Eventually(OperatorPodPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		g.Eventually(OperatorImage(t, ctx, ns), TestTimeoutShort).Should(Equal(defaults.OperatorImage()))

		// Check the IntegrationPlatform has been reconciled after setting the expected container registry
		g.Expect(UpdatePlatform(t, ctx, ns, func(ip *v1.IntegrationPlatform) {
			ip.Spec.Build.Registry.Address = containerRegistry
		})).To(Succeed())
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

		// TODO Remove OLM CSV and test
		// g.Eventually(OperatorPod(t, ctx, ns)).Should(BeNil())
		// g.Eventually(Platform(t, ctx, ns)).Should(BeNil())
		// g.Eventually(Integration(t, ctx, ns, "yaml"), TestTimeoutShort).ShouldNot(BeNil())
		// g.Eventually(IntegrationConditionStatus(t, ctx, ns, "yaml", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
	})
}

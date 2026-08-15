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

package helm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	. "github.com/onsi/gomega"
)

func TestHelmInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, operatorNs string) {
		// Let's make sure no CRD is yet available in the cluster
		// as we must make the procedure to install them accordingly
		g.Expect(CRDs(t)()).Should(BeNil(), "No Camel K CRDs should be previously installed for this test")

		registry := os.Getenv("KAMEL_INSTALL_REGISTRY")
		g.Expect(registry).NotTo(BeEmpty(), "KAMEL_INSTALL_REGISTRY env var must not be empty")

		operatorID := "helm-ck"
		os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../../")
		ExpectExecSucceed(t, g,
			exec.Command(
				"helm",
				"install",
				"camel-k",
				fmt.Sprintf("../../../docs/charts/camel-k-%s.tgz", defaults.Version),
				"--set", "operator.env[0].name=REGISTRY_ADDRESS",
				"--set", "operator.env[0].value="+registry,
				// We expect the testing infra to make it available a secret
				// named "my-registry" in the installation namespace
				"--set", "operator.env[1].name=REGISTRY_SECRET",
				"--set", "operator.env[1].value=my-registry",
				"--set", "operator.env[2].name=REGISTRY_INSECURE",
				"--set-string", "operator.env[2].value=true",
				"--set", fmt.Sprintf("operator.operatorId=%s", operatorID),
				"-n", operatorNs,
				"--force",
			),
		)
		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)

		g.Eventually(OperatorPod(t, ctx, operatorNs)).ShouldNot(BeNil())
		// Check if restricted security context has been applied
		operatorPod := OperatorPod(t, ctx, operatorNs)()
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.RunAsNonRoot).To(Equal(DefaultOperatorSecurityContext().RunAsNonRoot))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.Capabilities).To(Equal(DefaultOperatorSecurityContext().Capabilities))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.SeccompProfile).To(Equal(DefaultOperatorSecurityContext().SeccompProfile))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation).To(Equal(DefaultOperatorSecurityContext().AllowPrivilegeEscalation))

		// Test a simple route
		t.Run("simple route", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, operatorNs, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, operatorNs, "yaml", v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, operatorNs, "yaml")).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, operatorNs, "yaml")).Should(ContainSubstring("Magicstring!"))
		})

		ExpectExecSucceed(t, g,
			exec.Command(
				"helm",
				"uninstall",
				"camel-k",
				"-n",
				operatorNs,
			),
		)

		g.Eventually(OperatorPod(t, ctx, operatorNs)).Should(BeNil())

		// Test CRD uninstall (will remove Integrations as well)
		UninstallCRDs(t, ctx, g, "../../../")

		g.Eventually(CRDs(t)).Should(BeNil())
	})
}

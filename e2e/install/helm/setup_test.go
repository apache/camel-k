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
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	. "github.com/onsi/gomega"
)

func TestHelmInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		containerRegistry, ok := os.LookupEnv("KAMEL_INSTALL_REGISTRY")
		g.Expect(ok).To(BeTrue())
		// Let's make sure no CRD is yet available in the cluster
		// as we must make the procedure to install them accordingly
		g.Eventually(CRDs(t)).Should(BeNil(), "No Camel K CRDs should be previously installed for this test")
		operatorID := "helm-ck"
		os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../../")
		ExpectExecSucceed(t, g,
			exec.Command(
				"helm",
				"install",
				"camel-k",
				fmt.Sprintf("../../../docs/charts/camel-k-%s.tgz", defaults.Version),
				"--set",
				fmt.Sprintf("platform.build.registry.address=%s", containerRegistry),
				"--set",
				"platform.build.registry.insecure=true",
				"--set",
				fmt.Sprintf("operator.operatorId=%s", operatorID),
				"-n",
				ns,
				"--force",
			),
		)

		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		// Check if restricted security context has been applied
		operatorPod := OperatorPod(t, ctx, ns)()
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.RunAsNonRoot).To(Equal(kubernetes.DefaultOperatorSecurityContext().RunAsNonRoot))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.Capabilities).To(Equal(kubernetes.DefaultOperatorSecurityContext().Capabilities))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.SeccompProfile).To(Equal(kubernetes.DefaultOperatorSecurityContext().SeccompProfile))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation).To(Equal(kubernetes.DefaultOperatorSecurityContext().AllowPrivilegeEscalation))

		// Test a simple route
		t.Run("simple route", func(t *testing.T) {
			name := RandomizedSuffixName("yaml")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		})

		ExpectExecSucceed(t, g,
			exec.Command(
				"helm",
				"uninstall",
				"camel-k",
				"-n",
				ns,
			),
		)

		g.Eventually(OperatorPod(t, ctx, ns)).Should(BeNil())
	})
}

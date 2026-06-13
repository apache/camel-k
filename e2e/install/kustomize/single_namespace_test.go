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
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	testutil "github.com/apache/camel-k/v2/e2e/support/util"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	. "github.com/onsi/gomega"
)

func TestKustomizeSingleNamespace(t *testing.T) {
	kustomizeDir := testutil.MakeTempCopyDir(t, "../../../install")

	// The operator is expected to be installed in "operators" namespace
	// it also expects to reconcile correctly an Integration in namespace "tenant-a"
	// but it won't reconcile in any other namespaces, for example, "tenan-b"
	WithNamedTestNamespace(t, func(ctx context.Context, g *WithT, operatorNs string) {
		WithNamedTestNamespace(t, func(ctx context.Context, g *WithT, tenantNs string) {
			// Let's make sure no CRD is yet available in the cluster
			// as we must make the procedure to install them accordingly
			g.Eventually(CRDs(t)).Should(BeNil(), "No Camel K CRDs should be previously installed for this test")
			ExpectExecSucceed(t, g, Kubectl(
				"apply",
				"-k",
				fmt.Sprintf("%s/overlays/kubernetes/single-namespace", kustomizeDir),
				"--server-side",
			))
			g.Eventually(OperatorPod(t, ctx, operatorNs)).ShouldNot(BeNil())
			g.Eventually(OperatorPodPhase(t, ctx, operatorNs)).Should(Equal(corev1.PodRunning))

			WithNamedTestNamespace(t, func(ctx context.Context, g *WithT, tenantNs string) {
				// Test a simple integration in "tenant-b" is not reconciled
				g.Expect(KamelRun(t, ctx, tenantNs, "files/yaml.yaml").Execute()).To(Succeed())
				g.Consistently(IntegrationPhase(t, ctx, tenantNs, "yaml"), 10*time.Second).Should(BeEmpty())
			}, "tenant-b")

			// Test a simple integration in "tenant-a" is reconciled and runs correctly
			g.Expect(KamelRun(t, ctx, tenantNs, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, tenantNs, "yaml", v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, tenantNs, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			// Test operator only uninstall
			UninstallOperator(t, ctx, g, operatorNs, "../../../")

			g.Eventually(OperatorPod(t, ctx, operatorNs)).Should(BeNil())
			g.Eventually(Integration(t, ctx, "tenant-a", "yaml"), TestTimeoutShort).ShouldNot(BeNil())
			g.Eventually(IntegrationConditionStatus(t, ctx, "tenant-a", "yaml", v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))

			// Test CRD uninstall (will remove Integrations as well)
			UninstallCRDs(t, ctx, g, "../../../")

			g.Eventually(OperatorPod(t, ctx, operatorNs)).Should(BeNil())
			g.Eventually(CRDs(t)).Should(BeNil())
		}, "tenant-a")
	}, "operators")
}

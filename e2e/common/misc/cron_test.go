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

package misc

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestRunCronExample(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-cron"
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("cron", func(t *testing.T) {
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/cron.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationCronJob(t, ctx, ns, "cron"), TestTimeoutLong).ShouldNot(BeNil())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron", v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "cron"), TestTimeoutMedium).Should(ContainSubstring("Magicstring!"))
		})

		t.Run("cron-yaml", func(t *testing.T) {
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/cron-yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationCronJob(t, ctx, ns, "cron-yaml"), TestTimeoutLong).ShouldNot(BeNil())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-yaml", v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "cron-yaml"), TestTimeoutMedium).Should(ContainSubstring("Magicstring!"))
		})

		t.Run("cron-timer", func(t *testing.T) {
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/cron-timer.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationCronJob(t, ctx, ns, "cron-timer"), TestTimeoutLong).ShouldNot(BeNil())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-timer", v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "cron-timer"), TestTimeoutMedium).Should(ContainSubstring("Magicstring!"))
		})

		t.Run("cron-fallback", func(t *testing.T) {
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/cron-fallback.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "cron-fallback"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-fallback", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "cron-fallback"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		})

		t.Run("cron-quartz", func(t *testing.T) {
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/cron-quartz.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "cron-quartz"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-quartz", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "cron-quartz"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		})

		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

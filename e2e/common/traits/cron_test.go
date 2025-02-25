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

package common

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestRunCronExample(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {

		t.Run("cron-timer", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/cron-timer.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationCronJob(t, ctx, ns, "cron-timer"), TestTimeoutLong).ShouldNot(BeNil())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-timer", v1.IntegrationConditionReady), TestTimeoutMedium).Should(
				Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-timer", v1.IntegrationConditionCronJobAvailable),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			// As it's a cron, we expect it's triggered, executed and turned off
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, "cron-timer"), TestTimeoutMedium).Should(Equal(ptr.To(int32(1))))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "cron-timer")).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, "cron-timer")).Should(ContainSubstring("Magicstring!"))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, "cron-timer")).Should(Equal(ptr.To(int32(0))))
			g.Eventually(DeleteIntegrations(t, ctx, ns)).Should(Equal(0))
		})

		t.Run("cron-java", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/CronJava.java").Execute()).To(Succeed())
			g.Eventually(IntegrationCronJob(t, ctx, ns, "cron-java"), TestTimeoutLong).ShouldNot(BeNil())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-java", v1.IntegrationConditionReady), TestTimeoutMedium).Should(
				Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-java", v1.IntegrationConditionCronJobAvailable),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			// As it's a cron, we expect it's triggered, executed and turned off
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, "cron-java"), TestTimeoutMedium).Should(Equal(ptr.To(int32(1))))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "cron-java")).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, "cron-java")).Should(ContainSubstring("Magicstring!"))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, "cron-java")).Should(Equal(ptr.To(int32(0))))
			g.Eventually(DeleteIntegrations(t, ctx, ns)).Should(Equal(0))
		})

		t.Run("cron-tab", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/cron-tab.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationCronJob(t, ctx, ns, "cron-tab"), TestTimeoutLong).ShouldNot(BeNil())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-tab", v1.IntegrationConditionReady), TestTimeoutMedium).Should(
				Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-tab", v1.IntegrationConditionCronJobAvailable),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			// As it's a cron, we expect it's triggered, executed and turned off
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, "cron-tab"), TestTimeoutMedium).Should(Equal(ptr.To(int32(1))))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "cron-tab")).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, "cron-tab")).Should(ContainSubstring("Magicstring!"))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, "cron-tab")).Should(Equal(ptr.To(int32(0))))
			g.Eventually(DeleteIntegrations(t, ctx, ns)).Should(Equal(0))
		})

		t.Run("cron-quartz", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/cron-quartz.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationCronJob(t, ctx, ns, "cron-quartz"), TestTimeoutLong).ShouldNot(BeNil())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-quartz", v1.IntegrationConditionReady), TestTimeoutShort).Should(
				Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-quartz", v1.IntegrationConditionCronJobAvailable),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			// As it's a cron, we expect it's triggered, executed and turned off
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, "cron-quartz"), TestTimeoutMedium).Should(Equal(ptr.To(int32(1))))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "cron-quartz")).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, "cron-quartz")).Should(ContainSubstring("Magicstring!"))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, "cron-quartz")).Should(Equal(ptr.To(int32(0))))
			g.Eventually(DeleteIntegrations(t, ctx, ns)).Should(Equal(0))
		})

		t.Run("cron-fallback", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/cron-fallback.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationCronJob(t, ctx, ns, "cron-fallback"), TestTimeoutLong).Should(BeNil())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-fallback", v1.IntegrationConditionReady), TestTimeoutShort).Should(
				Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "cron-fallback", v1.IntegrationConditionCronJobAvailable),
				TestTimeoutMedium).Should(Equal(corev1.ConditionFalse))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "cron-fallback")).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, "cron-fallback"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			g.Eventually(DeleteIntegrations(t, ctx, ns)).Should(Equal(0))
		})

	})
}

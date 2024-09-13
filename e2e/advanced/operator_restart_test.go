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

package advanced

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestOperatorRestart(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		name := RandomizedSuffixName("yaml")

		t.Run("Operator started", func(t *testing.T) {
			InstallOperator(t, ctx, g, ns)
			g.Eventually(OperatorPod(t, ctx, ns)).Should(Not(BeNil()))
			g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutShort).Should(Equal(v1.IntegrationPlatformPhaseReady))
			g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			g.Eventually(Kit(t, ctx, ns, IntegrationKit(t, ctx, ns, name)())).Should(Not(BeNil()))
			g.Eventually(Integration(t, ctx, ns, name)).Should(Not(BeNil()))
		})

		t.Run("Operator uninstalled", func(t *testing.T) {
			UninstallOperator(t, ctx, g, ns, "../../")
			g.Eventually(OperatorPod(t, ctx, ns)).Should(BeNil())
			g.Eventually(Platform(t, ctx, ns)).Should(BeNil())
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			g.Eventually(Kit(t, ctx, ns, IntegrationKit(t, ctx, ns, name)())).Should(Not(BeNil()))
			g.Eventually(Integration(t, ctx, ns, name)).Should(Not(BeNil()))
		})

		t.Run("Operator reinstalled", func(t *testing.T) {
			InstallOperator(t, ctx, g, ns)
			g.Eventually(OperatorPod(t, ctx, ns)).Should(Not(BeNil()))
			g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutShort).Should(Equal(v1.IntegrationPlatformPhaseReady))
			g.Consistently(OperatorLogs(t, ctx, ns), 1*time.Minute, 3*time.Second).Should(Not(ContainSubstring("error")))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			g.Eventually(Kit(t, ctx, ns, IntegrationKit(t, ctx, ns, name)())).Should(Not(BeNil()))
			g.Eventually(Integration(t, ctx, ns, name)).Should(Not(BeNil()))
		})
	})
}

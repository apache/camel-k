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
	"fmt"
	"net"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
)

func TestKamelCLIDebug(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("debug local default port check", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Expect(portIsInUse("127.0.0.1", "5005")()).To(BeFalse())

			debugTestContext, cancel := context.WithCancel(ctx)
			defer cancelAndWait(cancel)
			go KamelWithContext(t, debugTestContext, "debug", "yaml", "-n", ns).ExecuteContext(debugTestContext)

			g.Eventually(portIsInUse("127.0.0.1", "5005"), TestTimeoutMedium, 5*time.Second).Should(BeTrue())
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
			g.Eventually(IntegrationPods(t, ctx, ns, "yaml"), TestTimeoutMedium, 5*time.Second).Should(HaveLen(0))
		})

		t.Run("debug local port check", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Expect(portIsInUse("127.0.0.1", "5006")()).To(BeFalse())

			debugTestContext, cancel := context.WithCancel(ctx)
			defer cancelAndWait(cancel)
			go KamelWithContext(t, debugTestContext, "debug", "yaml", "--port", "5006", "-n", ns).ExecuteContext(debugTestContext)

			g.Eventually(portIsInUse("127.0.0.1", "5006"), TestTimeoutMedium, 5*time.Second).Should(BeTrue())
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
			g.Eventually(IntegrationPods(t, ctx, ns, "yaml"), TestTimeoutMedium, 5*time.Second).Should(HaveLen(0))
		})

		t.Run("debug logs check", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

			debugTestContext, cancel := context.WithCancel(ctx)
			defer cancelAndWait(cancel)
			go KamelWithContext(t, debugTestContext, "debug", "yaml", "-n", ns).ExecuteContext(debugTestContext)

			g.Eventually(IntegrationLogs(t, ctx, ns, "yaml"), TestTimeoutMedium).Should(ContainSubstring("Listening for transport dt_socket at address: 5005"))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
			g.Eventually(IntegrationPods(t, ctx, ns, "yaml"), TestTimeoutMedium, 5*time.Second).Should(HaveLen(0))
		})

		t.Run("Pod config test", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

			debugTestContext, cancel := context.WithCancel(ctx)
			defer cancelAndWait(cancel)
			go KamelWithContext(t, debugTestContext, "debug", "yaml", "-n", ns).ExecuteContext(debugTestContext)

			g.Eventually(func() string {
				return IntegrationPod(t, ctx, ns, "yaml")().Spec.Containers[0].Args[0]
			}).Should(ContainSubstring("-agentlib:jdwp=transport=dt_socket,server=y,suspend=y,address=*:5005"))
			g.Expect(IntegrationPod(t, ctx, ns, "yaml")().GetLabels()["camel.apache.org/debug"]).To(Not(BeNil()))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
			g.Eventually(IntegrationPods(t, ctx, ns, "yaml"), TestTimeoutMedium, 5*time.Second).Should(HaveLen(0))
		})

		g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		g.Eventually(IntegrationPods(t, ctx, ns, "yaml"), TestTimeoutMedium, 5*time.Second).Should(HaveLen(0))
	})
}

func portIsInUse(host string, port string) func() bool {
	return func() bool {

		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), time.Second*5)

		if conn != nil {
			defer conn.Close()
		}

		return conn != nil && err == nil
	}
}

func cancelAndWait(cancel context.CancelFunc) {
	cancel()
	time.Sleep(TestTimeoutShort)
}

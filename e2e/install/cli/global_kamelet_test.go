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

package cli

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
)

func TestRunGlobalKamelet(t *testing.T) {
	WithGlobalOperatorNamespace(t, func(g *WithT, operatorNamespace string) {
		operatorID := "camel-k-global-kamelet"
		g.Expect(KamelInstallWithID(t, operatorID, operatorNamespace, "--global", "--force").Execute()).To(Succeed())

		t.Run("Global operator + namespaced kamelet test", func(t *testing.T) {

			// NS2: namespace without operator
			WithNewTestNamespace(t, func(g *WithT, ns2 string) {
				g.Expect(CreateTimerKamelet(t, operatorID, ns2, "my-own-timer-source")()).To(Succeed())

				g.Expect(KamelInstallWithID(t, operatorID, ns2, "--skip-operator-setup", "--olm=false").Execute()).To(Succeed())

				g.Expect(KamelRunWithID(t, operatorID, ns2, "files/timer-kamelet-usage.groovy").Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ns2, "timer-kamelet-usage"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ns2, "timer-kamelet-usage"), TestTimeoutShort).Should(ContainSubstring("Hello world"))
				g.Expect(Kamel(t, "delete", "--all", "-n", ns2).Execute()).To(Succeed())
			})
		})

		t.Run("Global operator + global kamelet test", func(t *testing.T) {

			g.Expect(CreateTimerKamelet(t, operatorID, operatorNamespace, "my-own-timer-source")()).To(Succeed())

			// NS3: namespace without operator
			WithNewTestNamespace(t, func(g *WithT, ns3 string) {
				g.Expect(KamelInstallWithID(t, operatorID, ns3, "--skip-operator-setup", "--olm=false").Execute()).To(Succeed())

				g.Expect(KamelRunWithID(t, operatorID, ns3, "files/timer-kamelet-usage.groovy").Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ns3, "timer-kamelet-usage"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ns3, "timer-kamelet-usage"), TestTimeoutShort).Should(ContainSubstring("Hello world"))
				g.Expect(Kamel(t, "delete", "--all", "-n", ns3).Execute()).To(Succeed())
				g.Expect(TestClient(t).Delete(TestContext, Kamelet(t, "my-own-timer-source", operatorNamespace)())).To(Succeed())
			})
		})

		g.Expect(Kamel(t, "uninstall", "-n", operatorNamespace, "--skip-crd", "--skip-cluster-roles=false", "--skip-cluster-role-bindings=false").Execute()).To(Succeed())
	})

}

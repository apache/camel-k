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
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestKamelReset(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := "camel-k-cli-reset"
		g.Expect(CopyCamelCatalog(t, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, operatorID, ns)).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("Reset the whole platform", func(t *testing.T) {
			name := RandomizedSuffixName("yaml1")
			g.Expect(KamelRunWithID(t, operatorID, ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			g.Eventually(Kit(t, ns, IntegrationKit(t, ns, name)())).Should(Not(BeNil()))
			g.Eventually(Integration(t, ns, name)).Should(Not(BeNil()))

			g.Expect(Kamel(t, "reset", "-n", ns).Execute()).To(Succeed())

			g.Expect(Integration(t, ns, name)()).To(BeNil())
			g.Expect(Kits(t, ns)()).To(HaveLen(0))
		})

		t.Run("Reset skip-integrations", func(t *testing.T) {
			name := RandomizedSuffixName("yaml2")
			g.Expect(KamelRunWithID(t, operatorID, ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			g.Eventually(Kit(t, ns, IntegrationKit(t, ns, name)())).Should(Not(BeNil()))
			g.Eventually(Integration(t, ns, name)).Should(Not(BeNil()))

			g.Expect(Kamel(t, "reset", "-n", ns, "--skip-integrations").Execute()).To(Succeed())

			g.Expect(Integration(t, ns, name)()).To(Not(BeNil()))
			g.Expect(Kits(t, ns)()).To(HaveLen(0))
		})

		t.Run("Reset skip-kits", func(t *testing.T) {
			name := RandomizedSuffixName("yaml3")
			g.Expect(KamelRunWithID(t, operatorID, ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			kitName := IntegrationKit(t, ns, name)()
			g.Eventually(Kit(t, ns, kitName)).Should(Not(BeNil()))
			g.Eventually(Integration(t, ns, name)).Should(Not(BeNil()))

			g.Expect(Kamel(t, "reset", "-n", ns, "--skip-kits").Execute()).To(Succeed())

			g.Expect(Integration(t, ns, name)()).To(BeNil())
			g.Expect(Kit(t, ns, kitName)()).To(Not(BeNil()))
		})
		// Clean up
		g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

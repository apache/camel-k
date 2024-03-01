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

	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-cli-reset"
		Expect(CopyCamelCatalog(t, ns, operatorID)).To(Succeed())
		Expect(CopyIntegrationKits(t, ns, operatorID)).To(Succeed())
		Expect(KamelInstallWithID(t, operatorID, ns).Execute()).To(Succeed())

		Eventually(SelectedPlatformPhase(t, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("Reset the whole platform", func(t *testing.T) {
			name := RandomizedSuffixName("yaml1")
			Expect(KamelRunWithID(t, operatorID, ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			Eventually(Kit(t, ns, IntegrationKit(t, ns, name)())).Should(Not(BeNil()))
			Eventually(Integration(t, ns, name)).Should(Not(BeNil()))

			Expect(Kamel(t, "reset", "-n", ns).Execute()).To(Succeed())

			Expect(Integration(t, ns, name)()).To(BeNil())
			Expect(Kits(t, ns)()).To(HaveLen(0))
		})

		t.Run("Reset skip-integrations", func(t *testing.T) {
			name := RandomizedSuffixName("yaml2")
			Expect(KamelRunWithID(t, operatorID, ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			Eventually(Kit(t, ns, IntegrationKit(t, ns, name)())).Should(Not(BeNil()))
			Eventually(Integration(t, ns, name)).Should(Not(BeNil()))

			Expect(Kamel(t, "reset", "-n", ns, "--skip-integrations").Execute()).To(Succeed())

			Expect(Integration(t, ns, name)()).To(Not(BeNil()))
			Expect(Kits(t, ns)()).To(HaveLen(0))
		})

		t.Run("Reset skip-kits", func(t *testing.T) {
			name := RandomizedSuffixName("yaml3")
			Expect(KamelRunWithID(t, operatorID, ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			kitName := IntegrationKit(t, ns, name)()
			Eventually(Kit(t, ns, kitName)).Should(Not(BeNil()))
			Eventually(Integration(t, ns, name)).Should(Not(BeNil()))

			Expect(Kamel(t, "reset", "-n", ns, "--skip-kits").Execute()).To(Succeed())

			Expect(Integration(t, ns, name)()).To(BeNil())
			Expect(Kit(t, ns, kitName)()).To(Not(BeNil()))
		})
		// Clean up
		Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

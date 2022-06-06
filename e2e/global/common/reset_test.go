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
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
)

func TestKamelReset(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(KamelInstall(ns).Execute()).To(Succeed())

		t.Run("Reset the whole platform", func(t *testing.T) {

			name := "yaml1"
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			Eventually(Kit(ns, IntegrationKit(ns, name)())).Should(Not(BeNil()))
			Eventually(Integration(ns, name)).Should(Not(BeNil()))

			Expect(Kamel("reset", "-n", ns).Execute()).To(Succeed())

			Expect(Integration(ns, name)()).To(BeNil())
			Expect(Kits(ns)()).To(HaveLen(0))

		})

		t.Run("Reset skip-integrations", func(t *testing.T) {

			name := "yaml2"
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			Eventually(Kit(ns, IntegrationKit(ns, name)())).Should(Not(BeNil()))
			Eventually(Integration(ns, name)).Should(Not(BeNil()))

			Expect(Kamel("reset", "-n", ns, "--skip-integrations").Execute()).To(Succeed())

			Expect(Integration(ns, name)()).To(Not(BeNil()))
			Expect(Kits(ns)()).To(HaveLen(0))

		})

		t.Run("Reset skip-kits", func(t *testing.T) {

			name := "yaml3"
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			kitName := IntegrationKit(ns, name)()
			Eventually(Kit(ns, kitName)).Should(Not(BeNil()))
			Eventually(Integration(ns, name)).Should(Not(BeNil()))

			Expect(Kamel("reset", "-n", ns, "--skip-kits").Execute()).To(Succeed())

			Expect(Integration(ns, name)()).To(BeNil())
			Expect(Kit(ns, kitName)()).To(Not(BeNil()))

		})
		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

//go:build integration && high_memory
// +build integration,high_memory

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

package native

import (
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestNativeHighMemoryIntegrations(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-quarkus-high-memory-native"
		Expect(KamelInstallWithID(operatorID, ns,
			"--build-timeout", "90m0s",
			"--maven-cli-option", "-Dquarkus.native.native-image-xmx=9g",
		).Execute()).To(Succeed())
		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("java native support", func(t *testing.T) {
			name := "java-native"
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name,
				"-t", "quarkus.build-mode=native",
				"-t", "builder.tasks-limit-memory=quarkus-native:9.5Gi",
			).Execute()).To(Succeed())

			Eventually(IntegrationPodPhase(ns, name), TestTimeoutVeryLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationPod(ns, name), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Java Magicstring!"))

			t.Run("java native same should not rebuild", func(t *testing.T) {
				name := "java-native-clone"
				Expect(KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name,
					"-t", "quarkus.build-mode=native",
					"-t", "builder.tasks-limit-memory=quarkus-native:9.5Gi",
				).Execute()).To(Succeed())

				// This one should run quickly as it suppose to reuse an IntegrationKit
				Eventually(IntegrationPodPhase(ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationPod(ns, name), TestTimeoutShort).
					Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
				Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
					Should(Equal(corev1.ConditionTrue))
				Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Java Magicstring!"))
				Eventually(IntegrationKit(ns, "java-native-clone")).Should(Equal(IntegrationKit(ns, "java-native")()))
			})

			t.Run("java native should rebuild", func(t *testing.T) {
				name := "java-native-2"
				Expect(KamelRunWithID(operatorID, ns, "files/Java2.java", "--name", name,
					"-t", "quarkus.build-mode=native",
					"-t", "builder.tasks-limit-memory=quarkus-native:9.5Gi",
				).Execute()).To(Succeed())

				Eventually(IntegrationPodPhase(ns, name), TestTimeoutVeryLong).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationPod(ns, name), TestTimeoutShort).
					Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
				Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
					Should(Equal(corev1.ConditionTrue))
				Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Java Magic2string!"))
				Eventually(IntegrationKit(ns, "java-native-2")).ShouldNot(Equal(IntegrationKit(ns, "java-native")()))
			})

			// Clean up
			Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		})

		t.Run("groovy native support", func(t *testing.T) {
			name := "groovy-native"
			Expect(KamelRunWithID(operatorID, ns, "files/Groovy.groovy", "--name", name,
				"-t", "quarkus.build-mode=native",
				"-t", "builder.tasks-limit-memory=quarkus-native:9.5Gi",
			).Execute()).To(Succeed())

			Eventually(IntegrationPodPhase(ns, name), TestTimeoutVeryLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationPod(ns, name), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))

			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Groovy Magicstring!"))

			// Clean up
			Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		})

		t.Run("kotlin native support", func(t *testing.T) {
			name := "kotlin-native"
			Expect(KamelRunWithID(operatorID, ns, "files/Kotlin.kts", "--name", name,
				"-t", "quarkus.build-mode=native",
				"-t", "builder.tasks-limit-memory=quarkus-native:9.5Gi",
			).Execute()).To(Succeed())

			Eventually(IntegrationPodPhase(ns, name), TestTimeoutVeryLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationPod(ns, name), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))

			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Kotlin Magicstring!"))

			// Clean up
			Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		})
	})
}

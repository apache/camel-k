//go:build integration && !high_memory
// +build integration,!high_memory

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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestNativeIntegrations(t *testing.T) {
	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := "camel-k-quarkus-native"
		g.Expect(KamelInstallWithID(t, operatorID, ns,
			"--build-timeout", "90m0s",
			"--maven-cli-option", "-Dquarkus.native.native-image-xmx=6g",
		).Execute()).To(Succeed())
		g.Eventually(PlatformPhase(t, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("unsupported integration source language", func(t *testing.T) {
			name := RandomizedSuffixName("unsupported-js")
			g.Expect(KamelRunWithID(t, operatorID, ns, "files/JavaScript.js", "--name", name,
				"-t", "quarkus.build-mode=native",
				"-t", "builder.tasks-limit-memory=quarkus-native:6.5Gi",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPhase(t, ns, name)).Should(Equal(v1.IntegrationPhaseError))
			g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionKitAvailable)).
				Should(Equal(corev1.ConditionFalse))

			// check integration schema does not contains unwanted default trait value.
			g.Eventually(UnstructuredIntegration(t, ns, name)).ShouldNot(BeNil())
			unstructuredIntegration := UnstructuredIntegration(t, ns, name)()
			quarkusTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "quarkus")
			g.Expect(quarkusTrait).ToNot(BeNil())
			g.Expect(len(quarkusTrait)).To(Equal(1))
			g.Expect(quarkusTrait["buildMode"]).ToNot(BeNil())

			// Clean up
			g.Expect(Kamel(t, "delete", name, "-n", ns).Execute()).To(Succeed())
		})

		t.Run("xml native support", func(t *testing.T) {
			name := RandomizedSuffixName("xml-native")
			g.Expect(KamelRunWithID(t, operatorID, ns, "files/Xml.xml", "--name", name,
				"-t", "quarkus.build-mode=native",
				"-t", "builder.tasks-limit-memory=quarkus-native:6.5Gi",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutVeryLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPod(t, ns, name), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
			g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))

			g.Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("XML Magicstring!"))

			// Clean up
			g.Expect(Kamel(t, "delete", name, "-n", ns).Execute()).To(Succeed())
		})

		t.Run("automatic rollout deployment from jvm to native kit", func(t *testing.T) {
			// Let's make sure we start from a clean state
			g.Expect(DeleteKits(t, ns)).To(Succeed())
			name := RandomizedSuffixName("yaml-native")
			g.Expect(KamelRunWithID(t, operatorID, ns, "files/yaml.yaml", "--name", name,
				"-t", "quarkus.build-mode=jvm",
				"-t", "quarkus.build-mode=native",
				"-t", "builder.tasks-limit-memory=quarkus-native:6.5Gi",
			).Execute()).To(Succeed())

			// Check that two Kits are created with distinct layout
			g.Eventually(Kits(t, ns, withFastJarLayout)).Should(HaveLen(1))
			g.Eventually(Kits(t, ns, withNativeLayout)).Should(HaveLen(1))

			// Check the fast-jar Kit is ready
			g.Eventually(Kits(t, ns, withFastJarLayout, KitWithPhase(v1.IntegrationKitPhaseReady)),
				TestTimeoutLong).Should(HaveLen(1))

			fastJarKit := Kits(t, ns, withFastJarLayout, KitWithPhase(v1.IntegrationKitPhaseReady))()[0]
			// Check the Integration uses the fast-jar Kit
			g.Eventually(IntegrationKit(t, ns, name), TestTimeoutShort).Should(Equal(fastJarKit.Name))
			// Check the Integration Pod uses the fast-jar Kit
			g.Eventually(IntegrationPodImage(t, ns, name)).Should(Equal(fastJarKit.Status.Image))

			// Check the Integration is ready
			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPod(t, ns, name), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(), ContainSubstring("java")))
			g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))

			g.Eventually(IntegrationLogs(t, ns, name), TestTimeoutMedium).Should(ContainSubstring("Magicstring!"))

			// ====================================
			// !!! THE MOST TIME-CONSUMING PART !!!
			// ====================================
			// Check the native Kit is ready
			g.Eventually(Kits(t, ns, withNativeLayout, KitWithPhase(v1.IntegrationKitPhaseReady)),
				TestTimeoutVeryLong).Should(HaveLen(1))

			nativeKit := Kits(t, ns, withNativeLayout, KitWithPhase(v1.IntegrationKitPhaseReady))()[0]
			// Check the Integration uses the native Kit
			g.Eventually(IntegrationKit(t, ns, name), TestTimeoutShort).Should(Equal(nativeKit.Name))
			// Check the Integration Pod uses the native Kit
			g.Eventually(IntegrationPodImage(t, ns, name)).Should(Equal(nativeKit.Status.Image))

			// Check the Integration is still ready
			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutVeryLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPod(t, ns, name), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
			g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))

			g.Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			t.Run("yaml native should not rebuild", func(t *testing.T) {
				name := RandomizedSuffixName("yaml-native-2")
				g.Expect(KamelRunWithID(t, operatorID, ns, "files/yaml2.yaml", "--name", name,
					"-t", "quarkus.build-mode=native",
					"-t", "builder.tasks-limit-memory=quarkus-native:6.5Gi",
				).Execute()).To(Succeed())

				// This one should run quickly as it suppose to reuse an IntegrationKit
				g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationPod(t, ns, name), TestTimeoutShort).
					Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
				g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
					Should(Equal(corev1.ConditionTrue))
				g.Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!2"))
				g.Eventually(IntegrationKit(t, ns, "yaml-native-2")).Should(Equal(IntegrationKit(t, ns, "yaml-native")()))
			})

			// Clean up
			g.Expect(Kamel(t, "delete", name, "-n", ns).Execute()).To(Succeed())
		})

	})
}
